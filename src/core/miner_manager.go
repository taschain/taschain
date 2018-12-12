package core

import (
	"common"
	"consensus/groupsig"
	"errors"
	"github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
	"middleware/types"
	"network"
	"storage/trie"
	"storage/vm"
	"sync"
	"time"
)

var emptyValue [0]byte

/*
**  Creator: Kaede
**  Date: 2018/9/19 下午3:45
**  Description: 在AccountDB上使用特殊地址存储旷工信息，分为重矿工和轻矿工
*/
type MinerManager struct {
	blockchain   BlockChain
	cache        *lru.Cache
	lock         sync.Mutex
	heavyupdate  bool
	heavytrigger *time.Timer
	heavyMiners  []string
}

const heavyTriggerDuration = time.Second * 10

type MinerIterator struct {
	iter  *trie.Iterator
	cache *lru.Cache
}

var MinerManagerImpl *MinerManager

func initMinerManager(blockchain BlockChain) error {
	cache, _ := lru.New(500)
	MinerManagerImpl = &MinerManager{cache: cache, blockchain: blockchain, heavyupdate: true, heavytrigger: time.NewTimer(heavyTriggerDuration), heavyMiners: make([]string, 0)}
	go MinerManagerImpl.loop()
	return nil
}

/*
	重矿工需要构建到FULL_NODE_VIRTUAL_GROUP_ID组，供网络发现使用
 */
func (mm *MinerManager) loop() {
	for {
		<-mm.heavytrigger.C
		if mm.heavyupdate {
			iter := mm.MinerIterator(types.MinerTypeHeavy, nil)
			array := make([]string, 0)
			for iter.Next() {
				miner, _ := iter.Current()
				gid := groupsig.DeserializeId(miner.Id)
				array = append(array, gid.String())
			}
			mm.heavyMiners = array
			network.GetNetInstance().BuildGroupNet(network.FULL_NODE_VIRTUAL_GROUP_ID, array)
			Logger.Infof("MinerManager HeavyMinerUpdate Size:%d",len(array))
			mm.heavyupdate = false
		}
		mm.heavytrigger.Reset(heavyTriggerDuration)
	}
}

func (mm *MinerManager) getMinerDatabase(ttype byte) (common.Address) {
	switch ttype {
	case types.MinerTypeLight:
		return common.LightDBAddress
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress
	}
	return common.Address{}
}

//返回值：1成功添加，-1旧数据仍然存在，添加失败
func (mm *MinerManager) AddMiner(id []byte, miner *types.Miner, accountdb vm.AccountDB) int {
	Logger.Debugf("MinerManager AddMiner %d", miner.Type)
	db := mm.getMinerDatabase(miner.Type)

	latestStateDB := mm.blockchain.LatestStateDB()
	if latestStateDB.GetData(db, string(id)) != nil {
		return -1
	} else {
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		if miner.Type == types.MinerTypeHeavy {
			mm.heavyupdate = true
		}
		return 1
	}
}

func (mm *MinerManager) AddGenesesMiner(miners []*types.Miner, accountdb vm.AccountDB) {
	Logger.Infof("MinerManager AddGenesesMiner")
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _, miner := range miners {
		if accountdb.GetData(dbh, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeHeavy
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbh, string(miner.Id), data)
			mm.heavyMiners = append(mm.heavyMiners, groupsig.DeserializeId(miner.Id).String())
			//Logger.Debugf("AddGenesesMiner Heavy %+v %+v",miner.Id,data)
		}
		if accountdb.GetData(dbl, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, string(miner.Id), data)
			Logger.Debugf("AddGenesesMiner Light %+v %+v", miner.Id, data)
		}
	}
	mm.heavyupdate = true
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner {
	if accountdb == nil && ttype == types.MinerTypeHeavy {
		if result, ok := mm.cache.Get(string(id)); ok {
			return result.(*types.Miner)
		}
	}
	if accountdb == nil {
		accountdb = mm.blockchain.LatestStateDB()
	}
	db := mm.getMinerDatabase(ttype)
	data := accountdb.GetData(db, string(id))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		msgpack.Unmarshal(data, &miner)
		if ttype == types.MinerTypeHeavy {
			mm.cache.Add(string(id), &miner)
		}
		return &miner
	}
	return nil
}

func (mm *MinerManager) RemoveMiner(id []byte, ttype byte, accountdb vm.AccountDB) {
	Logger.Debugf("MinerManager RemoveMiner %d", ttype)
	if ttype == types.MinerTypeHeavy {
		mm.cache.Remove(string(id))
		mm.heavyupdate = true
	}
	db := mm.getMinerDatabase(ttype)
	accountdb.SetData(db, string(id), emptyValue[:])
}

//返回值：true Abort添加，false 数据不存在或状态不对，Abort失败
func (mm *MinerManager) AbortMiner(id []byte, ttype byte, height uint64, accountdb vm.AccountDB) bool {
	miner := mm.GetMinerById(id, ttype, accountdb)

	if miner != nil && miner.Status == types.MinerStatusNormal {
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height

		db := mm.getMinerDatabase(ttype)
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		Logger.Debugf("MinerManager AbortMiner Update Success %+v", miner)
		if ttype == types.MinerTypeHeavy {
			mm.cache.Remove(string(id))
		}
		return true
	} else {
		Logger.Debugf("MinerManager AbortMiner Update Fail %+v", miner)
		return false
	}
}

//获取质押总和
func (mm *MinerManager) GetTotalStakeByHeight(height uint64) uint64 {
	iter := mm.MinerIterator(types.MinerTypeHeavy, nil)
	var total uint64 = 0
	for iter.Next() {
		miner, _ := iter.Current()
		if height >= miner.ApplyHeight {
			if miner.Status == types.MinerStatusNormal || height < miner.AbortHeight {
				total += miner.Stake
			}
		}
	}
	if total == 0 {
		//Logger.Errorf("GetTotalStakeByHeight get 0 %d %s", height, mm.blockchain.(*FullBlockChain).latestBlock.StateTree.Hex())
		iter = mm.MinerIterator(types.MinerTypeHeavy, nil)
		for ; iter.Next(); {
			miner, _ := iter.Current()
			Logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}
	return total
}

func (mm *MinerManager) MinerIterator(ttype byte, accountdb vm.AccountDB) *MinerIterator {
	db := mm.getMinerDatabase(ttype)
	if accountdb == nil {
		//accountdb,_ = core.NewAccountDB(mm.blockchain.latestBlock.StateTree,mm.blockchain.stateCache)
		accountdb = mm.blockchain.LatestStateDB()
	}
	iterator := &MinerIterator{iter: accountdb.DataIterator(db, "")}
	if ttype == types.MinerTypeHeavy {
		iterator.cache = mm.cache
	}
	return iterator
}

func (mm *MinerManager) RemoveCache(transactions []*types.Transaction) {
	for _, tx := range transactions {
		if tx.Type == types.TransactionTypeMinerApply || tx.Type == types.MinerStatusAbort {
			mm.cache.Remove(string(tx.Source[:]))
		}
	}
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	if mi.cache != nil {
		if result, ok := mi.cache.Get(string(mi.iter.Key)); ok {
			return result.(*types.Miner), nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iter.Value, &miner)
	if err != nil {
		Logger.Debugf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iter.Key, err, mi.iter.Value)
	}
	//else {
	//	Logger.Debugf("MinerIterator Unmarshal Normal %+v %+v %+v", mi.iter.Key, miner, mi.iter.Value)
	//}
	if len(miner.Id) == 0 {
		err = errors.New("empty miner")
	}
	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iter.Next()
}

func (mm *MinerManager) GetHeavyMiners() []string {
	return mm.heavyMiners
}
