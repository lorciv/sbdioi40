package sbdioi40

import (
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	networkutils "github.com/gophercloud/utils/openstack/networking/v2/networks"
)

// Application holds information about a SBDIOI40 application.
type Application struct {
	Name     string
	Services []Service
	network  networks.Network
	subnet   subnets.Subnet
}

// Network returns the network address for the given application. The address is
// specified in CIDR format.
func (a Application) Network() string {
	return a.subnet.CIDR
}

// DNSServers returns the list of the addresses to the DNS servers for the given
// application.
//
// TODO: DNS servers may change across different platforms!
func (a Application) DNSServers() []string {
	return a.subnet.DNSNameservers
}

func (a Application) String() string {
	return fmt.Sprintf("application %q with %s", a.Name, a.Services)
}

// Service holds information about a virtual machine that belongs to an application.
type Service struct {
	Name   string
	port   ports.Port
	server servers.Server
}

// IPAddr returns the IP address for the application's virtual network.
func (s Service) IPAddr() string {
	if len(s.port.FixedIPs) == 0 {
		panic("service with no associated IP address")
	}
	return s.port.FixedIPs[0].IPAddress
}

func (s Service) String() string {
	return fmt.Sprintf("service %q", s.Name)
}

// ListApplications lists all the SBDIOI40 applications that are hosted by the given
// platform.
func (p *Platform) ListApplications() ([]Application, error) {
	iFalse := false
	page, err := networks.List(p.neutron, external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
		External:        &iFalse, // exclude external networks when looking for apps
	}).AllPages()
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

// Application gets information about the named application currently hosted by
// the platform.
func (p *Platform) Application(name string) (Application, error) {
	// get network and subnet
	netID, err := networkutils.IDFromName(p.neutron, name+"net")
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}
	network, err := networks.Get(p.neutron, netID).Extract()
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}

	var subnet *subnets.Subnet
	for _, subnetID := range network.Subnets {
		s, err := subnets.Get(p.neutron, subnetID).Extract()
		if err != nil {
			return Application{}, fmt.Errorf("cannot get subnet of %s: %v", name, err)
		}
		if s.Name == name+"subnet" {
			subnet = s
		}
	}
	if subnet == nil {
		return Application{}, fmt.Errorf("no subnet found for %s", name)
	}

	// get the ports
	page, err := ports.List(p.neutron, ports.ListOpts{
		NetworkID:   netID,
		DeviceOwner: "compute:nova", // skip non-vm ports (such as dhcp)
	}).AllPages()
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}
	allPorts, err := ports.ExtractPorts(page)
	if err != nil {
		return Application{}, fmt.Errorf("cannot get application %s: %v", name, err)
	}

	app := Application{
		Name:    name,
		network: *network,
		subnet:  *subnet,
	}
	for _, port := range allPorts {
		serviceName := trimPrefixSuffix(port.Name, app.Name, "port")
		server, err := servers.Get(p.nova, port.DeviceID).Extract()
		if err != nil {
			return Application{}, fmt.Errorf("cannot get service %s: %v", serviceName, err)
		}
		app.Services = append(app.Services, Service{
			Name:   serviceName,
			port:   port,
			server: *server,
		})
	}

	return app, nil
}

func trimPrefixSuffix(s string, prefix string, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	return strings.TrimSuffix(s, suffix)
}
