package web

import (
	"github.com/hashicorp/go-hclog"
)

// Option functions configure the webserver with various values.
type Option func(s *Server)

// WithLogger sets the logger for the server.
func WithLogger(l hclog.Logger) Option {
	return func(s *Server) {
		s.l = l.Named("web")
	}
}

// WithStore sets up the storage backend.
func WithStore(st Store) Option {
	return func(s *Server) {
		s.store = st
	}
}

// WithAuth configures the authentication backend.
func WithAuth(auth Auth) Option {
	return func(s *Server) {
		s.a = append(s.a, auth)
	}
}
