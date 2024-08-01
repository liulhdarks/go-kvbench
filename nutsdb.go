package kvbench

import (
	"sync"

	"github.com/xujiajun/nutsdb"
)

var nutsdbBucket = "keys"

type nutsdbStore struct {
	mu sync.RWMutex
	db *nutsdb.DB
}

var defaultSegmentSize int64 = 256 * nutsdb.MB

// DefaultOptions represents the default options.
var pptions = func() nutsdb.Options {
	return nutsdb.Options{
		EntryIdxMode: nutsdb.HintKeyAndRAMIdxMode,
		SegmentSize:  defaultSegmentSize,
		NodeNum:      1,
		RWMode:       nutsdb.FileIO,
		SyncEnable:   false,
	}
}()

func nutsdbKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}
func NewNutsdbStore(path string, fsync bool) (Store, error) {
	if path == ":memory:" {
		return nil, ErrMemoryNotAllowed
	}

	opt := pptions
	opt.SyncEnable = fsync
	opt.Dir = path

	db, err := nutsdb.Open(opt)
	if err != nil {
		return nil, err
	}

	return &nutsdbStore{
		db: db,
	}, nil
}

func (s *nutsdbStore) Close() error {
	s.db.Close()
	return nil
}

func (s *nutsdbStore) PSet(keys, vals [][]byte) error {
	return s.db.Update(func(tx *nutsdb.Tx) error {
		for i, k := range keys {
			tx.Put(nutsdbBucket, k, vals[i], 0)
		}

		return nil
	})
}

func (s *nutsdbStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var vals = make([][]byte, len(keys))
	var oks = make([]bool, len(keys))

	var err error

	s.db.View(func(tx *nutsdb.Tx) error {
		for i, k := range keys {
			e, err := tx.Get(nutsdbBucket, k)
			if e != nil {
				vals[i] = e.Value
			}

			oks[i] = (err == nil)
		}

		return nil
	})

	return vals, oks, err
}

func (s *nutsdbStore) Set(key, value []byte) error {
	return s.db.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(nutsdbBucket, key, value, 0)
	})
}

func (s *nutsdbStore) Get(key []byte) ([]byte, bool, error) {
	var v []byte
	var ok bool
	var err error
	var e *nutsdb.Entry

	s.db.View(func(tx *nutsdb.Tx) error {
		e, err = tx.Get(nutsdbBucket, key)
		if e != nil {
			v = e.Value
		}
		ok = err == nil
		return err
	})

	return v, ok, err
}

func (s *nutsdbStore) Del(key []byte) (bool, error) {
	err := s.db.Update(func(tx *nutsdb.Tx) error {
		return tx.Delete(nutsdbBucket, key)
	})

	return err == nil, err
}

func (s *nutsdbStore) Keys(pattern []byte, limit int, withvals bool) ([][]byte, [][]byte, error) {
	var keys [][]byte
	var vals [][]byte

	err := s.db.View(func(tx *nutsdb.Tx) error {
		entries, _, err := tx.PrefixScan(nutsdbBucket, pattern, 0, nutsdb.ScanNoLimit)

		if err != nil {
			return err
		}

		for i, entry := range entries {
			keys[i] = entry.Key
			vals[i] = entry.Value
		}

		return nil
	})

	return keys, vals, err
}

func (s *nutsdbStore) FlushDB() error {
	return s.db.Close()
}
