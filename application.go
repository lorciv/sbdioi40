package sbdioi40

import (
	"fmt"
)

// Application holds information about a SBDIOI40 application.
type Application struct {
	Name      string
	networkID string
	Services  []Service
}

func (a Application) String() string {
	return fmt.Sprintf("application %q with %s", a.Name, a.Services)
}

// Service holds information about a virtual machine that belongs to an application.
type Service struct {
	Name     string
	portID   string
	serverID string
}

func (s Service) String() string {
	return fmt.Sprintf("service %q", s.Name)
}
