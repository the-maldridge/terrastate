package file

import (
	"flag"
	"io/ioutil"
	"strings"

	"github.com/the-maldridge/terrastate/internal/auth"
)

var (
	accountsFile = flag.String("accounts_file", "./accounts.txt", "Accounts file")
)

func init() {
	auth.Register("file", New)
}

// New can be used to get a new instance of this backend
func New() (auth.Service, error) {
	accounts, err := loadAccounts()
	if err != nil {
		return nil, err
	}

	x := fileBackend{
		accounts: accounts,
	}
	return &x, nil
}

type fileBackend struct {
	accounts map[string]string
}

func (x *fileBackend) AuthUser(user, pass string) error {
	key, ok := x.accounts[user]
	if !ok || key != pass {
		return auth.ErrUnauthenticated
	}
	return nil
}

func loadAccounts() (map[string]string, error) {
	data, err := ioutil.ReadFile(*accountsFile)
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
