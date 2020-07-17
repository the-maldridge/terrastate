package auth

import (
	"errors"
)

var (
	// ErrNoSuchBackend is to be returned when a backend is
	// requested that does not exist.
	ErrNoSuchBackend = errors.New("no backend with that name exists")

	// ErrUnauthenticated is returned when a user cannot be
	// positively authenticated by any backend.
	ErrUnauthenticated = errors.New("the specified user could not authenticate")
)
