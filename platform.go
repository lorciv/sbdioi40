package sbdioi40

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	networkutils "github.com/gophercloud/utils/openstack/networking/v2/networks"
)

// Connect connects to an OpenStack platform within the SBDIOI40 project by using
// the given credentials.
func Connect(url, user, pass string) (*Platform, error) {
	client, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
		IdentityEndpoint: url,
		Username:         user,
		Password:         pass,
		TenantName:       "sbdioi40",
		DomainName:       "Default",
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %v", url, err)
	}

	opts := gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	}
	neutron, err := openstack.NewNetworkV2(client, opts)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %v", url, err)
	}
	nova, err := openstack.NewComputeV2(client, opts)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %v", url, err)
	}
	glance, err := openstack.NewImageServiceV2(client, opts)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %v", url, err)
	}

	return &Platform{
		neutron: neutron,
		nova:    nova,
		glance:  glance,
	}, nil
}

// Platform represents an established connection to an OpenStack platform within
// the sbdioi40 project.
type Platform struct {
	neutron *gophercloud.ServiceClient
	nova    *gophercloud.ServiceClient
	glance  *gophercloud.ServiceClient
}

func (p *Platform) String() string {
	return "platform " + p.neutron.IdentityBase
}

// ListApplications lists the sbdioi40 applications currently hosted by the platform.
func (p *Platform) ListApplications() ([]Application, error) {
	page, err := networks.List(p.neutron, nil).AllPages()
	if err != nil {
		return nil, err
	}
	allNets, err := networks.ExtractNetworks(page)
	if err != nil {
		return nil, err
	}

	var applications []Application

	for _, net := range allNets {
		application, err := p.Application(strings.TrimSuffix(net.Name, "net"))
		if err != nil {
			return nil, err
		}

		applications = append(applications, application)
	}

	return applications, nil
}

// Application gets information about a specific application hosted by the platform.
func (p *Platform) Application(name string) (Application, error) {
	// get the network
	netName := name + "net"
	netID, err := networkutils.IDFromName(p.neutron, netName)
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}

	// get the ports
	page, err := ports.List(p.neutron, ports.ListOpts{
		NetworkID: netID,
	}).AllPages()
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}
	allPorts, err := ports.ExtractPorts(page)
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}

	app := Application{
		Name:      name,
		networkID: netID,
	}
	for _, port := range allPorts {
		if port.DeviceOwner != "compute:nova" {
			// skip non-vm ports (such as dhcp)
			continue
		}

		app.Services = append(app.Services, Service{
			Name:     trimPrefixSuffix(port.Name, app.Name, "port"),
			portID:   port.ID,
			serverID: port.DeviceID,
		})
	}

	return app, nil
}

func trimPrefixSuffix(s string, prefix string, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSuffix(s, suffix)
}
