package web

import (
	"github.com/hashicorp/go-hclog"
)

// Option functions configure the webserver with various values.
type Option func(s *Server) error

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(s *Server) error {
		s.l = l.Named("web")
		return nil
	}
}

// WithStore sets up the storage backend.
func WithStore(st Store) Option {
	return func(s *Server) error {
		s.store = st
		return nil
	}
}

// WithAuth configures the authentication backend.
func WithAuth(auth Auth) Option {
	return func(s *Server) error {
		s.a = auth
		return nil
	}
}
