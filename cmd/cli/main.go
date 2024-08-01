package main

import (
	"context"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smallnest/kvbench"
	"github.com/smallnest/log"
)

var (
	duration = flag.Duration("d", 10*time.Second, "test duration for each case")
	c        = flag.Int("c", runtime.NumCPU(), "concurrent goroutines")
	setCount = flag.Int("set", 4000000, "set count")
	size     = flag.Int("size", 256, "data size")
	fsync    = flag.Bool("fsync", false, "fsync")
	s        = flag.String("s", "map", "store type")
	savePath = flag.String("save", "", "save path")
	data     = make([]byte, *size)
)

type Record struct {
	Name    string
	Headers []string
	Values  []int
}

func main() {
	rand.Seed(123)
	flag.Parse()
	fmt.Printf("duration=%v, c=%d size=%d store=%s\n", *duration, *c, *size, *s)

	var memory bool
	var path string
	if strings.HasSuffix(*s, "/memory") {
		memory = true
		path = ":memory:"
		*s = strings.TrimSuffix(*s, "/memory")
	}

	store, path, err := getStore(*s, *fsync, path)
	if err != nil {
		panic(err)
	}
	if !memory {
		defer os.RemoveAll(path)
	}

	defer store.Close()
	name := *s
	if memory {
		name = name + "/memory"
	}
	if *fsync {
		name = name + "/fsync"
	} else {
		name = name + "/nofsync"
	}
	record := &Record{
		Name:   name,
		Values: make([]int, 0),
	}
	record.Headers = append(record.Headers, "name")
	testBatchWriteFixCount(record, name, store, *setCount)
	showMemUsage(record, name)
	showDiskUsage(record, name, path)
	testKeys(record, name, store)
	testSet(record, name, store)
	testGet(record, name, store)
	testGetSet(record, name, store)
	testDelete(record, name, store)
	saveReorder(record)
}

func showMemUsage(record *Record, name string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("%s Alloc = %v MiB", name, m.Alloc/1024/1024)
	fmt.Printf("\tTotalAlloc = %v MiB", m.TotalAlloc/1024/1024)
	fmt.Printf("\tSys = %v MiB", m.Sys/1024/1024)
	fmt.Printf("\tHeapAlloc = %v MiB", m.HeapAlloc/1024/1024)
	fmt.Printf("\tHeapObjects = %v", m.HeapObjects)
	fmt.Printf("\tHeapInuse = %v MiB", m.HeapInuse/1024/1024)
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
	record.Headers = append(record.Headers, "MemUsage(MiB)")
	record.Values = append(record.Values, int(m.Alloc/1024/1024))
	record.Headers = append(record.Headers, "HeapInuse(MiB)")
	record.Values = append(record.Values, int(m.HeapInuse/1024/1024))
}

func showDiskUsage(record *Record, name string, path string) {
	var fileSize int64
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return
	}
	if fileInfo.IsDir() {
		fileSize, err = GetDirSize(path)
	} else {
		fileSize = fileInfo.Size()
	}
	fmt.Printf("%s disk usage: %d MiB\n", name, int(fileSize/1024/1024))
	record.Headers = append(record.Headers, "DiskUsage(MiB)")
	record.Values = append(record.Values, int(fileSize/1024/1024))
}

// test batch writes
func testBatchWrite(name string, store kvbench.Store) {
	var wg sync.WaitGroup
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var total uint64
	for i := 0; i < *c; i++ {
		wg.Add(1)
		go func(proc int) {
			batchSize := uint64(1000)
			var keyList, valList [][]byte
			for i := uint64(0); i < batchSize; i++ {
				keyList = append(keyList, genKey(i))
				valList = append(valList, make([]byte, *size))
			}
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					// Fill random keys and values.
					for i := range keyList {
						rand.Read(keyList[i])
						rand.Read(valList[i])
					}
					err := store.PSet(keyList, valList)
					if err != nil {
						fmt.Printf("%s error: %v\n", name, err)
						panic(err)
					}
					atomic.AddUint64(&total, uint64(len(keyList)))
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Printf("%s batch write test inserted: %d entries; took: %s s\n", name, total, time.Since(start))
}

// test batch writes
func testBatchWriteFixCount(record *Record, name string, store kvbench.Store, count int) {
	start := time.Now()
	var total uint64
	batchSize := 1000
	pageCount := 0
	if count%batchSize == 0 {
		pageCount = count / batchSize
	} else {
		pageCount = count/batchSize + 1
	}
	for i := 0; i < pageCount; i++ {
		startIdx := i * batchSize
		endIdx := startIdx + batchSize
		var keyList, valList [][]byte
		for i := startIdx; i < endIdx; i++ {
			keyList = append(keyList, genKey(uint64(endIdx-startIdx)))
			valList = append(valList, make([]byte, *size))
		}
		for i := range keyList {
			rand.Read(keyList[i])
			rand.Read(valList[i])
			v := rand.Intn(127 - 32)
			keyList[i][0] = byte(32 + v)
		}
		err := store.PSet(keyList, valList)
		if err != nil {
			fmt.Printf("%s error: %v\n", name, err)
			panic(err)
		}
		atomic.AddUint64(&total, uint64(len(keyList)))
	}
	fmt.Printf("%s batch write test inserted: %d entries; took: %s s , mean: %f\n", name, total, time.Since(start), time.Since(start).Seconds())
	record.Headers = append(record.Headers, "batch write cost(s)")
	record.Values = append(record.Values, int(time.Since(start).Seconds()))
}

// test get
func testGet(record *Record, name string, store kvbench.Store) {
	var wg sync.WaitGroup
	wg.Add(*c)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	counts := make([]int, *c)
	start := time.Now()
	for j := 0; j < *c; j++ {
		index := uint64(j)
		go func() {
			var count int
			i := index
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					_, ok, _ := store.Get(genKey(i))
					if !ok {
						i = index
					}
					i += uint64(*c)
					count++
				}
			}
			counts[index] = count
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(start)
	d := int64(dur)
	var n int
	for _, count := range counts {
		n += count
	}
	fmt.Printf("%s get rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(n)*1e6/(d/1e3), d/int64((n)*(*c)), int(dur.Seconds()))
	record.Headers = append(record.Headers, "Get op/s")
	record.Values = append(record.Values, int(int64(n)*1e6/(d/1e3)))
}

// test get
func testKeys(record *Record, name string, store kvbench.Store) {
	_, _, err := store.Keys(genKeyPrefix(0), 0, true)
	if err != nil && errors.Is(err, kvbench.ErrNotSupported) {
		fmt.Printf("%s keys rate: %d op/s, mean: %d ns, took: %d s\n", name, -1, -1, -1)
		record.Headers = append(record.Headers, "Keys op/s")
		record.Values = append(record.Values, -1)
		return
	}
	var wg sync.WaitGroup
	wg.Add(*c)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	counts := make([]int, *c)
	start := time.Now()
	for j := 0; j < *c; j++ {
		index := uint64(j)
		go func() {
			var count int
			i := index
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					_, _, err := store.Keys(genKeyPrefix(i), 0, true)
					if err != nil {
						i = index
					}
					i += uint64(*c)
					count++
				}
			}
			counts[index] = count
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(start)
	d := int64(dur)
	var n int
	for _, count := range counts {
		n += count
	}
	fmt.Printf("%s keys rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(n)*1e6/(d/1e3), d/int64((n)*(*c)), int(dur.Seconds()))
	record.Headers = append(record.Headers, "Keys op/s")
	record.Values = append(record.Values, int(int64(n)*1e6/(d/1e3)))
}

// test multiple get/one set
func testGetSet(record *Record, name string, store kvbench.Store) {
	var wg sync.WaitGroup
	wg.Add(*c)

	ch := make(chan struct{})

	var setCount uint64

	go func() {
		i := uint64(0)
		for {
			select {
			case <-ch:
				return
			default:
				store.Set(genKey(i), data)
				setCount++
				i++
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	counts := make([]int, *c)
	start := time.Now()
	for j := 0; j < *c; j++ {
		index := uint64(j)
		go func() {
			var count int
			i := index
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					store.Get(genKey(i))
					i += uint64(*c)
					count++
				}
			}
			counts[index] = count
			wg.Done()
		}()
	}
	wg.Wait()
	close(ch)
	dur := time.Since(start)
	d := int64(dur)
	var n int
	for _, count := range counts {
		n += count
	}
	record.Headers = append(record.Headers, "Setmixed op/s")
	if setCount == 0 {
		fmt.Printf("%s setmixed rate: -1 op/s, mean: -1 ns, took: %d s\n", name, int(dur.Seconds()))
		record.Values = append(record.Values, -1)
	} else {
		fmt.Printf("%s setmixed rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(setCount)*1e6/(d/1e3), d/int64(setCount), int(dur.Seconds()))
		record.Values = append(record.Values, int(int64(setCount)*1e6/(d/1e3)))
	}
	fmt.Printf("%s getmixed rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(n)*1e6/(d/1e3), d/int64((n)*(*c)), int(dur.Seconds()))
	record.Headers = append(record.Headers, "Getmixed op/s")
	record.Values = append(record.Values, int(int64(n)*1e6/(d/1e3)))
}

func testSet(record *Record, name string, store kvbench.Store) {
	var wg sync.WaitGroup
	wg.Add(*c)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	counts := make([]int, *c)
	start := time.Now()
	for j := 0; j < *c; j++ {
		index := uint64(j)
		go func() {
			count := 0
			i := index
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					store.Set(genKey(i), data)
					i += uint64(*c)
					count++
				}
			}
			counts[index] = count
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(start)
	d := int64(dur)
	var n int
	for _, count := range counts {
		n += count
	}
	fmt.Printf("%s set rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(n)*1e6/(d/1e3), d/int64((n)*(*c)), int(dur.Seconds()))
	record.Headers = append(record.Headers, "Set op/s")
	record.Values = append(record.Values, int(int64(n)*1e6/(d/1e3)))
}

func testDelete(record *Record, name string, store kvbench.Store) {
	var wg sync.WaitGroup
	wg.Add(*c)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	counts := make([]int, *c)
	start := time.Now()
	for j := 0; j < *c; j++ {
		index := uint64(j)
		go func() {
			var count int
			i := index
		LOOP:
			for {
				select {
				case <-ctx.Done():
					break LOOP
				default:
					store.Del(genKey(i))
					i += uint64(*c)
					count++
				}
			}
			counts[index] = count
			wg.Done()
		}()
	}
	wg.Wait()
	dur := time.Since(start)
	d := int64(dur)
	var n int
	for _, count := range counts {
		n += count
	}

	fmt.Printf("%s del rate: %d op/s, mean: %d ns, took: %d s\n", name, int64(n)*1e6/(d/1e3), d/int64((n)*(*c)), int(dur.Seconds()))
	record.Headers = append(record.Headers, "Del op/s")
	record.Values = append(record.Values, int(int64(n)*1e6/(d/1e3)))
}

func genKey(i uint64) []byte {
	r := make([]byte, 9)
	v := rand.Intn(127 - 32)
	r[0] = byte(32 + v)
	binary.BigEndian.PutUint64(r[1:], i)
	return r
}

func genKeyPrefix(i uint64) []byte {
	r := make([]byte, 3)
	rand.Read(r)
	v := rand.Intn(127 - 32)
	r[0] = byte(32 + v)
	return r
}

func getStore(s string, fsync bool, path string) (kvbench.Store, string, error) {
	var store kvbench.Store
	var err error
	switch s {
	default:
		err = fmt.Errorf("unknown store type: %v", s)
	case "map":
		if path == "" {
			path = "map.db"
		}
		store, err = kvbench.NewMapStore(path, fsync)
	case "btree":
		if path == "" {
			path = "btree.db"
		}
		store, err = kvbench.NewBTreeStore(path, fsync)
	case "bolt":
		if path == "" {
			path = "bolt.db"
		}
		store, err = kvbench.NewBoltStore(path, fsync)
	case "bbolt":
		if path == "" {
			path = "bbolt.db"
		}
		store, err = kvbench.NewBboltStore(path, fsync)
	case "leveldb":
		if path == "" {
			path = "leveldb.db"
		}
		store, err = kvbench.NewLevelDBStore(path, fsync)
	case "kv":
		log.Warningf("kv store is unstable")
		if path == "" {
			path = "kv.db"
		}
		store, err = kvbench.NewKVStore(path, fsync)
	case "badger":
		if path == "" {
			path = "badger.db"
		}
		store, err = kvbench.NewBadgerStore(path, fsync)
	case "buntdb":
		if path == "" {
			path = "buntdb.db"
		}
		store, err = kvbench.NewBuntdbStore(path, fsync)
	//case "rocksdb":
	//	if path == "" {
	//		path = "rocksdb.db"
	//	}
	//	store, err = kvbench.NewRocksdbStore(path, fsync)
	case "pebble":
		if path == "" {
			path = "pebble.db"
		}
		store, err = kvbench.NewPebbleStore(path, fsync)
	case "pogreb":
		if path == "" {
			path = "pogreb.db"
		}
		store, err = kvbench.NewPogrebStore(path, fsync)
	case "nutsdb":
		if path == "" {
			path = "nutsdb.db"
		}
		store, err = kvbench.NewNutsdbStore(path, fsync)
	}

	return store, path, err
}

// GetDirSize 用于获取指定目录的总大小（以字节为单位）。
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// 向savePath追加数据
func saveReorder(record *Record) {
	if *savePath == "" {
		return
	}
	values := make([]string, 0, len(record.Values))
	values = append(values, record.Name)
	for _, v := range record.Values {
		values = append(values, strconv.Itoa(v))
	}
	if _, err := os.Stat(*savePath); err != nil && os.IsNotExist(err) {
		file, err := os.Create(*savePath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write(record.Headers)
		writer.Write(values)
		if err := writer.Error(); err != nil {
			log.Fatal(err)
		}
	} else {
		file, err := os.OpenFile(*savePath, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write(values)
		if err := writer.Error(); err != nil {
			log.Fatal(err)
		}
	}
}
