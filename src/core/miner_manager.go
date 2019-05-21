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
	"utility"

	"common"
	"consensus/groupsig"
	"middleware/types"
	"network"
	"storage/trie"
	"storage/vm"

	"github.com/hashicorp/golang-lru"
	"github.com/vmihailenco/msgpack"
	"middleware/ticker"
)

const (
	heavyMinerNetTriggerInterval = 10
	heavyMinerCountKey           = "heavy_miner_count"
	lightMinerCountKey           = "light_miner_count"
)

var (
	emptyValue         [0]byte
	minerCountIncrease = MinerCountOperation{0}
	minerCountDecrease = MinerCountOperation{1}
)

type StakeStatus = int

const (
	Staked StakeStatus = iota
	StakeFrozen
)

type StakeFlagByte = byte

const (
	LightStaked StakeFlagByte = (types.MinerTypeLight << 4) | byte(Staked)
	LightStakeFrozen StakeFlagByte = (types.MinerTypeLight << 4) | byte(StakeFrozen)
	HeavyStaked StakeFlagByte = (types.MinerTypeHeavy << 4) | byte(Staked)
	HeavyStakeFrozen StakeFlagByte = (types.MinerTypeHeavy << 4) | byte(StakeFrozen)
)

var MinerManagerImpl *MinerManager

type MinerManager struct {
	hasNewHeavyMiner     bool
	heavyMiners          []string

	ticker *ticker.GlobalTicker
	lock sync.RWMutex
}

type MinerCountOperation struct {
	Code int
}

type MinerIterator struct {
	iterator *trie.Iterator
	cache    *lru.Cache
}

func initMinerManager(ticker *ticker.GlobalTicker) {
	MinerManagerImpl = &MinerManager{
		hasNewHeavyMiner: true,
		heavyMiners: make([]string, 0),
		ticker: ticker,
	}

	MinerManagerImpl.ticker.RegisterPeriodicRoutine("build_virtual_net", MinerManagerImpl.buildVirtualNetRoutine, heavyMinerNetTriggerInterval)
	MinerManagerImpl.ticker.StartTickerRoutine("build_virtual_net", false)
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
	mm.lock.RLock()
	defer mm.lock.RUnlock()
	mems := make([]string, len(mm.heavyMiners))
	copy(mems, mm.heavyMiners)
	return mems
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

func (mm *MinerManager) buildVirtualNetRoutine() bool {
	mm.lock.Lock()
	defer mm.lock.Unlock()
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
	return true
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
		if miner.Stake < common.VerifyStake && miner.Type == types.MinerTypeLight {
			miner.Status = types.MinerStatusAbort
		} else {
			mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
		}
		data, _ := msgpack.Marshal(miner)
		accountdb.SetData(db, string(id), data)
		if miner.Type == types.MinerTypeHeavy {
			mm.hasNewHeavyMiner = true
		}
		return 1
	}
}

func(mm *MinerManager) activateMiner(miner *types.Miner, accountdb vm.AccountDB) {
	db := mm.getMinerDatabase(miner.Type)
	minerData := accountdb.GetData(db, string(miner.Id))
	if minerData == nil || len(minerData) == 0 {
		return
	}
	var dbMiner types.Miner
	err := msgpack.Unmarshal(minerData, &dbMiner)
	if err != nil {
		Logger.Errorf("activateMiner: Unmarshal %d error, ", miner.Id)
		return
	}
	if dbMiner.Stake < common.VerifyStake && miner.Type == types.MinerTypeLight{
		return
	}
	miner.Stake = dbMiner.Stake
	miner.Status = types.MinerStatusNormal
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(miner.Id), data)
	if miner.Type == types.MinerTypeHeavy {
		mm.hasNewHeavyMiner = true
	}
	mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
}

func (mm *MinerManager) addGenesesMiner(miners []*types.Miner, accountdb vm.AccountDB) {
	dbh := mm.getMinerDatabase(types.MinerTypeHeavy)
	dbl := mm.getMinerDatabase(types.MinerTypeLight)

	for _, miner := range miners {
		if accountdb.GetData(dbh, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeHeavy
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbh, string(miner.Id), data)
			mm.AddStakeDetail(miner.Id, miner, miner.Stake, accountdb)
			mm.heavyMiners = append(mm.heavyMiners, groupsig.DeserializeId(miner.Id).String())
			mm.updateMinerCount(types.MinerTypeHeavy, minerCountIncrease, accountdb)
		}
		if accountdb.GetData(dbl, string(miner.Id)) == nil {
			miner.Type = types.MinerTypeLight
			data, _ := msgpack.Marshal(miner)
			accountdb.SetData(dbl, string(miner.Id), data)
			mm.AddStakeDetail(miner.Id, miner, miner.Stake, accountdb)
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
		if ttype == types.MinerTypeHeavy {
			mm.hasNewHeavyMiner = true
		}
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

func (mm *MinerManager) getMinerStakeDetailDatabase() (common.Address) {
	return common.MinerStakeDetailDBAddress
}

func (mm *MinerManager) getDetailDBKey(from []byte, minerAddr []byte, _type byte, status StakeStatus) []byte{
	var pledgFlagByte = (_type << 4) | byte(status)
	key := []byte{StakeFlagByte(pledgFlagByte)}
	key = append(key, minerAddr...)
	key = append(key, from...)
	Logger.Debugf("getDetailDBKey: toHex-> %s", common.ToHex(key))
	return key
}

func (mm *MinerManager) AddStakeDetail(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool{
	dbAddr := mm.getMinerStakeDetailDatabase()
	key := mm.getDetailDBKey(from, miner.Id, miner.Type, Staked)
	detailData := accountdb.GetData(dbAddr, string(key))
	if detailData == nil || len(detailData) == 0 {
		Logger.Debugf("MinerManager.AddStakeDetail set new key: %s, value: %s", common.ToHex(key), common.ToHex(common.Uint64ToByte(amount)))
		accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(amount))
	} else {
		preAmount := common.ByteToUint64(detailData)
		// if overflow
		if preAmount + amount < preAmount {
			Logger.Debug("MinerManager.AddStakeDetail return false(overflow)")
			return false
		}
		Logger.Debugf("MinerManager.AddStakeDetail set key: %s, value: %s", common.ToHex(key), common.ToHex(common.Uint64ToByte(amount)))
		accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(preAmount+amount))
	}
	return true
}

func (mm *MinerManager) CancelStake(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool {
	dbAddr := mm.getMinerStakeDetailDatabase()
	key := mm.getDetailDBKey(from, miner.Id, miner.Type, Staked)
	stakedData := accountdb.GetData(dbAddr, string(key))
	if stakedData == nil || len(stakedData) == 0 {
		Logger.Debug("MinerManager.CancelStake  false(cannot find stake data)")
		return false
	} else {
		preStake := common.ByteToUint64(stakedData)
		frozenKey := mm.getDetailDBKey(from, miner.Id, miner.Type, StakeFrozen)
		frozenData := accountdb.GetData(dbAddr, string(frozenKey))
		var preFrozen, newFrozen, newStake uint64
		if frozenData == nil || len(frozenData) == 0 {
			preFrozen = 0
		} else {
			preFrozen = common.ByteToUint64(frozenData[:8])
		}
		newStake = preStake - amount
		newFrozen = preFrozen + amount
		if preStake < amount || newFrozen < preFrozen{
			//preStake: 200000000000, preFrozen: 0, newStake: 18446743973709551616, newFrozen: 300000000000
			Logger.Debugf("MinerManager.CancelStake return false(overflow or not enough staked: preStake: %d, " +
				"preFrozen: %d, newStake: %d, newFrozen: %d)", preStake, preFrozen, newStake, newFrozen)
			return false
		}
		if newStake == 0 {
			accountdb.RemoveData(dbAddr, string(key))
		} else {
			accountdb.SetData(dbAddr, string(key), common.Uint64ToByte(newStake))
		}
		newFrozenData := common.Uint64ToByte(newFrozen)
		newFrozenData = append(newFrozenData, common.Uint64ToByte(height)...)
		accountdb.SetData(dbAddr, string(frozenKey), newFrozenData)
		Logger.Debugf("MinerManager.CancelStake success from:%s, to: %s, value: %d ", common.ToHex(from), common.ToHex(miner.Id), amount)
		return true
	}
}

func (mm MinerManager) GetLatestCancelStakeHeight(from []byte, miner *types.Miner, accountdb vm.AccountDB) uint64{
	dbAddr := mm.getMinerStakeDetailDatabase()
	frozenKey := mm.getDetailDBKey(from, miner.Id, miner.Type, StakeFrozen)
	frozenData := accountdb.GetData(dbAddr, string(frozenKey))
	if frozenData == nil || len(frozenData) == 0{
		return common.MaxUint64
	} else {
		return common.ByteToUint64(frozenData[8:])
	}
}

func (mm *MinerManager) RefundStake(from []byte, miner *types.Miner, accountdb vm.AccountDB) (uint64, bool) {
	dbAddr := mm.getMinerStakeDetailDatabase()
	frozenKey := mm.getDetailDBKey(from, miner.Id, miner.Type, StakeFrozen)
	frozenData := accountdb.GetData(dbAddr, string(frozenKey))
	if frozenData == nil || len(frozenData) == 0 {
		Logger.Debug("MinerManager.RefundStake return false(cannot find frozen data)")
		return 0, false
	} else {
		preFrozen := common.ByteToUint64(frozenData[:8])
		accountdb.RemoveData(dbAddr, string(frozenKey))
		return preFrozen, true
	}
}


func (mm *MinerManager) AddStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool {
	Logger.Debugf("Miner manager addStake, minerid: %d", miner.Id)
	db := mm.getMinerDatabase(miner.Type)
	miner.Stake += amount
	if miner.Stake < amount {
		Logger.Debug("MinerManager.AddStake return false (overflow)")
		return false
	}
	if miner.Status == types.MinerStatusAbort &&
		miner.Type == types.MinerTypeLight &&
		miner.Stake >= common.VerifyStake {
		miner.Status = types.MinerStatusNormal
		mm.updateMinerCount(miner.Type, minerCountIncrease, accountdb)
	}
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(id), data)
	return true
}

func(mm *MinerManager) ReduceStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool {
	Logger.Debugf("Miner manager reduceStake, minerid: %d", miner.Id)
	db := mm.getMinerDatabase(miner.Type)
	if miner.Stake < amount {
		return false
	}
	miner.Stake -= amount
	if miner.Status == types.MinerStatusNormal &&
		miner.Type == types.MinerTypeLight &&
		miner.Stake < common.VerifyStake {
		if GroupChainImpl.WhetherMemberInActiveGroup(id, height) {
			Logger.Debugf("TVMExecutor Execute MinerRefund Light Fail(Still In Active Group) %s", common.ToHex(id))
			return false
		}
		miner.Status = types.MinerStatusAbort
		mm.updateMinerCount(miner.Type, minerCountDecrease, accountdb)
	}
	data, _ := msgpack.Marshal(miner)
	accountdb.SetData(db, string(id), data)
	return true
}


func(mm *MinerManager) Transaction2MinerParams(tx *types.Transaction) ( _type byte, id []byte, value uint64){
	data := common.FromHex(string(tx.Data))
	if len(data) == 0 {
		return
	}
	_type = data[0]
	if len(data) < common.AddressLength+1 {
		return
	}
	id = data[1:common.AddressLength+1]
	if len(data) > common.AddressLength+1 {
		value = common.ByteToUint64(data[common.AddressLength+1:])
	}
	return
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

func (mi *MinerManager) Transaction2Miner(tx *types.Transaction) *types.Miner {
	data := common.FromHex(string(tx.Data))
	var miner types.Miner
	msgpack.Unmarshal(data, &miner)
	return &miner
}