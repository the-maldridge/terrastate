package netauth

import (
	"flag"
	"log"

	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NetAuth/NetAuth/pkg/client"
	"github.com/the-maldridge/TerraState/internal/auth"
)

var (
	reqGroup = flag.String("required_group", "", "Required group for use of TerraState")
)

func init() {
	auth.Register("NetAuth", New)
}

// New obtains a new authentication service that uses the NetAuth
// backend.
func New() (auth.Service, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/netauth/")
	viper.AddConfigPath("$HOME/.netauth")
	viper.AddConfigPath(".")
	viper.Set("client.ServiceName", "TerraState")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Fatal error reading configuration: ", err)
	}

	// Grab a client
	c, err := client.New()
	if err != nil {
		return nil, err
	}

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

	if *reqGroup == "" {
		return nil
	}

	groups, err := b.nacl.ListGroups(user, true)
	if status.Code(err) != codes.OK {
		log.Println("RPC Error: ", err)
		return auth.ErrUnauthenticated
	}

	for _, g := range groups {
		if g.GetName() == *reqGroup {
			return nil
		}
	}

	return auth.ErrUnauthenticated
}
