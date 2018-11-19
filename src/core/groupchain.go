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
	"sync"
	"encoding/json"
	"fmt"
	"encoding/binary"
	"os"
	"common"
	"storage/tasdb"
	"middleware/types"
	"middleware/notify"
	"utility"
	"bytes"
	"errors"
)

const GROUP_STATUS_KEY = "gcurrent"
const GROUP_COUNT_KEY = "gcount"

var GroupChainImpl *GroupChain

type GroupChainConfig struct {
	group string
}

type GroupChain struct {
	config *GroupChainConfig

	// key id, value group
	// key number, value id
	groups tasdb.Database

	count uint64

	// 读写锁
	lock sync.RWMutex

	lastGroup *types.Group

	activeGroups []*types.Group

	//preCache map[string]*types.Group
	//preCache *sync.Map
}

type GroupIterator struct {
	current *types.Group
}

func (iterator *GroupIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupIterator) MovePre() *types.Group {
	iterator.current = GroupChainImpl.GetGroupById(iterator.current.PreGroup)
	return iterator.current
}

func (chain *GroupChain) NewIterator() *GroupIterator {
	return &GroupIterator{current: chain.lastGroup}
}

func (chain *GroupChain) LastGroup() *types.Group {
	return chain.lastGroup
}

func defaultGroupChainConfig() *GroupChainConfig {
	return &GroupChainConfig{
		group: "groupldb",
	}
}

func getGroupChainConfig() *GroupChainConfig {
	defaultConfig := defaultGroupChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &GroupChainConfig{
		group: common.GlobalConf.GetString(CONFIG_SEC, "group", defaultConfig.group),
	}
}

func ClearGroup(config *GroupChainConfig) {
	os.RemoveAll("database")
}

func initGroupChain(genesisInfo *types.GenesisInfo) error {
	chain := &GroupChain{
		config: getGroupChainConfig(),
		//preCache: new(sync.Map),
	}

	var err error

	chain.groups, err = tasdb.NewDatabase(chain.config.group)
	if nil != err {
		return err
	}

	build(chain, genesisInfo)

	GroupChainImpl = chain
	return nil
}

func build(chain *GroupChain, genesisInfo *types.GenesisInfo) {
	lastId, _ := chain.groups.Get([]byte(GROUP_STATUS_KEY))
	count, _ := chain.groups.Get([]byte(GROUP_COUNT_KEY))
	var lastGroup *types.Group
	if lastId != nil {
		data, _ := chain.groups.Get(lastId)
		err := json.Unmarshal(data, &lastGroup)
		if err != nil {
			panic("build group Unmarshal fail")
		}
		chain.count = utility.ByteToUInt64(count)
	} else {
		lastGroup = &genesisInfo.Group
		chain.AddGroup(lastGroup, nil, nil)
	}
	chain.lastGroup = lastGroup

	//chain.count = binary.BigEndian.Uint64(count)
	//var i uint64
	//for i = 0; i <= chain.count; i++ {
	//	groupId, _ := chain.groups.Get(generateKey(i))
	//	if nil != groupId {
	//		chain.now = append(chain.now, groupId)
	//	}
	//
	//}
}

func generateKey(i uint64) []byte {
	return intToBytes(i)
}

func intToBytes(n uint64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(n))
	return buf
}

func (chain *GroupChain) Count() uint64 {
	return chain.count
}
func (chain *GroupChain) Close() {
	chain.groups.Close()
}

func (chain *GroupChain) GetGroupsByHeight(height uint64) ([]*types.Group, error) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	if chain.count <= height {
		return nil, fmt.Errorf("exceed local height")
	}

	result := make([]*types.Group, chain.count-height)
	for i := height; i < chain.count; i++ {
		group := chain.getGroupByHeight(i)
		if nil != group {
			result[i-height] = group
		}

	}
	return result, nil
}

func (chain *GroupChain) GetGroupByHeight(height uint64) (*types.Group) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	return chain.getGroupByHeight(height)
}

func (chain *GroupChain) getGroupByHeight(height uint64) *types.Group {
	groupId, _ := chain.groups.Get(generateKey(height))
	if nil != groupId {
		return chain.getGroupById(groupId)
	}

	return nil
}

func (chain *GroupChain) GetSyncGroupsByHeight(height uint64, limit int) ([]*types.Group) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	return chain.getSyncGroupsByHeight(height, limit)
}

func (chain *GroupChain) getSyncGroupsByHeight(height uint64, limit int) []*types.Group {
	result := make([]*types.Group, 0)
	for i := 0; i < limit; i++ {
		groupId, _ := chain.groups.Get(generateKey(height + uint64(i)))
		if nil != groupId {
			result = append(result, chain.getGroupById(groupId))
		} else {
			break
		}
	}

	return result
}

func (chain *GroupChain) GetOtherLackGroups(topId []byte, existIds [][]byte) ([]*types.Group) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()
	return chain.getOtherLackGroups(topId, existIds)
}

func (chain *GroupChain) getOtherLackGroups(topId []byte, existIds [][]byte) ([]*types.Group) {
	result := make([]*types.Group, 0)
	for group := chain.lastGroup; group != nil && !bytes.Equal(topId, group.Id); group = chain.getPreGroup(group) {
		if filt(existIds, group.Id) {
			continue
		} else {
			result = append(result, group)
		}
	}
	return result
}

func (chain *GroupChain) getPreGroup(group *types.Group) *types.Group {
	if group.PreGroup != nil {
		return chain.getGroupById(group.PreGroup)
	} else {
		return nil
	}
}

func filt(existIds [][]byte, target []byte) bool {
	for _, id := range existIds {
		if bytes.Equal(id, target) {
			return true
		}
	}
	return false
}

func (chain *GroupChain) GetGroupById(id []byte) *types.Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.getGroupById(id)
}

func (chain *GroupChain) getGroupById(id []byte) *types.Group {
	data, _ := chain.groups.Get(id)
	if nil == data || 0 == len(data) {
		return nil
	}

	var group *types.Group
	err := json.Unmarshal(data, &group)
	if err != nil {
		return nil
	}
	return group
}

//func (chain *GroupChain) MissingGroupIds() [][]byte{
//	result := make([][]byte,0)
//	chain.preCache.Range(func(key, value interface{}) bool {
//		result = append(result,[]byte(key.(string)))
//		return true
//	})
//
//	return result
//}
//
//func (chain *GroupChain) ExistingGroupIds() [][]byte{
//	result := make([][]byte,0)
//	chain.preCache.Range(func(key, value interface{}) bool {
//		if group,ok := value.(*types.Group);ok{
//			result = append(result,group.Id)
//		}
//
//		return true
//	})
//
//	return result
//}

//func (chain *GroupChain) repairPreGroup(groupId []byte){
//	for{
//		if group,ok := chain.preCache.Load(string(groupId));ok{
//			if group,ok := group.(*types.Group);ok {
//				if nil == chain.save(group) {
//					//chain.lastGroup = group
//					chain.preCache.Delete(string(groupId))
//					groupId = group.Id
//				}
//			}
//		} else {
//			break
//		}
//	}
//}

//func (chain *GroupChain) canOnChain(preGroupId []byte) (bool, []*types.Group){
//	reslut := make([]*types.Group,0)
//	for{
//		if preGroupId == nil{
//			return true,reslut
//		} else if ok,_ := chain.groups.Has(preGroupId);ok{
//			return true,reslut
//		} else {
//			if g,ok := chain.preCache.Load(string(preGroupId));ok{
//				group,_ := g.(*types.Group)
//				reslut = append([]*types.Group{group},reslut...)
//				preGroupId = group.PreGroup
//			} else {
//				return false,reslut
//			}
//		}
//	}
//}

func (chain *GroupChain) AddGroup(group *types.Group, sender []byte, signature []byte) error {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	if nil == group {
		return fmt.Errorf("nil group")
	}
	if logger != nil {
		logger.Debugf("GroupChain AddGroup %+v", group)
	}

	if !isDebug {
		if nil != group.Parent {
			exist, _ := chain.groups.Has(group.Parent)
			//parent := chain.getGroupById(group.Parent)
			//if nil == parent {
			if !exist {
				return fmt.Errorf("parent is not existed")
			}
		}
		if nil != group.PreGroup {
			//exist,_ := chain.groups.Has(group.PreGroup)
			//if !exist{
			//	chain.preCache.Store(string(group.PreGroup), group)
			//	return fmt.Errorf("pre group is not existed")
			//}
			if !bytes.Equal(chain.lastGroup.Id, group.PreGroup) {
				return fmt.Errorf("pre not equal lastgroup")
			}
		} else {
			return chain.save(group)
		}
	}

	// todo: 通过父亲节点公钥校验本组的合法性
	//if nil != group.PreGroup{
	//	can,preGroups := chain.canOnChain(group.PreGroup)
	//	if can {
	//		for _,g := range preGroups{
	//			chain.save(g)
	//		}
	//		return chain.save(group)
	//	} else{
	//		chain.preCache.Store(string(group.PreGroup), group)
	//		return fmt.Errorf("pre group is not existed")
	//	}
	//} else {
	//	return chain.save(group)
	//}
	ret := chain.save(group)
	//chain.lastGroup = group
	//if nil == ret{
	//	chain.repairPreGroup(group.Id)
	//	//if next,ok := chain.preCache[string(group.Id)];ok{
	//	//	chain.AddGroup(next, sender, signature)
	//	//}
	//}
	return ret
}

func (chain *GroupChain) save(group *types.Group) error {
	if nil != group.Id {
		if exist, _ := chain.groups.Has(group.Id); exist {
			notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group,})
			return errors.New("group already exist")
		}
	}

	group.GroupHeight = chain.count
	if logger != nil {
		logger.Debugf("GroupChain save %+v", group)
	}
	data, err := json.Marshal(group)
	if nil != err {
		return err
	}
	chain.groups.Put(generateKey(chain.count), group.Id)
	chain.groups.Put([]byte(GROUP_STATUS_KEY), group.Id)
	chain.count++
	chain.groups.Put([]byte(GROUP_COUNT_KEY), utility.UInt64ToByte(chain.count))

	//fmt.Printf("[group]put real one succ.count: %d, overwrite: %t, id:%x \n", chain.count, overWrite, group.Id)

	err = chain.groups.Put(group.Id, data)
	if nil == err {
		chain.lastGroup = group
		//chain.activeGroups = append(chain.activeGroups, group)
		notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group,})
	}
	return err
}

func (chain *GroupChain) GetSyncGroupsById(id []byte) []*types.Group {
	result := make([]*types.Group, 0)
	group := chain.getGroupById(id)
	if group == nil {
		return result
	}
	return chain.GetSyncGroupsByHeight(group.GroupHeight+1, 5)
}

func (chain *GroupChain) RemoveDismissGroupFromCache(blockHeight uint64) {
	chain.lock.Lock()
	defer chain.lock.Unlock()
	for ; len(chain.activeGroups) > 0; {
		group := chain.activeGroups[0]
		if group.DismissHeight <= blockHeight {
			chain.activeGroups = chain.activeGroups[1:]
		} else {
			break
		}
	}
}

func (chain *GroupChain) WhetherMemberInActiveGroup(id []byte, currentHeight uint64, applyHeight uint64, abortHeight uint64) bool {
	//for _,group := range chain.activeGroups{
	//	for _,member := range group.Members{
	//		if bytes.Equal(member.Id,id){
	//			return true
	//		}
	//	}
	//}

	middle := chain.count / 2
	for i := middle; i < chain.count; i++ {
		group := chain.GetGroupByHeight(i)
		if group.BeginHeight > abortHeight {
			break
		}
		for _, member := range group.Members {
			if bytes.Equal(member.Id, id) {
				return true
			}
		}
	}
	for j := middle - 1; j >= 0; j-- {
		group := chain.GetGroupByHeight(j)
		if group == nil {
			Logger.Debugf("WhetherMemberInActiveGroup Group nil height:%d", j)
			break
		}
		if group.DismissHeight < applyHeight || group.DismissHeight < currentHeight {
			break
		}
		for _, member := range group.Members {
			if bytes.Equal(member.Id, id) {
				return true
			}
		}
	}
	return false
}
