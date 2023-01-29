package bc

import (
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/prologic/bitcask"

	"github.com/the-maldridge/terrastate/internal/store"
	"github.com/the-maldridge/terrastate/internal/web"
)

// bcStore is the type that must satisfy web.Store
type bcStore struct {
	s *bitcask.Bitcask

	l hclog.Logger
}

func init() {
	store.RegisterCallback(newFactory)
}

func newFactory() {
	store.RegisterFactory("bitcask", newBCStore)
}

func newBCStore(l hclog.Logger) (web.Store, error) {
	x := new(bcStore)
	x.l = l.Named("bitcask")

	p := os.Getenv("TS_BITCASK_PATH")
	if p == "" {
		l.Error("TS_BITCASK_PATH must be set")
		return nil, errors.New("required variable unset")
	}

	opts := []bitcask.Option{
		bitcask.WithMaxKeySize(1024),
		bitcask.WithMaxValueSize(1024 * 1000 * 5), // 5MiB
		bitcask.WithSync(true),
	}
	b, err := bitcask.Open(p, opts...)
	if err != nil {
		l.Error("Error initializing bitcask", "error", err)
		return nil, err
	}
	x.s = b

	return x, nil
}

func (b *bcStore) Get(k []byte) ([]byte, error) {
	v, err := b.s.Get(k)
	switch err {
	case nil:
		return v, nil
	case bitcask.ErrKeyNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *bcStore) Put(k, v []byte) error {
	return b.s.Put(k, v)
}

func (b *bcStore) Del(k []byte) error {
	return b.s.Delete(k)
}

func (b *bcStore) Close() error {
	return b.s.Close()
}

func (b *bcStore) Sync() error {
	return b.s.Sync()
}
