package sbdioi40

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

// Platform represents an established connection to an OpenStack platform within
// the SBDIOI40 project.
type Platform struct {
	keystone *gophercloud.ServiceClient
	neutron  *gophercloud.ServiceClient
	nova     *gophercloud.ServiceClient
	glance   *gophercloud.ServiceClient
}

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
	keystone, err := openstack.NewIdentityV3(client, opts)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %v", url, err)
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
		keystone: keystone,
		neutron:  neutron,
		nova:     nova,
		glance:   glance,
	}, nil
}

func (p *Platform) String() string {
	return "platform " + p.neutron.IdentityBase
}
