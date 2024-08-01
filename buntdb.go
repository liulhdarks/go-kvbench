package kvbench

import (
	"sync"

	"github.com/tidwall/buntdb"
)

type buntdbStore struct {
	mu sync.RWMutex
	db *buntdb.DB
}

func buntdbKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}

func NewBuntdbStore(path string, fsync bool) (Store, error) {
	opts := buntdb.Config{}
	if fsync {
		opts.SyncPolicy = buntdb.Always
	}
	db, err := buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	db.SetConfig(opts)

	return &buntdbStore{
		db: db,
	}, nil
}

func (s *buntdbStore) Close() error {
	s.db.Close()
	return nil
}

func (s *buntdbStore) PSet(keys, vals [][]byte) error {
	var err error
	s.db.Update(func(tx *buntdb.Tx) error {
		for i, k := range keys {
			_, _, err := tx.Set(string(k), string(vals[i]), nil)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (s *buntdbStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var vals = make([][]byte, len(keys))
	var oks = make([]bool, len(keys))

	err := s.db.View(func(tx *buntdb.Tx) error {
		for i, k := range keys {
			v, err := tx.Get(string(k))
			if err == nil {
				vals[i] = []byte(v)
				oks[i] = true
			}
		}
		return nil
	})

	return vals, oks, err
}

func (s *buntdbStore) Set(key, value []byte) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(string(key), string(value), nil)
		return err
	})
}

func (s *buntdbStore) Get(key []byte) ([]byte, bool, error) {
	var v []byte

	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(string(key))
		if err == nil {
			v = []byte(val)
		}
		return err
	})

	return v, v != nil, err
}

func (s *buntdbStore) Del(key []byte) (bool, error) {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(string(key))
		return err
	})
	return err == nil, err
}

func (s *buntdbStore) Keys(pattern []byte, limit int, withvals bool) ([][]byte, [][]byte, error) {
	var keys [][]byte
	var vals [][]byte

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(string(pattern), func(key, value string) bool {
			keys = append(keys, []byte(key))
			if withvals {
				vals = append(vals, []byte(value))
			}
			return true
		})
	})

	return keys, vals, err
}

func (s *buntdbStore) FlushDB() error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		return tx.DeleteAll()
	})
}
