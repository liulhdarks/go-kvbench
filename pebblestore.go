package kvbench

import (
	"fmt"
	"io"
	"sync"

	"github.com/cockroachdb/pebble"
)

type pebbleStore struct {
	mu sync.RWMutex
	db *pebble.DB
	wo *pebble.WriteOptions
}

func pebbleKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}

func NewPebbleStore(path string, fsync bool) (Store, error) {
	if path == ":memory:" {
		return nil, ErrMemoryNotAllowed
	}

	opts := &pebble.Options{}
	if !fsync {
		opts.DisableWAL = true
	}

	wo := &pebble.WriteOptions{}
	wo.Sync = fsync

	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}

	return &pebbleStore{
		db: db,
		wo: wo,
	}, nil
}

func (s *pebbleStore) Close() error {
	s.db.Close()
	return nil
}

func (s *pebbleStore) PSet(keys, vals [][]byte) error {
	wb := s.db.NewBatch()

	for i, k := range keys {
		wb.Set(k, vals[i], s.wo)
	}
	return wb.Commit(s.wo)
}

func (s *pebbleStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var vals = make([][]byte, len(keys))
	var oks = make([]bool, len(keys))

	var err error
	var closer io.Closer
	for i, k := range keys {
		vals[i], closer, err = s.db.Get(k)
		oks[i] = (err == nil)
		if closer != nil {
			closer.Close()
		}
	}
	return vals, oks, err
}

func (s *pebbleStore) Set(key, value []byte) error {
	return s.db.Set(key, value, s.wo)
}

func (s *pebbleStore) Get(key []byte) ([]byte, bool, error) {
	v, closer, err := s.db.Get(key)
	if err != nil {
		if err != pebble.ErrNotFound {
			fmt.Println(err)
		}
		return nil, false, err
	}
	if closer != nil {
		closer.Close()
	}
	return v, v != nil, err
}

func (s *pebbleStore) Del(key []byte) (bool, error) {
	err := s.db.Delete(key, s.wo)
	return err == nil, err
}

func (s *pebbleStore) Keys(pattern []byte, limit int, withvals bool) ([][]byte, [][]byte, error) {
	var keys [][]byte
	var vals [][]byte

	// 定义前缀
	prefix := []byte("myprefix")

	// 创建迭代器，指定范围为前缀
	iter := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xFF),
	})
	defer iter.Close()

	// 遍历具有前缀的所有键值对
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		keys = append(keys, key)
		if withvals {
			value := iter.Value()
			vals = append(vals, value)
		}
	}

	//io := &pebble.IterOptions{}
	//it := s.db.NewIter(io)
	//defer it.Close()
	//it.SeekGE(pattern)
	//
	//for ; it.Valid(); it.Next() {
	//	key := it.Key()
	//	if !bytes.HasPrefix(key, pattern) {
	//		break
	//	}
	//
	//	k := it.Key()
	//	keys = append(keys, k)
	//
	//	if withvals {
	//		value := it.Value()
	//		vals = append(vals, value)
	//	}
	//}

	return keys, vals, nil
}

func (s *pebbleStore) FlushDB() error {
	return s.db.Flush()
}
