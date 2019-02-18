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
	"storage/tasdb"
	"utility"
)

var emptyValue [0]byte

/*
**  Creator: Kaede
**  Date: 2018/9/19 下午3:45
**  Description: 在AccountDB上使用特殊地址存储旷工信息，分为重矿工和轻矿工
*/
type MinerManager struct {
	blockchain   BlockChain
	lock         sync.RWMutex
	heavyupdate  bool
	heavytrigger *time.Timer
	heavyMiners  []string

	heavyMinerCount uint64
	lightMinerCount uint64
	minerCountDB    tasdb.Database
}

type MinerCountOperation struct {
	Code int
}

const (
	heavyTriggerDuration = time.Second * 10
	heavyMinerCountKey   = "heavy_miner_count"
	lightMinerCountKey   = "light_miner_count"
	minerCountPrefix     = "miner_count"
)

var (
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
)

type MinerIterator struct {
	iter  *trie.Iterator
	cache *lru.Cache
}

var MinerManagerImpl *MinerManager

func initMinerManager(blockchain BlockChain) error {
	MinerManagerImpl = &MinerManager{blockchain: blockchain, heavyupdate: true, heavytrigger: time.NewTimer(heavyTriggerDuration), heavyMiners: make([]string, 0), heavyMinerCount: 0, lightMinerCount: 0}
	MinerManagerImpl.lock = sync.RWMutex{}
	var err error
	MinerManagerImpl.minerCountDB, err = tasdb.NewDatabase(minerCountPrefix)
	if err != nil {
		Logger.Errorf("init miner manager error:%s", err.Error())
		return err
	}
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
			Logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(array))
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

	if accountdb.GetData(db, string(id)) != nil {
		return -1
	} else {
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		if miner.Type == types.MinerTypeHeavy {
			mm.heavyupdate = true
		}
		mm.updateMinerCount(miner.Type, minerCountIncrease)
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
			mm.updateMinerCount(types.MinerTypeHeavy, minerCountIncrease)
			//Logger.Debugf("AddGenesesMiner Heavy %+v %+v",miner.Id,data)
		}
		if accountdb.GetData(dbl, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, string(miner.Id), data)
			mm.updateMinerCount(types.MinerTypeLight, minerCountIncrease)
			Logger.Debugf("AddGenesesMiner Light %+v %+v", miner.Id, data)
		}
	}
	mm.heavyupdate = true
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner {
	if accountdb == nil {
		accountdb = mm.blockchain.LatestStateDB()
	}
	db := mm.getMinerDatabase(ttype)
	data := accountdb.GetData(db, string(id))
	if data != nil && len(data) > 0 {
		var miner types.Miner
		msgpack.Unmarshal(data, &miner)
		return &miner
	}
	return nil
}

func (mm *MinerManager) RemoveMiner(id []byte, ttype byte, accountdb vm.AccountDB) {
	Logger.Debugf("MinerManager RemoveMiner %d", ttype)
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
		mm.updateMinerCount(ttype, minerCountDecrease)
		Logger.Debugf("MinerManager AbortMiner Update Success %+v", miner)
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
	return iterator
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

func (mm *MinerManager) updateMinerCount(minerType byte, operation MinerCountOperation) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	if minerType == types.MinerTypeHeavy {
		heavyMinerCountByte, err := mm.minerCountDB.Get([]byte(heavyMinerCountKey))
		if err != nil {
			Logger.Errorf("Miner count db get heavy miner count error:%s", err.Error())
			return
		}
		heavyMinerCount := utility.ByteToUInt64(heavyMinerCountByte)
		if operation == minerCountIncrease {
			heavyMinerCount++
		} else if operation == minerCountDecrease {
			heavyMinerCount--
		}
		err = mm.minerCountDB.Put([]byte(heavyMinerCountKey), utility.UInt64ToByte(heavyMinerCount))
		if err != nil {
			Logger.Errorf("Miner count db put heavy miner count error:%s", err.Error())
			return
		}
		mm.heavyMinerCount = heavyMinerCount
		return
	}

	if minerType == types.MinerTypeLight {
		lightMinerCountByte, err := mm.minerCountDB.Get([]byte(lightMinerCountKey))
		if err != nil {
			Logger.Errorf("Miner count db get light miner count error:%s", err.Error())
			return
		}
		lightMinerCount := utility.ByteToUInt64(lightMinerCountByte)
		if operation == minerCountIncrease {
			lightMinerCount++
		} else if operation == minerCountDecrease {
			lightMinerCount++
		}
		err = mm.minerCountDB.Put([]byte(lightMinerCountKey), utility.UInt64ToByte(lightMinerCount))
		if err != nil {
			Logger.Errorf("Miner count db put light miner count error:%s", err.Error())
			return
		}
		mm.lightMinerCount = lightMinerCount
		return
	}
	Logger.Error("Unknown miner type:%d", minerType)
}

func (mm *MinerManager) HeavyMinerCount() uint64 {
	mm.lock.RLock()
	defer mm.lock.RUnlock()

	if mm.heavyMinerCount != 0 {
		return mm.heavyMinerCount
	}

	heavyMinerCountByte, err := mm.minerCountDB.Get([]byte(heavyMinerCountKey))
	if err != nil {
		Logger.Errorf("Miner count db get heavy miner count error:%s", err.Error())
		return 0
	}
	mm.heavyMinerCount = utility.ByteToUInt64(heavyMinerCountByte)
	return mm.heavyMinerCount
}

func (mm *MinerManager) LightMinerCount() uint64 {
	mm.lock.RLock()
	defer mm.lock.RUnlock()

	if mm.lightMinerCount != 0 {
		return mm.lightMinerCount
	}

	lightMinerCountByte, err := mm.minerCountDB.Get([]byte(lightMinerCountKey))
	if err != nil {
		Logger.Errorf("Miner count db get light miner count error:%s", err.Error())
		return 0
	}
	mm.lightMinerCount = utility.ByteToUInt64(lightMinerCountByte)
	return mm.lightMinerCount
}
