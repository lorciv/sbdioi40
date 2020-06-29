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

// Snapshot is a snapshot of a SBDIOI40 application.
type Snapshot struct {
	App       *Application
	Dir       string // TODO: may be no longer necessary
	CreatedAt time.Time
	Items     []ServiceSnapshot
}

func (s Snapshot) String() string {
	timefmt := "2006-01-02 15:04:05.00000"
	return fmt.Sprintf("snapshot of app %s (%s) in %s", s.App.Name, s.CreatedAt.Format(timefmt), s.Dir)
}

// ServiceSnapshot is a snapshot of a single service belonging to an application.
// It corresponds to a raw file in the local storage.
type ServiceSnapshot struct {
	Service *Service
	Path    string
	image   images.Image
}

// Snapshot creates a snapshot of the named application and returns an object
// that holds information about it. The data is downloaded from the platform
// and stored locally.
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

		// TODO: snapshot and download each service concurrently

		imageID, err := servers.CreateImage(p.nova, serv.server.ID, servers.CreateImageOpts{
			Name: app.Name + serv.Name + "snap",
		}).ExtractImageID()
		if err != nil {
			return Snapshot{}, err
		}
		// TODO: try to call image create with "wait" option instead
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

		snap.Items = append(snap.Items, ServiceSnapshot{
			Service: &serv,
			Path:    f.Name(),
			image:   *image,
		})

		// TODO: remove the service snapshot image from the OpenStack platform

		log.Printf("service snapshot of %s completed (%d bytes)", serv.Name, written)
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

		log.Printf("image %s not ready; waiting...", image.Name)
		time.Sleep(time.Second << uint(tries)) // exponential back-off
	}

	return fmt.Errorf("image %s not ready after %s; giving up", imageID, timeout)
}
