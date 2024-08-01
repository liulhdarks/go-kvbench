package kvbench

import (
	"encoding/binary"
	"flag"
	"os"
	"testing"
)

var count = flag.Int("count", 1000, "item count for test")

var stores = []struct {
	Name    string
	Path    string
	Factory func(path string, fsync bool) (Store, error)
}{
	{"badger", "badger.db", NewBadgerStore},
	{"bbolt", "bbolt.db", NewBboltStore},
	{"bolt", "bolt.db", NewBboltStore},
	{"leveldb", "leveldb.db", NewLevelDBStore},
	{"kv", "kv.db", NewKVStore},
	{"buntdb", "buntdb.db", NewBuntdbStore},
	{"rocksdb", "rocksdb.db", NewRocksdbStore},
	{"pebble", "pebble.db", NewPebbleStore},
	{"pogreb", "pogreb.db", NewPogrebStore},
	{"btree", "btree.db", NewBTreeStore},
	{"btree/memory", ":memory:", NewBTreeStore},
	{"nutsdb", "nutsdb.db", NewNutsdbStore},
	{"map", "map.db", NewMapStore},
	{"map/memory", ":memory:", NewMapStore},
}

func prefixKey(i int) []byte {
	r := make([]byte, 8)
	binary.BigEndian.PutUint64(r, uint64(i))
	return r
}

func wrapfsync(fn func(*testing.T, Store, bool), store Store, fsync bool) func(*testing.T) {
	return func(t *testing.T) {
		fn(t, store, fsync)
	}
}
func TestStore_fsync(t *testing.T) {
	for _, s := range stores {
		store, err := s.Factory(s.Path, true)
		if err != nil {
			os.RemoveAll(s.Path)
			t.Fatal(err)
		}
		t.Run(s.Name, wrapfsync(testStore, store, true))
		os.RemoveAll(s.Path)
	}
}

func TestStore_nofsync(t *testing.T) {
	for _, s := range stores {
		store, err := s.Factory(s.Path, false)
		if err != nil {
			os.RemoveAll(s.Path)
			t.Fatal(err)
		}
		t.Run(s.Name, wrapfsync(testStore, store, false))
		os.RemoveAll(s.Path)
	}
}

func testStore(t *testing.T, store Store, fsync bool) {
	v := make([]byte, 256)

	defer store.Close()

	t.Run("set", func(tt *testing.T) {
		for i := 0; i < *count; i++ {
			err := store.Set(prefixKey(i), v)
			if err != nil {
				tt.Fatalf("failed to set key %d: %v", i, err)
			}
		}
	})

	t.Run("get", func(tt *testing.T) {
		for i := 0; i < *count; i++ {
			_, ok, err := store.Get(prefixKey(i))
			if err != nil {
				tt.Fatalf("failed to get key %d: %v", i, err)
			}
			if !ok {
				tt.Fatalf("the key %d does not exist", i)
			}
		}
	})
}
