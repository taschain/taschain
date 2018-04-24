package datasource

import (
	"sync"
	"os"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"common"
)

var (
	ErrLDBInit = errors.New("LDB instance not inited")
)

type LDBDatabase struct {
	db *leveldb.DB

	quitLock sync.Mutex
	quitChan chan chan error

	filename      string
	cacheConfig   int
	handlesConfig int

	inited bool
}

func NewLDBDatabase(file string, cache int, handles int) (Database, error) {

	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}

	db, err := newLevelDBInstance(file, cache, handles)
	if err != nil {
		return nil, err
	}

	return &LDBDatabase{
		filename:      file,
		db:            db,
		cacheConfig:   cache,
		handlesConfig: handles,
		inited:        true,
	}, nil
}

// 生成leveldb实例
func newLevelDBInstance(file string, cache int, handles int) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (ldb *LDBDatabase) Clear() error {
	ldb.inited = false
	ldb.Close()

	// todo: 直接删除文件，是不是过于粗暴？
	os.RemoveAll(ldb.Path())

	db, err := newLevelDBInstance(ldb.Path(), ldb.cacheConfig, ldb.handlesConfig)
	if err != nil {
		return err
	}

	ldb.db = db
	ldb.inited = true
	return nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return db.filename
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	if !db.inited {
		return ErrLDBInit
	}

	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	if !db.inited {
		return false, ErrLDBInit
	}

	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	if !db.inited {
		return nil, ErrLDBInit
	}

	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil

}

// Delete deletes the key from the queue and database
func (db *LDBDatabase) Delete(key []byte) error {
	if !db.inited {
		return ErrLDBInit
	}
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	if !db.inited {
		return nil
	}
	return db.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (db *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (db *LDBDatabase) Close() {
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			//db.log.Error("Metrics collection failed", "err", err)
		}
	}

	db.db.Close()
	//err := db.db.Close()
	//if err == nil {
	//	db.log.Info("Database closed")
	//} else {
	//	db.log.Error("Failed to close database", "err", err)
	//}
}

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type ldbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}


type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() (*MemDatabase, error) {
	return &MemDatabase{
		db: make(map[string][]byte),
	}, nil
}

func (db *MemDatabase) Clear() error{
	db.db =make(map[string][]byte)
	return nil
}
func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) Close() {}

func (db *MemDatabase) NewBatch() Batch {
	return &memBatch{db: db}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct{ k, v []byte }

type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}