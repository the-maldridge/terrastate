package file

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"
)

var (
	accountsFile string
)

func init() {
	auth.RegisterCallback(cb)

	accountsFile = os.Getenv("TS_USER_FILE")
	if accountsFile == "" {
		accountsFile = "accounts.txt"
	}
}

func cb() {
	auth.RegisterFactory("file", New)
}

// New can be used to get a new instance of this backend
func New(l hclog.Logger) (web.Auth, error) {
	accounts, err := loadAccounts()
	if err != nil {
		return nil, err
	}

	x := fileBackend{
		accounts: accounts,
		l:        l.Named("file"),
	}
	return &x, nil
}

type fileBackend struct {
	accounts map[string]string

	l hclog.Logger
}

func (x *fileBackend) AuthUser(ctx context.Context, user, pass, project string) error {
	key, ok := x.accounts[user]
	if !ok || key != pass {
		return auth.ErrUnauthenticated
	}
	return nil
}

func (x *fileBackend) SetLogger(l hclog.Logger) {
	x.l = l
}

func loadAccounts() (map[string]string, error) {
	data, err := ioutil.ReadFile(accountsFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data[:]), "\n")
	accounts := make(map[string]string)
	for _, l := range lines {
		parts := strings.Split(l, ":")
		if len(parts) != 2 {
			continue
		}
		accounts[parts[0]] = parts[1]
	}
	return accounts, nil
}
