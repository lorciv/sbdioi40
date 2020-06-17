package sbdioi40

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

// Snapshot is a snapshot of a sbdioi40 application.
type Snapshot struct {
	App       *Application
	Dir       string // TODO: remove
	CreatedAt time.Time
	Files     []SnapshotFile
}

func (s Snapshot) String() string {
	timefmt := "2006-01-02 15:04:05.00000"
	return fmt.Sprintf("snapshot of %s (%s) in %s", s.App.Name, s.CreatedAt.Format(timefmt), s.Dir)
}

// SnapshotFile is a snapshot of a single service belonging to a sbdioi40 application.
type SnapshotFile struct {
	Service         *Service
	Path            string
	DiskFormat      string
	ContainerFormat string
}

// Snapshot creates a snapshot of the given application, downloads it to the
// local storage and returns a Snapshot object that holds information about it.
func (p *Platform) Snapshot(appname string) (Snapshot, error) {
	app, err := p.Application(appname)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot failed: %v", err)
	}

	dir, err := ioutil.TempDir("", "sbdioi40-snap-"+appname)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{
		App:       &app,
		Dir:       dir,
		CreatedAt: time.Now(),
	}

	for _, serv := range app.Services {
		serv := serv // necessary: capture iteration variable

		// TODO: snapshot and download in parallel

		imageID, err := servers.CreateImage(p.nova, serv.serverID, servers.CreateImageOpts{
			Name: app.Name + serv.Name + "snap",
		}).ExtractImageID()
		if err != nil {
			return Snapshot{}, err
		}
		// TODO: try to call image create with "wait" option
		if err := p.waitForImage(imageID); err != nil {
			return Snapshot{}, err
		}

		image, err := images.Get(p.glance, imageID).Extract()
		if err != nil {
			return Snapshot{}, err
		}

		reader, err := imagedata.Download(p.glance, imageID).Extract()
		if err != nil {
			return Snapshot{}, err
		}
		defer reader.Close()
		f, err := os.Create(filepath.Join(dir, serv.Name+".raw"))
		if err != nil {
			return Snapshot{}, err
		}
		defer f.Close()
		written, err := io.Copy(f, reader)
		if err != nil {
			return Snapshot{}, err
		}

		snap.Files = append(snap.Files, SnapshotFile{
			Service:         &serv,
			Path:            f.Name(),
			DiskFormat:      image.DiskFormat,
			ContainerFormat: image.ContainerFormat,
		})

		// TODO: remove the snapshot image from the OpenStack platform

		log.Printf("snapshot of %s completed (%d bytes)", serv.Name, written)
	}

	return snap, nil
}

func (p *Platform) waitForImage(imageID string) error {
	const timeout = 1 * time.Minute
	deadline := time.Now().Add(timeout)
	for tries := 0; time.Now().Before(deadline); tries++ {
		image, err := images.Get(p.glance, imageID).Extract()
		if err != nil {
			return err
		}
		if image.Status == images.ImageStatusActive {
			return nil // success
		}

		log.Printf("image %s not ready; retrying...", image.Name)
		time.Sleep(time.Second << uint(tries))
	}

	return fmt.Errorf("image %s not ready after %s; giving up", imageID, timeout)
}
