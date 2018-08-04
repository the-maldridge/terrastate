package netauth

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NetAuth/NetAuth/pkg/client"
	"github.com/the-maldridge/TerraState/internal/auth"
)

func init() {
	auth.Register("NetAuth", New)
}

// New obtains a new authentication service that uses the NetAuth
// backend.
func New() (auth.Service, error) {
	// Grab a client
	c, err := client.New(nil)
	if err != nil {
		return nil, err
	}

	// Set the service ID
	c.SetServiceID("TerraState")

	x := netAuthBackend{
		nacl: c,
	}

	return &x, nil
}

type netAuthBackend struct {
	nacl *client.NetAuthClient
}

func (b *netAuthBackend) AuthUser(user, pass string) error {
	result, err := b.nacl.Authenticate(user, pass)
	if status.Code(err) != codes.OK || !result.GetSuccess() {
		return auth.ErrUnauthenticated
	}
	return nil
}
