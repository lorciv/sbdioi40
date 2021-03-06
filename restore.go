package sbdioi40

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	flavorutils "github.com/gophercloud/utils/openstack/compute/v2/flavors"
)

// Restore uploads the given snapshot of an application to the platform and restores
// the application.
//
// TODO: Restore expects to find the m1.tiny flavor and the default security group.
// It should recreate them as well, if they are missing.
func (p *Platform) Restore(snap *Snapshot) error {
	if !snap.Available() {
		return errors.New("snapshot is not available")
	}
	// recreate the network and the subnet, and connect it to the router
	network, err := networks.Create(p.neutron, networks.CreateOpts{
		Name: snap.App.network.Name,
	}).Extract()
	if err != nil {
		return fmt.Errorf("restoring network for %s: %v", snap.App.Name, err)
	}

	subnet, err := subnets.Create(p.neutron, subnets.CreateOpts{
		Name:           snap.App.Name + "subnet",
		NetworkID:      network.ID,
		IPVersion:      4, // TODO: add support for any IP version
		CIDR:           snap.App.Network(),
		DNSNameservers: snap.App.DNSServers(),
	}).Extract()
	if err != nil {
		return fmt.Errorf("restoring subnet for %s: %v", snap.App.Name, err)
	}

	_, err = routers.AddInterface(p.neutron, p.router.ID, routers.AddInterfaceOpts{
		SubnetID: subnet.ID,
	}).Extract()
	if err != nil {
		return fmt.Errorf("connecting network to router for %s: %v", snap.App.Name, err)
	}

	for _, snapItem := range snap.Items {
		// upload the snapshot image
		f, err := os.Open(snapItem.Path)
		if err != nil {
			return fmt.Errorf("cannot restore %s: %v", snap.App.Name, err)
		}
		defer f.Close()
		image, err := images.Create(p.glance, images.CreateOpts{
			Name:            snap.App.Name + snapItem.Service.Name + "snap",
			DiskFormat:      snapItem.image.DiskFormat,
			ContainerFormat: snapItem.image.ContainerFormat,
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

		port, err := ports.Create(p.neutron, ports.CreateOpts{
			Name:      snapItem.Service.port.Name,
			NetworkID: network.ID,
			FixedIPs: []ports.IP{
				{SubnetID: subnet.ID, IPAddress: snapItem.Service.IPAddr()},
			},
		}).Extract()
		if err != nil {
			return fmt.Errorf("restoring port for service %s: %v", snapItem.Service.Name, err)
		}

		server, err := servers.Create(p.nova, servers.CreateOpts{
			Name:     snap.App.Name + snapItem.Service.Name + "vm",
			ImageRef: image.ID,
			Networks: []servers.Network{
				{Port: port.ID},
			},
			FlavorRef:      flavorID,
			SecurityGroups: []string{"default"},
		}).Extract()
		if err != nil {
			return fmt.Errorf("restoring %s: %v", snap.App.Name, err)
		}

		// wait for server start-up before removing its image
		if err := p.waitForServer(server.ID); err != nil {
			log.Printf("%v; won't remove snapshot from the platform", err)
		}

		log.Printf("restored service %s", snapItem.Service.Name)

		if err := images.Delete(p.glance, image.ID).ExtractErr(); err != nil {
			log.Printf("snapshot delete failed: %v; still on the platform", err)
		}
	}

	return nil
}

func (p *Platform) waitForServer(serverID string) error {
	var serverName string

	const timeout = 1 * time.Minute
	deadline := time.Now().Add(timeout)

	for tries := 0; time.Now().Before(deadline); tries++ {
		server, err := servers.Get(p.nova, serverID).Extract()
		if err != nil {
			return err
		}
		if serverName == "" {
			serverName = server.Name
		}

		if server.Status == "ACTIVE" {
			return nil // success
		}

		log.Printf("server %s not active yet; waiting...", serverName)
		time.Sleep(time.Second << uint(tries)) // exponential back-off
	}

	return fmt.Errorf("server %s not active after %s; giving up", serverName, timeout)
}
