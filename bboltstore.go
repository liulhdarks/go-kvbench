package kvbench

import (
	"bytes"
	"sync"

	"go.etcd.io/bbolt"
)

var bboltBucket = []byte("keys")

type bboltStore struct {
	mu sync.RWMutex
	db *bbolt.DB
}

func bboltKey(key []byte) []byte {
	r := make([]byte, len(key)+1)
	r[0] = 'k'
	copy(r[1:], key)
	return r
}
func NewBboltStore(path string, fsync bool) (Store, error) {
	if path == ":memory:" {
		return nil, ErrMemoryNotAllowed
	}
	db, err := bbolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	db.NoSync = !fsync
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bboltBucket)
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &bboltStore{
		db: db,
	}, nil
}

func (s *bboltStore) Close() error {
	s.db.Close()
	return nil
}

func (s *bboltStore) PSet(keys, values [][]byte) error {
	return s.db.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bboltBucket)
		for i := 0; i < len(keys); i++ {
			if err := b.Put(bboltKey(keys[i]), values[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *bboltStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
	var values [][]byte
	var oks []bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bboltBucket)
		for i := 0; i < len(keys); i++ {
			v := b.Get(bboltKey(keys[i]))
			if v == nil {
				values = append(values, nil)
				oks = append(oks, false)
			} else {
				values = append(values, bcopy(v))
				oks = append(oks, true)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return values, oks, nil
}

func (s *bboltStore) Set(key, value []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(bboltBucket).Put(bboltKey(key), value)
	})
}

func (s *bboltStore) Get(key []byte) ([]byte, bool, error) {
	var v []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		v = tx.Bucket(bboltBucket).Get(bboltKey(key))
		return nil
	})
	return v, v != nil, err
}

func (s *bboltStore) Del(key []byte) (bool, error) {
	var v []byte
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bkey := bboltKey(key)
		v = tx.Bucket(bboltBucket).Get(bkey)
		return tx.Bucket(bboltBucket).Delete(bkey)
	})
	return v != nil, err
}

func (s *bboltStore) Keys(pattern []byte, limit int, withvalues bool) ([][]byte, [][]byte, error) {
	//spattern := string(pattern)
	//min, max := match.Allowable(spattern)
	//bmin := []byte(min)
	var keys [][]byte
	var vals [][]byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte(bboltBucket)).Cursor()

		prefix := pattern
		for key, value := c.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, value = c.Next() {
			keys = append(keys, key)
			if withvalues {
				vals = append(vals, bcopy(value))
			}
		}
		return nil
	})
	//err := s.db.View(func(tx *bbolt.Tx) error {
	//	if len(spattern) > 0 && spattern[0] == '*' {
	//		err := tx.Bucket(bboltBucket).ForEach(func(key, value []byte) error {
	//			if limit > -1 && len(keys) >= limit {
	//				return errors.New("done")
	//			}
	//			skey := string(key[1:])
	//			if match.Match(skey, spattern) {
	//				keys = append(keys, []byte(skey))
	//				if withvalues {
	//					vals = append(vals, bcopy(value))
	//				}
	//			}
	//			return nil
	//		})
	//		if err != nil && err.Error() == "done" {
	//			err = nil
	//		}
	//		return err
	//	}
	//	c := tx.Bucket(bboltBucket).Cursor()
	//	for key, value := c.Seek(bmin); key != nil; key, value = c.Next() {
	//		if limit > -1 && len(keys) >= limit {
	//			break
	//		}
	//		skey := string(key[1:])
	//		if skey >= max {
	//			break
	//		}
	//		if match.Match(skey, spattern) {
	//			keys = append(keys, []byte(skey))
	//			if withvalues {
	//				vals = append(vals, bcopy(value))
	//			}
	//		}
	//	}
	//	return nil
	//})
	return keys, vals, err
}

func (s *bboltStore) FlushDB() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(bboltBucket); err != nil {
			return err
		}
		_, err := tx.CreateBucket(bboltBucket)
		return err
	})
}
