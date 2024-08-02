# KVBench

Cloned from [smallnest/kvbench](https://github.com/smallnest/kvbench). Compared to the smallnest/kvbench codebase:
1. Fixed some incorrect logic of KV database prefix query.
2. Fixed an issue where some KV database configurations were incorrect and persistence was not enabled
3. Batch writing is changed to write a fixed amount of data instead of a fixed time, which can make the subsequent query evaluation fairer.
4. Add the evaluation of prefix query.
5. Add the evaluation of memory usage and disk usage.

KVBench is a Redis server clone backed by a few different Go databases. 

It's intended to be used with the `redis-benchmark` command to test the
performance of various Go databases.  It has support for redis pipelining. The
`redis-benchmark` can run as explained here https://github.com/tidwall/kvbench#examples.

This cloned version adds more kv databases and automatic scripts.

Features:

- Databases
  - [badger](https://github.com/dgraph-io/badger)
  - [BboltDB](https://go.etcd.io/bbolt)
  - [BoltDB](https://github.com/boltdb/bolt)
  - [buntdb](https://github.com/tidwall/buntdb)
  - [LevelDB](https://github.com/syndtr/goleveldb)
  - [cznic/kv](https://github.com/cznic/kv)
  - [rocksdb](https://github.com/tecbot/gorocksdb)
  - [pebble](https://github.com/cockroachdb/pebble)
  - [pogreb](https://github.com/akrylysov/pogreb)
  - [nutsdb](https://github.com/xujiajun/nutsdb)
  - [sniper](https://github.com/recoilme/sniper)
  - map (in-memory) with [AOF persistence](https://redis.io/topics/persistence)
  - btree (in-memory) with [AOF persistence](https://redis.io/topics/persistence)
- Option to disable fsync
- Compatible with Redis clients


## SSD benchmark
The following benchmarks show the throughput of inserting/reading keys (of size
9 bytes) and values (of size 256 bytes). Batch write cost is the time it takes to write 4,000,000 keys and values.

Computer configuration: Apple M1 Pro, 16GB RAM, 1TB SSD

### nofsync

**throughputs**

| name | BatchWrite cost(s) | MemUsage(MiB) | HeapInuse(MiB) | DiskUsage(MiB) | Prefix op/s | Set op/s | Get op/s | Setmixed op/s | Getmixed op/s | Del op/s |
| --- | --- | --- | --- | --- | -- | --- | --- | --- | --- | --- |
| nutsdb | 14 | 1716 | 1741 | 1280 | 135690 | 112565 | 1604634 | 24623 | 274211 | 147513 |
| badger | 11 | 471 | 473 | 2369 | 27352 | 87116 | 547923 | 8317 | 376446 | 121904 |
| bbolt | 139 | 36 | 38 | 1584 | 739850 | 22814 | 781529 | 11605 | 603891 | 92845 |
| bolt | 140 | 34 | 36 | 1584 | 733836 | 19607 | 731267 | 9719 | 542892 | 22224 |
| leveldb | 92 | 17 | 19 | 1060 | 175546 | 56641 | 481400 | 37804 | 94591 | 390857 |
| buntdb | 18 | 1912 | 1915 | 1264 | 411 | 19757 | 2098415 | 8102 | 82720 | 267903 |
| pebble | 81 | 2 | 4 | 1052 | 168755 | 62585 | 559572 | 66058 | 67319 | 384918 |
| pogreb | 40 | 1 | 2 | 1154 | - | 77509 | 2256807 | 45270 | 457269 | 1179826 |
| btree | 11 | 1650 | 1652 | 1113 | 1096963 | 189305 | 2293178 | 64222 | 658351 | 1418046 |
| btree/memory | 8 | 1538 | 1540 | - | 919914 | 2215700 | 68597 | 806471 | 815062 |  |
| map | 5 | 1895 | 1896 | 1113 | 2364810 | 181388 | 5585045 | 98508 | 1111368 | 2514275 |
| map/memory | 2 | 1890 | 1892 | - | 1084926 | 5380740 | 125090 | 1994535 | 1991464 |  |

**Index ranking**

The higher the ranking, the better

| Rank | BatchWrite | MemUsage | DiskUsage | Prefix  | Set | Get | Setmixed | Getmixed | Delete  |
|------|-----------| --- | --- |---------| --- | --- | --- | --- |---------|
| 1    | badger    | pogreb | pebble | bbolt   | nutsdb | pogreb | pebble | bbolt | pogreb  |
| 2    | nutsdb    | pebble | leveldb | bolt    | badger | buntdb | pogreb | bolt | leveldb |
| 3    | buntdb    | leveldb | pogreb | leveldb | pogreb | nutsdb | leveldb | pogreb | pebble  |
| 4    | pogreb    | bolt | buntdb | pebble  | pebble | bbolt | nutsdb | badger | buntdb  |
| 5    | pebble    | bbolt | nutsdb | nutsdb  | leveldb | bolt | bbolt | nutsdb | nutsdb  |
| 6    | leveldb   | badger | bolt | badger  | bbolt | pebble | bolt | leveldb | badger  |
| 7    | bbolt     | nutsdb | bbolt | buntdb  | buntdb | badger | badger | buntdb | bbolt   |
| 8    | bolt      | buntdb | badger | -       | bolt | leveldb | buntdb | pebble | bolt    |

* gogreb does not support prefix queries

### fsync

**throughputs**

Coming soon...


