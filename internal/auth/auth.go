package auth

import (
	"flag"
)

var (
	backends map[string]ServiceFactory
	backend  = flag.String("auth_backend", "", "Authentication Backend")
)

// Service contains the methods needed to authenticate a caller to
// TerraState.
type Service interface {
	AuthUser(string, string) error
}

// A ServiceFactory creates an auth.Service
type ServiceFactory func() (Service, error)

func init() {
	backends = make(map[string]ServiceFactory)
}

// Register allows implementations of the auth.Service to register
// themselves on startup.
func Register(name string, f ServiceFactory) {
	if _, ok := backends[name]; ok {
		// Already registered
		return
	}
	backends[name] = f
}

// New returns a service for authentication ready to use.
func New() (Service, error) {
	b, ok := backends[*backend]
	if !ok {
		return nil, ErrNoSuchBackend
	}
	return b()
}

// List provides a list of known authentication mechanisms
func List() []string {
	l := []string{}
	for b := range backends {
		l = append(l, b)
	}
	return l
}
