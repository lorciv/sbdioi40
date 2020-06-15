package sbdioi40

import (
	"fmt"
)

type Application struct {
	Name      string
	networkID string
	Services  []Service
}

func (a Application) String() string {
	return fmt.Sprintf("application %q with %s", a.Name, a.Services)
}

type Service struct {
	Name     string
	portID   string
	serverID string
}

func (s Service) String() string {
	return fmt.Sprintf("service %q", s.Name)
}
