package sbdioi40

import (
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

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

type Platform struct {
	client *gophercloud.ProviderClient
}

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
			if !strings.HasPrefix(port.Name, application.Name) {
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

func trimPrefixSuffix(s string, prefix string, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSuffix(s, suffix)
}
