package sbdioi40

import (
	"fmt"
	"log"
	"os"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	flavorutils "github.com/gophercloud/utils/openstack/compute/v2/flavors"
	"github.com/gophercloud/utils/openstack/networking/v2/ports"
)

// Restore uploads the given snapshot of an application to the platform and restores
// the application.
//
// TODO: Restore assumes that the application's network and ports on the destination
// platform are already set up. Instead, they should always be recreated (the presence
// or absence of a network correspond to the presence or absence of the application).
// Also, Restore expects to find the m1.tiny flavor and the default security group.
// I should recreate them as well, if they are missing.
func (p *Platform) Restore(snap Snapshot) error {
	for _, snapItem := range snap.Items {
		// upload the snapshot image
		f, err := os.Open(snapItem.Path)
		if err != nil {
			return fmt.Errorf("cannot restore %s: %v", snap.App.Name, err)
		}
		defer f.Close()

		image, err := images.Create(p.glance, images.CreateOpts{
			Name:            snap.App.Name + snapItem.Service.Name + "snap",
			DiskFormat:      snapItem.DiskFormat,
			ContainerFormat: snapItem.ContainerFormat,
		}).Extract()
		if err != nil {
			// TODO: remove already created images from destination platform
			return fmt.Errorf("cannot restore %s: %v", snap.App.Name, err)
		}
		if err := imagedata.Upload(p.glance, image.ID, f).ExtractErr(); err != nil {
			// TODO: remove already created images from destination platform
			return fmt.Errorf("restoring %s: %v", snap.App.Name, err)
		}
		log.Printf("uploaded service snapshot for %s", snapItem.Service.Name)

		// TODO: delete snapshot item from local storage (and finally the snapshot dir)

		// create and launch the service
		flavorID, err := flavorutils.IDFromName(p.nova, "m1.tiny")
		if err != nil {
			return fmt.Errorf("restoring %s: %v", snap.App.Name, err)
		}
		portID, err := ports.IDFromName(p.neutron, snap.App.Name+snapItem.Service.Name+"port")
		if err != nil {
			return fmt.Errorf("restoring %s: %v", snap.App.Name, err)
		}
		_, err = servers.Create(p.nova, servers.CreateOpts{
			Name:     snap.App.Name + snapItem.Service.Name + "vm",
			ImageRef: image.ID,
			Networks: []servers.Network{
				{Port: portID},
			},
			FlavorRef:      flavorID,
			SecurityGroups: []string{"default"},
		}).Extract()
		if err != nil {
			return fmt.Errorf("restoring %s: %v", snap.App.Name, err)
		}
		log.Printf("restored service %s", snapItem.Service.Name)
	}

	return nil
}
