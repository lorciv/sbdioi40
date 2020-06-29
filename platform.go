package sbdioi40

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

// Platform represents an established connection to an OpenStack platform within
// the SBDIOI40 project.
type Platform struct {
	project  projects.Project
	router   routers.Router
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

	// connect to the essential OpenStack services
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

	// get project-level information
	page, err := projects.List(keystone, projects.ListOpts{
		Name: "sbdioi40",
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("cannot find project %q: %v", "sbdioi40", err)
	}
	allProjects, err := projects.ExtractProjects(page)
	if err != nil {
		return nil, fmt.Errorf("cannot find project %q: %v", "sbdioi40", err)
	}
	project := allProjects[0]

	page, err = routers.List(neutron, routers.ListOpts{
		Name:      "router",
		ProjectID: project.ID,
	}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("cannot find router: %v", err)
	}
	allRouters, err := routers.ExtractRouters(page)
	if err != nil {
		return nil, fmt.Errorf("cannot find router: %v", err)
	}
	router := allRouters[0]

	return &Platform{
		project:  project,
		router:   router,
		keystone: keystone,
		neutron:  neutron,
		nova:     nova,
		glance:   glance,
	}, nil
}

func (p *Platform) String() string {
	return "platform " + p.neutron.IdentityBase
}
