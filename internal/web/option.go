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

// WithAuth configures the group prefix that will be required
func WithAuthPrefix(prefix string) Option {
	return func(s *Server) {
		s.prefix = prefix
	}
}
