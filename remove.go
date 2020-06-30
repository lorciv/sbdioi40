package sbdioi40

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

// Remove removes an application from the given platform. If the application does not exist,
// or a problem occurs, it returns an error.
func (p *Platform) Remove(appname string) error {
	app, err := p.Application(appname)
	if err != nil {
		return err
	}

	_, err = routers.RemoveInterface(p.neutron, p.router.ID, routers.RemoveInterfaceOpts{
		SubnetID: app.subnet.ID,
	}).Extract()
	if err != nil {
		return fmt.Errorf("cannot remove router connection for %s: %v", appname, err)
	}

	for _, serv := range app.Services {
		if err := servers.Delete(p.nova, serv.server.ID).ExtractErr(); err != nil {
			return fmt.Errorf("cannot remove server for %s: %v", serv, err)
		}
		if err := ports.Delete(p.neutron, serv.port.ID).ExtractErr(); err != nil {
			return fmt.Errorf("cannot remove port for %s: %v", serv, err)
		}
	}

	if err := networks.Delete(p.neutron, app.network.ID).ExtractErr(); err != nil {
		return fmt.Errorf("cannot remove network for %s: %v", appname, err)
	}

	return nil
}
