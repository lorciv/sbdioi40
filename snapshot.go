package sbdioi40

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
)

// Snapshot is a remote snapshot of a sbdioi40 application.
type Snapshot struct {
	App       *Application
	images    []serviceSnapshot
	CreatedAt time.Time
}

func (s Snapshot) String() string {
	return fmt.Sprintf("snapshot of %s (%s)", s.App.Name, s.CreatedAt.Format("2006-01-02 15:04:05.00000"))
}

type serviceSnapshot struct {
	service *Service
	imageID string
}

// Snapshot creates a snapshot of the given application and returns a Snapshot
// object that holds information about it.
func (p *Platform) Snapshot(app *Application) (Snapshot, error) {
	nova, err := openstack.NewComputeV2(p.client, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{
		App:       app,
		CreatedAt: time.Now(),
	}

	for _, serv := range app.Services {
		serv := serv // necessary: capture iteration variable

		imageID, err := servers.CreateImage(nova, serv.serverID, servers.CreateImageOpts{
			Name: app.Name + serv.Name + "snap",
		}).ExtractImageID()
		if err != nil {
			return Snapshot{}, err
		}
		log.Printf("snapshot %s done", serv.Name)

		snap.images = append(snap.images, serviceSnapshot{
			service: &serv,
			imageID: imageID,
		})
	}

	return snap, nil
}

// Download downlads a snapshot to the local storage. It returns the location of
// the temporary directory where the data can be found.
func (p *Platform) Download(snap Snapshot) (string, error) {
	glance, err := openstack.NewImageServiceV2(p.client, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return "", err
	}

	dir, err := ioutil.TempDir("", "sbdioi40-snap-"+snap.App.Name)
	if err != nil {
		return "", err
	}

	for _, img := range snap.images {
		if err := p.waitForImage(img.imageID); err != nil {
			return "", err
		}

		reader, err := imagedata.Download(glance, img.imageID).Extract()
		if err != nil {
			return "", err
		}
		defer reader.Close()
		f, err := os.Create(filepath.Join(dir, img.service.Name))
		if err != nil {
			return "", err
		}
		written, err := io.Copy(f, reader)
		if err != nil {
			return "", err
		}

		log.Printf("snapshot of %s completed (%d bytes)", img.service.Name, written)
	}

	return dir, nil
}

func (p *Platform) waitForImage(imageID string) error {
	glance, err := openstack.NewImageServiceV2(p.client, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return err
	}

	const timeout = 1 * time.Minute
	deadline := time.Now().Add(timeout)
	for tries := 0; time.Now().Before(deadline); tries++ {
		image, err := images.Get(glance, imageID).Extract()
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
