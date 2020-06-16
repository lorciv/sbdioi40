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
	osClient, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
		IdentityEndpoint: url,
		Username:         user,
		Password:         pass,
		TenantName:       "sbdioi40",
		DomainName:       "Default",
	})
	if err != nil {
		return nil, err
	}

	return &Platform{client: osClient}, nil
}

// Platform represents an established connection to an OpenStack platform within
// the sbdioi40 project.
type Platform struct {
	client *gophercloud.ProviderClient
}

func (p *Platform) String() string {
	return "platform " + p.client.IdentityBase
}

// ListApplications lists the sbdioi40 applications currently hosted by the platform.
func (p *Platform) ListApplications() ([]Application, error) {
	neutron, err := openstack.NewNetworkV2(p.client, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, err
	}

	page, err := networks.List(neutron, nil).AllPages()
	if err != nil {
		return nil, err
	}
	allNets, err := networks.ExtractNetworks(page)
	if err != nil {
		return nil, err
	}

	var applications []Application

	for _, net := range allNets {
		application := Application{
			Name:      strings.TrimSuffix(net.Name, "net"),
			networkID: net.ID,
		}

		page, err := ports.List(neutron, ports.ListOpts{
			NetworkID: application.networkID,
		}).AllPages()
		if err != nil {
			return nil, err
		}
		allPorts, err := ports.ExtractPorts(page)
		if err != nil {
			return nil, err
		}

		for _, port := range allPorts {
			if port.DeviceOwner != "compute:nova" {
				// skip non-vm ports (such as dhcp)
				continue
			}
			application.Services = append(application.Services, Service{
				Name:     trimPrefixSuffix(port.Name, application.Name, "port"),
				portID:   port.ID,
				serverID: port.DeviceID,
			})
		}

		applications = append(applications, application)
	}

	return applications, nil
}

// Application gets information about a specific application hosted by the platform.
func (p *Platform) Application(name string) (Application, error) {
	neutron, err := openstack.NewNetworkV2(p.client, gophercloud.EndpointOpts{
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return Application{}, fmt.Errorf("neutron connection failed: %v", err)
	}

	// get the network
	netName := name + "net"
	netID, err := networkutils.IDFromName(neutron, netName)
	if err != nil {
		return Application{}, fmt.Errorf("application %s does not exist", name)
	}

	// get the ports
	page, err := ports.List(neutron, ports.ListOpts{
		NetworkID: netID,
	}).AllPages()
	if err != nil {
		return Application{}, err
	}
	allPorts, err := ports.ExtractPorts(page)
	if err != nil {
		return Application{}, err
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
