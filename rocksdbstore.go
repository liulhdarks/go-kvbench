package kvbench

//
//import (
//	"sync"
//
//	rocksdb "github.com/tecbot/gorocksdb"
//)
//
//type rocksdbStore struct {
//	mu sync.RWMutex
//	db *rocksdb.DB
//	ro *rocksdb.ReadOptions
//	wo *rocksdb.WriteOptions
//	fo *rocksdb.FlushOptions
//}
//
//func rocksdbKey(key []byte) []byte {
//	r := make([]byte, len(key)+1)
//	r[0] = 'k'
//	copy(r[1:], key)
//	return r
//}
//
//func NewRocksdbStore(path string, fsync bool) (Store, error) {
//	if path == ":memory:" {
//		return nil, ErrMemoryNotAllowed
//	}
//
//	opts := rocksdb.NewDefaultOptions()
//	opts.SetCreateIfMissing(true)
//
//	ro := rocksdb.NewDefaultReadOptions()
//	ro.SetFillCache(false)
//
//	wo := rocksdb.NewDefaultWriteOptions()
//	wo.SetSync(fsync)
//
//	fo := rocksdb.NewDefaultFlushOptions()
//
//	db, err := rocksdb.OpenDb(opts, path)
//	if err != nil {
//		return nil, err
//	}
//
//	return &rocksdbStore{
//		db: db,
//		ro: ro,
//		wo: wo,
//		fo: fo,
//	}, nil
//}
//
//func (s *rocksdbStore) Close() error {
//	s.db.Close()
//	return nil
//}
//
//func (s *rocksdbStore) PSet(keys, vals [][]byte) error {
//	wb := rocksdb.NewWriteBatch()
//	defer wb.Destroy()
//
//	for i, k := range keys {
//		wb.Put(k, vals[i])
//	}
//	return s.db.Write(s.wo, wb)
//}
//
//func (s *rocksdbStore) PGet(keys [][]byte) ([][]byte, []bool, error) {
//	var vals = make([][]byte, len(keys))
//	var oks = make([]bool, len(keys))
//
//	var err error
//	for i, k := range keys {
//		vals[i], err = s.db.GetBytes(s.ro, k)
//		oks[i] = (err == nil)
//	}
//	return vals, oks, err
//}
//
//func (s *rocksdbStore) Set(key, value []byte) error {
//	return s.db.Put(s.wo, key, value)
//}
//
//func (s *rocksdbStore) Get(key []byte) ([]byte, bool, error) {
//	v, err := s.db.GetBytes(s.ro, key)
//	return v, v != nil, err
//}
//
//func (s *rocksdbStore) Del(key []byte) (bool, error) {
//	err := s.db.Delete(s.wo, key)
//	return err == nil, err
//}
//
//func (s *rocksdbStore) Keys(pattern []byte, limit int, withvals bool) ([][]byte, [][]byte, error) {
//	var keys [][]byte
//	var vals [][]byte
//
//	it := s.db.NewIterator(s.ro)
//	defer it.Close()
//	it.Seek(pattern)
//
//	for it = it; it.Valid(); it.Next() {
//		key := it.Key()
//
//		k := make([]byte, key.Size())
//		copy(k, key.Data())
//		key.Free()
//
//		keys = append(keys, k)
//
//		if withvals {
//			value := it.Value()
//			v := make([]byte, value.Size())
//			copy(v, value.Data())
//			value.Free()
//			vals = append(vals, v)
//		}
//	}
//
//	return keys, vals, nil
//}
//
//func (s *rocksdbStore) FlushDB() error {
//	return s.db.Flush(s.fo)
//}
