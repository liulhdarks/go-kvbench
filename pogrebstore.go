package kvbench

import (
	"sync"

	"github.com/akrylysov/pogreb"
)

var pogrebBucket = []byte("keys")

type pogrebStore struct {
	mu sync.RWMutex
	db *pogreb.DB
}

func pogrebKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}
func NewPogrebStore(path string, fsync bool) (Store, error) {
	if path == ":memory:" {
		return nil, ErrMemoryNotAllowed
	}

	opts := &pogreb.Options{}

	if fsync {
		opts.BackgroundSyncInterval = -1
	}

	db, err := pogreb.Open(path, opts)
	if err != nil {
		return nil, err
	}

	return &pogrebStore{
		db: db,
	}, nil
}

func (s *pogrebStore) Close() error {
	s.db.Close()
	return nil
}

func (s *pogrebStore) PSet(keys, values [][]byte) error {
	var err error
	for i, k := range keys {
		e := s.db.Put(k, values[i])
		if e != nil {
			err = e
		}
	}
	return err
}

func (s *pogrebStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var values [][]byte
	var oks []bool
	var e, err error

	for i, k := range keys {
		values[i], e = s.db.Get(k)
		if e != nil {
			err = e
			oks = append(oks, false)
		} else {
			oks = append(oks, true)
		}
	}

	return values, oks, err
}

func (s *pogrebStore) Set(key, value []byte) error {
	return s.db.Put(key, value)
}

func (s *pogrebStore) Get(key []byte) ([]byte, bool, error) {
	v, err := s.db.Get(key)
	return v, err == nil, err
}

func (s *pogrebStore) Del(key []byte) (bool, error) {
	err := s.db.Delete(key)
	return err == nil, err
}

func (s *pogrebStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	return nil, nil, ErrNotSupported
}

func (s *pogrebStore) FlushDB() error {
	return s.db.Close()
}
