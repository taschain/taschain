package core

import (
	"middleware/types"
	"github.com/vmihailenco/msgpack"
	"github.com/hashicorp/golang-lru"
	"common"
	"storage/core/vm"
	"storage/trie"
	"sync"
	"github.com/pkg/errors"
)

var emptyValue [0]byte

const HeavyPrefix  = "heavy"
const LightPrefix  = "light"

type MinerManager struct {
	blockchain *BlockChain
	cache   *lru.Cache
	lock 	sync.Mutex
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

func (mm *MinerManager) getMinerDatabase(ttype byte) (common.Address,string){
	switch ttype {
	case types.MinerTypeLight:
		return  common.LightDBAddress,LightPrefix
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress,HeavyPrefix
	}
	return common.Address{},""
}

//返回值：1成功添加，-1旧数据仍然存在，添加失败
func (mm *MinerManager) AddMiner(id []byte, miner *types.Miner) int {
	db,prefix := mm.getMinerDatabase(miner.Type)

	if mm.blockchain.latestStateDB.GetData(db, string(id)) != nil{
		return -1
	} else {
		data,_ := msgpack.Marshal(miner)
		mm.blockchain.latestStateDB.SetData(db, prefix+string(id), data)
		return 1
	}
}

func (mm *MinerManager) AddGenesesMiner(miners []*types.Miner,accountdb vm.AccountDB) {
	Logger.Infof("MinerManager AddGenesesMiner")
	dbh,preh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl,prel := mm.getMinerDatabase(types.MinerTypeLight)

	for _,miner := range miners{
		if accountdb.GetData(dbh, preh + string(miner.Id)) == nil{
			miner.Type = types.MinerTypeHeavy
			data,_ := msgpack.Marshal(miner)
			accountdb.SetData(dbh, preh + string(miner.Id), data)
			Logger.Debugf("AddGenesesMiner Heavy %+v %+v",miner.Id,data)
		}
		if accountdb.GetData(dbl, prel + string(miner.Id)) == nil{
			miner.Type = types.MinerTypeLight
			data,_ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, prel + string(miner.Id), data)
			Logger.Debugf("AddGenesesMiner Light %+v %+v",miner.Id,data)
		}
	}
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte) *types.Miner {
	if ttype == types.MinerTypeHeavy {
		if result, ok := mm.cache.Get(string(id)); ok {
			return result.(*types.Miner)
		}
	}
	db,prefix := mm.getMinerDatabase(ttype)
	data := mm.blockchain.latestStateDB.GetData(db,prefix + string(id))
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
	db,prefix := mm.getMinerDatabase(ttype)
	mm.blockchain.latestStateDB.SetData(db,prefix+string(id),emptyValue[:])
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
		db,prefix := mm.getMinerDatabase(ttype)
		data,_ := msgpack.Marshal(miner)
		mm.blockchain.latestStateDB.SetData(db,prefix + string(id),data)
		return true
	} else {
		return false
	}
}

func (mm *MinerManager) GetTotalStakeByHeight(height uint64) uint64{
	//header := mm.blockchain.QueryBlockByHeight(height)
	//if header == nil{
	//	return 0
	//}
	//accountdb,_ := core.NewAccountDB(mm.blockchain.latestBlock.StateTree, mm.blockchain.stateCache)
	//if err != nil{
	//	panic(err)
	//}
	//mm.lock.Lock()
	//defer mm.lock.Unlock()
	iter := mm.MinerIterator(types.MinerTypeHeavy,nil)
	var total uint64 = 0
	for iter.Next(){
		miner,_ := iter.Current()
		if height >= miner.ApplyHeight{
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight{
				total += miner.Stake
			}
		}
	}
	if total == 0{
		Logger.Errorf("GetTotalStakeByHeight get 0 %d %s",height,mm.blockchain.latestBlock.StateTree.Hex())
		iter = mm.MinerIterator(types.MinerTypeHeavy,nil)
		for ;iter.Next(); {
			miner, _ := iter.Current()
			Logger.Debugf("GetTotalStakeByHeight %+v",miner)
		}
	} else {
		Logger.Debugf("GetTotalStakeByHeight get %d",total)
	}
	return total
}

func (mm *MinerManager) MinerIterator(ttype byte,accountdb vm.AccountDB) *MinerIterator{
	db,prefix := mm.getMinerDatabase(ttype)
	if accountdb == nil{
		//accountdb,_ = core.NewAccountDB(mm.blockchain.latestBlock.StateTree,mm.blockchain.stateCache)
		accountdb = mm.blockchain.latestStateDB
	}
	iterator := &MinerIterator{iter:accountdb.DataIterator(db,prefix)}
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
	if err != nil {
		Logger.Debugf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iter.Key, err, mi.iter.Value)
	} else {
		Logger.Debugf("MinerIterator Unmarshal Normal %+v %+v %+v", mi.iter.Key, miner, mi.iter.Value)
	}
	if len(miner.Id) == 0{
		err = errors.New("empty miner")
	}
	return &miner,err
}

func (mi *MinerIterator) Next() bool{
	return mi.iter.Next()
}
