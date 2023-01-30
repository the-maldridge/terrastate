package netauth

import (
	"context"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/netauth/netauth/pkg/netauth"
	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"
)

func init() {
	auth.RegisterCallback(cb)
}

func cb() {
	auth.RegisterFactory("netauth", New)
}

// New obtains a new authentication service that uses the NetAuth
// backend.
func New(l hclog.Logger) (web.Auth, error) {
	l = l.Named("netauth")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/netauth/")
	viper.AddConfigPath("$HOME/.netauth")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		l.Error("Fatal error reading configuration", "error", err)
		return nil, err
	}

	// Grab a client
	c, err := netauth.New()
	if err != nil {
		l.Error("Error during NetAuth initialization", "error", err)
		return nil, err
	}
	c.SetServiceName("terrastate")

	prefix := os.Getenv("TS_AUTH_PREFIX")
	if prefix == "" {
		prefix = "terrastate-"
	}
	l.Info("Expecting group prefix", "prefix", prefix)

	x := netAuthBackend{
		nacl:   c,
		l:      l,
		prefix: prefix,
	}

	return &x, nil
}

type netAuthBackend struct {
	nacl *netauth.Client
	l    hclog.Logger

	prefix string
}

func (b *netAuthBackend) AuthUser(ctx context.Context, user, pass, project string) error {
	err := b.nacl.AuthEntity(ctx, user, pass)
	if status.Code(err) != codes.OK {
		return err
	}

	groups, err := b.nacl.EntityGroups(ctx, user)
	if status.Code(err) != codes.OK {
		b.l.Warn("RPC Error: ", "error", err)
		return err
	}

	for _, g := range groups {
		b.l.Trace("Checking group for user", "user", user, "want", b.prefix+project, "have", g.GetName())
		if g.GetName() == b.prefix+project {
			b.l.Info("User authenticated", "project", project, "user", user)
			return nil
		}
	}
	b.l.Warn("User authentication failed", "project", project, "user", user)

	return auth.ErrUnauthenticated
}
