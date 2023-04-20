package ldap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"
)

func init() {
	auth.RegisterCallback(cb)
}

func cb() {
	auth.RegisterFactory("ldap", New)
}

type ldapBackend struct {
	l hclog.Logger

	prefix    string
	url       string
	base      string
	groupAttr string
	bindTmpl  string
}

// New obtains a new authentication service that uses an LDAP server.
func New(l hclog.Logger) (web.Auth, error) {
	x := ldapBackend{
		l:         l,
		prefix:    os.Getenv("TS_AUTH_PREFIX"),
		url:       os.Getenv("TS_LDAP_URL"),
		base:      os.Getenv("TS_LDAP_BASEDN"),
		groupAttr: os.Getenv("TS_LDAP_GROUPATTR"),
		bindTmpl:  os.Getenv("TS_LDAP_BIND_TEMPLATE"),
	}

	if x.url == "" {
		l.Error("Missing required config value", "key", "TS_LDAP_URL")
		return nil, errors.New("must specify TS_LDAP_URL")
	}

	if x.base == "" {
		l.Error("Missing required config value", "key", "TS_LDAP_BASEDN")
		return nil, errors.New("must specify TS_LDAP_BASEDN")
	}

	if x.bindTmpl == "" {
		l.Error("Missing required config value", "key", "TS_LDAP_BIND_TEMPLATE")
		return nil, errors.New("must specify TS_LDAP_BIND_TEMPLATE")
	}

	return &x, nil
}

func (l *ldapBackend) AuthUser(ctx context.Context, user, pass, project string) error {
	ldc, err := ldap.DialURL(l.url)
	if err != nil {
		l.l.Error("Error dialing LDAP server", "error", err)
		return err
	}

	if err := ldc.Bind(fmt.Sprintf(l.bindTmpl, user), pass); err != nil {
		return err
	}

	searchReq := ldap.NewSearchRequest(
		l.base,                        // BaseDN
		ldap.ScopeWholeSubtree,        // Scope
		ldap.NeverDerefAliases,        // DerefAliases
		1,                             // SizeLimit - We only expect to match exactly one user
		10,                            // TimeLimit
		false,                         // TypesOnly
		fmt.Sprintf("(uid=%s)", user), // Filter - Should match authenticated user
		[]string{l.groupAttr},         // Attributes
		nil,                           // Controls
	)

	res, err := ldc.Search(searchReq)
	if err != nil {
		l.l.Error("Error while performing ldap search", "error", err)
		return err
	}

	if len(res.Entries) == 0 {
		l.l.Warn("No resultant entity for authenticated user!?", "user", user)

		// Something weird is up, lets bail now.
		return auth.ErrUnauthenticated
	}

	for _, g := range res.Entries[0].GetAttributeValues(l.groupAttr) {
		// This is ugly and not fully correct according to the
		// spec for parsing a DN.  In reality this will work
		// for 99% of use cases, and is easier to reason about
		// what its doing.
		grp := strings.Split(strings.Split(g, ",")[0], "=")[1]

		l.l.Trace("Checking group for user", "user", user, "want", l.prefix+project, "have", grp)
		if grp == l.prefix+project {
			l.l.Info("User authenticated", "project", project, "user", user)
			return nil
		}
	}
	l.l.Warn("User authentication failed", "project", project, "user", user)

	return auth.ErrUnauthenticated
}
