package sbdioi40

import (
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

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
		imageID, err := servers.CreateImage(nova, serv.serverID, servers.CreateImageOpts{
			Name: app.Name + serv.Name + "snap",
		}).ExtractImageID()
		if err != nil {
			return Snapshot{}, err
		}

		snap.images = append(snap.images, serviceSnapshot{
			service: &serv,
			imageID: imageID},
		)
	}

	return snap, nil
}
