package htpasswd

import (
	"context"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/tg123/go-htpasswd"

	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"
)

var (
	htpasswdFile string
	htgroupFile  string
)

type htpasswdBackend struct {
	l hclog.Logger

	f *htpasswd.File
	g *htpasswd.HTGroup
}

func init() {
	auth.RegisterCallback(cb)

	htpasswdFile = os.Getenv("TS_HTPASSWD_FILE")
	if htpasswdFile == "" {
		htpasswdFile = ".htpasswd"
	}

	htgroupFile = os.Getenv("TS_HTGROUP_FILE")
	if htgroupFile == "" {
		htgroupFile = ".htgroup"
	}
}

func cb() {
	auth.RegisterFactory("htpasswd", New)
}

// New can be used to get a new instance of this backend
func New(l hclog.Logger) (web.Auth, error) {
	f, err := htpasswd.New(htpasswdFile, htpasswd.DefaultSystems, nil)
	if err != nil {
		return nil, err
	}

	g, err := htpasswd.NewGroups(htgroupFile, nil)
	if err != nil {
		return nil, err
	}

	x := htpasswdBackend{
		l: l.Named("htpasswd"),
		f: f,
		g: g,
	}
	return &x, nil
}
func (h *htpasswdBackend) AuthUser(ctx context.Context, user, pass, project string) error {
	if !h.f.Match(user, pass) {
		return auth.ErrUnauthenticated
	}

	if !h.g.IsUserInGroup(user, project) {
		return auth.ErrUnauthenticated
	}

	return nil
}

func (h *htpasswdBackend) SetLogger(l hclog.Logger) {
	h.l = l
}
