package kvbench

import (
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type leveldbStore struct {
	mu    sync.RWMutex
	db    *leveldb.DB
	path  string
	fsync bool
	wo    *opt.WriteOptions
}

func NewLevelDBStore(path string, fsync bool) (Store, error) {
	if path == ":memory:" {
		return nil, ErrMemoryNotAllowed
	}
	opts := &opt.Options{NoSync: !fsync}
	db, err := leveldb.OpenFile(path, opts)
	if err != nil {
		return nil, err
	}
	return &leveldbStore{
		db:    db,
		path:  path,
		fsync: fsync,
		wo:    &opt.WriteOptions{Sync: fsync},
	}, nil
}

func (s *leveldbStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Close()
	return nil
}
func (s *leveldbStore) PSet(keys, values [][]byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	batch := new(leveldb.Batch)
	for i := range keys {
		batch.Put(keys[i], values[i])
	}
	return s.db.Write(batch, s.wo)
}

func (s *leveldbStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var values [][]byte
	var oks []bool
	for i := range keys {
		value, ok, err := s.Get(keys[i])
		if err != nil {
			return nil, nil, err
		}
		values = append(values, value)
		oks = append(oks, ok)
	}
	return values, oks, nil
}

func (s *leveldbStore) Set(key, value []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Put(key, value, s.wo)
}

func (s *leveldbStore) Get(key []byte) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, err := s.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return v, true, nil
}

func (s *leveldbStore) Del(key []byte) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ok, err := s.db.Has(key, nil)
	if !ok || err != nil {
		return ok, err
	}
	err = s.db.Delete(key, s.wo)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *leveldbStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	var keys [][]byte
	var vals [][]byte
	iter := s.db.NewIterator(util.BytesPrefix([]byte("foo-")), nil)
	for iter.Next() {
		key := iter.Key()
		keys = append(keys, key)
		if withvalues {
			value := iter.Value()
			vals = append(vals, value)
		}
	}
	iter.Release()
	return keys, vals, nil
}

func (s *leveldbStore) FlushDB() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Close()
	os.RemoveAll(s.path)
	s.db = nil
	var opts *opt.Options
	if !s.fsync {
		opts = &opt.Options{NoSync: !s.fsync}
	}
	db, err := leveldb.OpenFile(s.path, opts)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}
