package core

import (
	"middleware/types"
	"github.com/vmihailenco/msgpack"
	"github.com/hashicorp/golang-lru"
	"common"
	"storage/core/vm"
	"storage/trie"
	"storage/core"
)

var emptyValue [0]byte

type MinerManager struct {
	blockchain *BlockChain
	cache   *lru.Cache
}

type MinerIterator struct {
	iter 	*trie.Iterator
	cache   *lru.Cache
}

var MinerManagerImpl *MinerManager

func initMinerManager(blockchain *BlockChain) error {
	cache,_ := lru.New(500)
	MinerManagerImpl = &MinerManager{cache:cache,blockchain:blockchain}
	return nil
}

func (mm *MinerManager) getMinerDatabase(ttype byte) common.Address{
	switch ttype {
	case types.MinerTypeLight:
		return  common.LightDBAddress
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress
	}
	return common.Address{}
}

//返回值：1成功添加，-1旧数据仍然存在，添加失败
func (mm *MinerManager) AddMiner(id []byte, miner *types.Miner) int {
	db := mm.getMinerDatabase(miner.Type)

	if mm.blockchain.latestStateDB.GetData(db, string(id)) != nil{
		return -1
	} else {
		data,_ := msgpack.Marshal(miner)
		mm.blockchain.latestStateDB.SetData(db, string(id), data)
		return 1
	}
}

func (mm *MinerManager) AddGenesesMiner(miners []*types.Miner) {
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _,miner := range miners{
		if mm.blockchain.latestStateDB.GetData(dbh, string(miner.Id)) == nil{
			miner.Type = types.MinerTypeHeavy
			data,_ := msgpack.Marshal(miner)
			mm.blockchain.latestStateDB.SetData(dbh, string(miner.Id), data)
		}
		if mm.blockchain.latestStateDB.GetData(dbl, string(miner.Id)) == nil{
			miner.Type = types.MinerTypeLight
			data,_ := msgpack.Marshal(miner)
			mm.blockchain.latestStateDB.SetData(dbl, string(miner.Id), data)
		}
	}
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte) *types.Miner {
	if ttype == types.MinerTypeHeavy {
		if result, ok := mm.cache.Get(string(id)); ok {
			return result.(*types.Miner)
		}
	}
	db := mm.getMinerDatabase(ttype)
	data := mm.blockchain.latestStateDB.GetData(db,string(id))
	if data != nil {
		var miner types.Miner
		msgpack.Unmarshal(data, &miner)
		if ttype == types.MinerTypeHeavy {
			mm.cache.Add(string(id), &miner)
		}
		return &miner
	}
	return nil
}

func (mm *MinerManager) RemoveMiner(id []byte, ttype byte){
	if ttype == types.MinerTypeHeavy {
		mm.cache.Remove(string(id))
	}
	db := mm.getMinerDatabase(ttype)
	mm.blockchain.latestStateDB.SetData(db,string(id),emptyValue[:])
}

//返回值：true Abort添加，false 数据不存在或状态不对，Abort失败
func (mm *MinerManager) AbortMiner(id []byte, ttype byte, height uint64) bool{
	miner := mm.GetMinerById(id,ttype)

	if miner != nil && miner.Status == types.MinerStatusNormal{
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height
		if ttype == types.MinerTypeHeavy {
			mm.cache.Remove(string(id))
		}
		db := mm.getMinerDatabase(ttype)
		data,_ := msgpack.Marshal(miner)
		mm.blockchain.latestStateDB.SetData(db,string(id),data)
		return true
	} else {
		return false
	}
}

func (mm *MinerManager) GetTotalStakeByHeight(height uint64) uint64{
	header := mm.blockchain.QueryBlockByHeight(height)
	if header == nil{
		return 0
	}
	accountdb,err := core.NewAccountDB(header.StateTree, mm.blockchain.stateCache)
	if err != nil{
		panic(err)
	}
	iter := mm.MinerIterator(types.MinerTypeHeavy,accountdb)
	var total uint64 = 0
	for ;iter.Next();{
		miner,_ := iter.Current()
		if height >= miner.ApplyHeight{
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight{
				total += miner.Stake
			}
		}
	}
	return total
}

func (mm *MinerManager) MinerIterator(ttype byte,accountdb vm.AccountDB) *MinerIterator{
	db := mm.getMinerDatabase(ttype)
	if accountdb == nil{
		accountdb = mm.blockchain.latestStateDB
	}
	iterator := &MinerIterator{iter:accountdb.DataIterator(db,"")}
	if ttype == types.MinerTypeHeavy{
		iterator.cache = mm.cache
	}
	return iterator
}

func (mm *MinerManager) RemoveCache(transactions []*types.Transaction){
	for _,tx := range transactions{
		if tx.Type == types.TransactionTypeMinerApply || tx.Type == types.MinerStatusAbort{
			mm.cache.Remove(string(tx.Source[:]))
		}
	}
}

func (mi *MinerIterator) Current() (*types.Miner,error){
	if mi.cache != nil{
		if result, ok := mi.cache.Get(string(mi.iter.Key)); ok {
			return result.(*types.Miner),nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iter.Value,&miner)
	return &miner,err
}

func (mi *MinerIterator) Next() bool{
	return mi.iter.Next()
}