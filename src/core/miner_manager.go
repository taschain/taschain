//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.
package core

import (
	"errors"
	"sync"
	"time"
	"utility"

	"common"
	"network"
	"storage/vm"
	"storage/trie"
	"middleware/types"
	"consensus/groupsig"

	"github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
)

const (
	heavyMinerNetTriggerInterval = time.Second * 10
	heavyMinerCountKey           = "heavy_miner_count"
	lightMinerCountKey           = "light_miner_count"
)

var (
	emptyValue         [0]byte
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
)

var MinerManagerImpl *MinerManager

type MinerManager struct {
	hasNewHeavyMiner     bool
	heavyMiners          []string
	heavyMinerNetTrigger *time.Timer

	lock sync.RWMutex
}

type MinerCountOperation struct {
	Code int
}

type MinerIterator struct {
	iterator *trie.Iterator
	cache    *lru.Cache
}

func initMinerManager() {
	MinerManagerImpl = &MinerManager{hasNewHeavyMiner: true, heavyMinerNetTrigger: time.NewTimer(heavyMinerNetTriggerInterval), heavyMiners: make([]string, 0)}
	MinerManagerImpl.lock = sync.RWMutex{}
	go MinerManagerImpl.loop()
}

func (mm *MinerManager) GetMinerById(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner {
	if accountdb == nil {
		accountdb = BlockChainImpl.LatestStateDB()
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

func (mm *MinerManager) GetTotalStake(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Errorf("Get account db by height %d error:%s", height, err.Error())
		return 0
	}

	iter := mm.minerIterator(types.MinerTypeHeavy, accountDB)
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
		iter = mm.minerIterator(types.MinerTypeHeavy, accountDB)
		for ; iter.Next(); {
			miner, _ := iter.Current()
			Logger.Debugf("GetTotalStakeByHeight %+v", miner)
		}
	}
	return total
}

func (mm *MinerManager) GetHeavyMiners() []string {
	return mm.heavyMiners
}

func (mm *MinerManager) MinerIterator(minerType byte, height uint64) *MinerIterator {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return nil
	}
	return mm.minerIterator(minerType, accountDB)
}

func (mm *MinerManager) HeavyMinerCount(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	heavyMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, heavyMinerCountKey)
	return utility.ByteToUInt64(heavyMinerCountByte)

}

func (mm *MinerManager) LightMinerCount(height uint64) uint64 {
	accountDB, err := BlockChainImpl.GetAccountDBByHeight(height)
	if err != nil {
		Logger.Error("Get account db by height %d error:%s", height, err.Error())
		return 0
	}
	lightMinerCountByte := accountDB.GetData(common.MinerCountDBAddress, lightMinerCountKey)
	return utility.ByteToUInt64(lightMinerCountByte)
}

func (mm *MinerManager) loop() {
	for {
		<-mm.heavyMinerNetTrigger.C
		if mm.hasNewHeavyMiner {
			iterator := mm.minerIterator(types.MinerTypeHeavy, nil)
			array := make([]string, 0)
			for iterator.Next() {
				miner, _ := iterator.Current()
				gid := groupsig.DeserializeId(miner.Id)
				array = append(array, gid.String())
			}
			mm.heavyMiners = array
			network.GetNetInstance().BuildGroupNet(network.FULL_NODE_VIRTUAL_GROUP_ID, array)
			Logger.Infof("MinerManager HeavyMinerUpdate Size:%d", len(array))
			mm.hasNewHeavyMiner = false
		}
		mm.heavyMinerNetTrigger.Reset(heavyMinerNetTriggerInterval)
	}
}

func (mm *MinerManager) getMinerDatabase(minerType byte) (common.Address) {
	switch minerType {
	case types.MinerTypeLight:
		return common.LightDBAddress
	case types.MinerTypeHeavy:
		return common.HeavyDBAddress
	}
	return common.Address{}
}

func (mm *MinerManager) addMiner(id []byte, miner *types.Miner, accountdb vm.AccountDB) int {
	Logger.Debugf("Miner manager add miner %d", miner.Type)
	db := mm.getMinerDatabase(miner.Type)

	if accountdb.GetData(db, string(id)) != nil {
		return -1
	} else {
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		if miner.Type == types.MinerTypeHeavy {
			mm.hasNewHeavyMiner = true
		}
		mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
		return 1
	}
}

func (mm *MinerManager) addGenesesMiner(miners []*types.Miner, accountdb vm.AccountDB) {
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _, miner := range miners {
		if accountdb.GetData(dbh, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeHeavy
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbh, string(miner.Id), data)
			mm.heavyMiners = append(mm.heavyMiners, groupsig.DeserializeId(miner.Id).String())
			mm.updateMinerCount(types.MinerTypeHeavy, minerCountIncrease, accountdb)
		}
		if accountdb.GetData(dbl, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, string(miner.Id), data)
			mm.updateMinerCount(types.MinerTypeLight, minerCountIncrease, accountdb)
		}
	}
	mm.hasNewHeavyMiner = true
}

func (mm *MinerManager) removeMiner(id []byte, ttype byte, accountdb vm.AccountDB) {
	Logger.Debugf("Miner manager remove miner %d", ttype)
	db := mm.getMinerDatabase(ttype)
	accountdb.SetData(db, string(id), emptyValue[:])
}

func (mm *MinerManager) abortMiner(id []byte, ttype byte, height uint64, accountdb vm.AccountDB) bool {
	miner := mm.GetMinerById(id, ttype, accountdb)
	if miner != nil && miner.Status == types.MinerStatusNormal {
		miner.Status = types.MinerStatusAbort
		miner.AbortHeight = height

		db := mm.getMinerDatabase(ttype)
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		mm.updateMinerCount(ttype, minerCountDecrease, accountdb)
		Logger.Debugf("Miner manager abort miner update success %+v", miner)
		return true
	} else {
		Logger.Debugf("Miner manager abort miner update fail %+v", miner)
		return false
	}
}

func (mm *MinerManager) minerIterator(minerType byte, accountdb vm.AccountDB) *MinerIterator {
	db := mm.getMinerDatabase(minerType)
	if accountdb == nil {
		accountdb = BlockChainImpl.LatestStateDB()
	}
	iterator := &MinerIterator{iterator: accountdb.DataIterator(db, "")}
	return iterator
}

func (mm *MinerManager) updateMinerCount(minerType byte, operation MinerCountOperation, accountdb vm.AccountDB) {
	if minerType == types.MinerTypeHeavy {
		heavyMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, heavyMinerCountKey)
		heavyMinerCount := utility.ByteToUInt64(heavyMinerCountByte)
		if operation == minerCountIncrease {
			heavyMinerCount++
		} else if operation == minerCountDecrease {
			heavyMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, heavyMinerCountKey, utility.UInt64ToByte(heavyMinerCount))
		return
	}

	if minerType == types.MinerTypeLight {
		lightMinerCountByte := accountdb.GetData(common.MinerCountDBAddress, lightMinerCountKey)
		lightMinerCount := utility.ByteToUInt64(lightMinerCountByte)
		if operation == minerCountIncrease {
			lightMinerCount++
		} else if operation == minerCountDecrease {
			lightMinerCount--
		}
		accountdb.SetData(common.MinerCountDBAddress, lightMinerCountKey, utility.UInt64ToByte(lightMinerCount))
		return
	}
	Logger.Error("Unknown miner type:%d", minerType)
}

func (mi *MinerIterator) Current() (*types.Miner, error) {
	if mi.cache != nil {
		if result, ok := mi.cache.Get(string(mi.iterator.Key)); ok {
			return result.(*types.Miner), nil
		}
	}
	var miner types.Miner
	err := msgpack.Unmarshal(mi.iterator.Value, &miner)
	if err != nil {
		Logger.Debugf("MinerIterator Unmarshal Error %+v %+v %+v", mi.iterator.Key, err, mi.iterator.Value)
	}

	if len(miner.Id) == 0 {
		err = errors.New("empty miner")
	}
	return &miner, err
}

func (mi *MinerIterator) Next() bool {
	return mi.iterator.Next()
}
