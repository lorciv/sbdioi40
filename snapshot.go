package sbdioi40

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

// Snapshot is a snapshot of a SBDIOI40 application.
type Snapshot struct {
	App       *Application
	CreatedAt time.Time
	Items     []ServiceSnapshot
}

// Available checks whether the given shapshot is available in the local storage. Only available
// snapshots may be restored.
func (s *Snapshot) Available() bool {
	for _, item := range s.Items {
		if item.Path == "" {
			return false
		}
	}
	return true
}

// Remove removes the given shapshot from the local storage. Once a snapshot is removed, it can
// no longer be restored. Any attempt to restore a removed snapshot will fail.
func (s *Snapshot) Remove() error {
	for _, item := range s.Items {
		if err := os.Remove(item.Path); err != nil {
			return err
		}
	}
	log.Printf("%s snapshot removed from local storage", s.App.Name)
	return nil
}

func (s *Snapshot) String() string {
	timefmt := "2006-01-02 15:04:05.00000"
	return fmt.Sprintf("snapshot of app %s (%s)", s.App.Name, s.CreatedAt.Format(timefmt))
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
func (p *Platform) Snapshot(appname string) (*Snapshot, error) {
	app, err := p.Application(appname)
	if err != nil {
		return nil, fmt.Errorf("snapshot failed: %v", err)
	}
	log.Printf("found %s", app)

	snap := &Snapshot{
		App:       &app,
		CreatedAt: time.Now(),
	}

	for _, serv := range app.Services {
		serv := serv // necessary: capture iteration variable

		// TODO: snapshot and download each service concurrently

		imageID, err := servers.CreateImage(p.nova, serv.server.ID, servers.CreateImageOpts{
			Name: app.Name + serv.Name + "snap",
		}).ExtractImageID()
		if err != nil {
			return nil, err
		}
		// TODO: try to call image create with "wait" option instead
		if err := p.waitForImage(imageID); err != nil {
			return nil, err
		}

		image, err := images.Get(p.glance, imageID).Extract()
		if err != nil {
			return nil, err
		}

		reader, err := imagedata.Download(p.glance, imageID).Extract()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		f, err := ioutil.TempFile("", "sbdioi40-"+image.Name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		written, err := io.Copy(f, reader)
		if err != nil {
			return nil, err
		}

		log.Printf("service snapshot of %s completed (%d bytes)", serv.Name, written)

		if err := images.Delete(p.glance, imageID).ExtractErr(); err != nil {
			log.Printf("image delete failed: %v; snapshot is still on the source platform", err)
		}

		snap.Items = append(snap.Items, ServiceSnapshot{
			Service: &serv,
			Path:    f.Name(),
			image:   *image,
		})
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
