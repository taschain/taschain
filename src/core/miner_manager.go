package core

import (
	"storage/tasdb"
	"core/datasource"
	"middleware/types"
	"github.com/vmihailenco/msgpack"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/hashicorp/golang-lru"
)

type MinerManager struct {
	lightDB tasdb.Database
	heavyDB tasdb.Database
	cache   *lru.Cache
}

type MinerIterator struct {
	iter iterator.Iterator
	cache   *lru.Cache
}

var MinerManagerImpl *MinerManager

func initMinerManager(config *BlockChainConfig) error {
	cache,_ := lru.New(500)
	MinerManagerImpl = &MinerManager{cache:cache}
	lightDB,err := datasource.NewDatabase(config.light)
	if err != nil{
		return err
	}
	MinerManagerImpl.lightDB = lightDB
	heavyDB,err := datasource.NewDatabase(config.heavy)
	if err != nil{
		return err
	}
	MinerManagerImpl.heavyDB = heavyDB
	return nil
}

func (mm *MinerManager) getMinerDatabase(ttype byte) tasdb.Database{
	switch ttype {
	case types.MinerTypeLight:
		return  mm.lightDB
	case types.MinerTypeHeavy:
		return mm.heavyDB
	}
	return nil
}

//返回值：1成功添加，-1旧数据仍然存在，添加失败
func (mm *MinerManager) AddMiner(id []byte, miner *types.Miner) int {
	db := mm.getMinerDatabase(miner.Type)

	if exist,_ := db.Has(id); exist{
		return -1
	} else {
		data,_ := msgpack.Marshal(miner)
		db.Put(id, data)
		return 1
	}
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte) (*types.Miner,error) {
	if ttype == types.MinerTypeHeavy {
		if result, ok := mm.cache.Get(id); ok {
			return result.(*types.Miner),nil
		}
	}
	db := mm.getMinerDatabase(ttype)
	data,err := db.Get(id)
	if err != nil{
		return nil,err
	} else {
		var miner types.Miner
		err = msgpack.Unmarshal(data,&miner)
		if ttype == types.MinerTypeHeavy {
			mm.cache.Add(id,&miner)
		}
		return &miner,err
	}
}

func (mm *MinerManager) RemoveMiner(id []byte, ttype byte) error{
	if ttype == types.MinerTypeHeavy {
		mm.cache.Remove(id)
	}
	db := mm.getMinerDatabase(ttype)
	return db.Delete(id)
}

//返回值：true Abort添加，false 数据不存在或状态不对，Abort失败
func (mm *MinerManager) AbortMiner(id []byte, ttype byte, height uint64) bool{
	miner,_ := mm.GetMinerById(id,ttype)

	if miner != nil && miner.Status == types.MinerStatusNormal{
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height
		if ttype == types.MinerTypeHeavy {
			mm.cache.Remove(id)
		}
		db := mm.getMinerDatabase(ttype)
		data,_ := msgpack.Marshal(miner)
		db.Put(id,data)
		return true
	} else {
		return false
	}
}

func (mm *MinerManager) GetTotalStakeByHeight(height uint64) uint64{
	iter := mm.MinerIterator(types.MinerTypeHeavy)
	var total uint64 = 0
	for miner,_ := iter.Current();iter.Next();miner,_ = iter.Current(){
		if height >= miner.ApplyHeight{
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight{
				total += miner.Stake
			}
		}
	}
	return total
}

func (mm *MinerManager) MinerIterator(ttype byte) *MinerIterator{
	db := mm.getMinerDatabase(ttype)
	iterator := &MinerIterator{iter:db.NewIterator()}
	if ttype == types.MinerTypeHeavy{
		iterator.cache = mm.cache
	}
	return iterator
}

func (mi *MinerIterator) Current() (*types.Miner,error){
	if mi.cache != nil{
		if result, ok := mi.cache.Get(mi.iter.Key()); ok {
			return result.(*types.Miner),nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iter.Value(),&miner)
	return &miner,err
}

func (mi *MinerIterator) Next() bool{
	return mi.iter.Next()
}
