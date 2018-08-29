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
	"core/datasource"
	"middleware/types"
	"redis"
	"middleware/notify"
)

const GROUP_STATUS_KEY = "gcurrent"

var GroupChainImpl *GroupChain

type GroupChainConfig struct {
	group string
}

type GroupChain struct {
	config *GroupChainConfig

	// key id, value group
	// key number, value id
	groups tasdb.Database

	// cache
	now [][]byte

	count uint64

	// 读写锁
	lock sync.RWMutex
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

func initGroupChain() error {
	chain := &GroupChain{
		config: getGroupChainConfig(),
		now:    *new([][]byte),
	}

	var err error

	chain.groups, err = datasource.NewDatabase(chain.config.group)
	if nil != err {
		return err
	}

	build(chain)

	GroupChainImpl = chain
	return nil
}

func build(chain *GroupChain) {
	count, _ := chain.groups.Get([]byte(GROUP_STATUS_KEY))
	if nil == count {
		// 创始块
		chain.save(types.GenesisGroup(), false)
		return
	}

	chain.count = binary.BigEndian.Uint64(count)
	var i uint64
	for i = 0; i <= chain.count; i++ {
		groupId, _ := chain.groups.Get(generateKey(i))
		if nil != groupId {
			chain.now = append(chain.now, groupId)
		}

	}

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
	return chain.count - 1
}
func (chain *GroupChain) Close() {
	chain.groups.Close()
}

func (chain *GroupChain) GetMemberPubkeyByID(id []byte) []byte {
	pubKey, _ := redis.GetPubKeyById(id)
	return pubKey
}

func (chain *GroupChain) GetMemberPubkeyByIDs(ids [][]byte) [][]byte {
	result, _ := redis.GetPubKeyByIds(ids)
	return result
}

func (chain *GroupChain) GetCandidates() ([][]byte, error) {
	return redis.GetAllNodeIds()
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

func (chain *GroupChain) AddGroup(group *types.Group, sender []byte, signature []byte) error {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	if nil == group {
		return fmt.Errorf("nil group")
	}

	if !isDebug {
		if nil != group.Parent {
			parent := chain.getGroupById(group.Parent)
			if nil == parent {
				return fmt.Errorf("parent is not existed")
			}
		}
	}

	// 检查是否已经存在
	var flag bool
	if nil != group.Id {
		flag, _ = chain.groups.Has(group.Id)

	} else if nil != group.Dummy {
		flag, _ = chain.groups.Has(group.Dummy)

	}
	// todo: 通过父亲节点公钥校验本组的合法性
	return chain.save(group, flag)
}

func (chain *GroupChain) save(group *types.Group, overWrite bool) error {
	data, err := json.Marshal(group)
	if nil != err {
		return err
	}

	// todo: 半成品组，不能参与铸块
	if group.Id != nil && !overWrite {
		chain.now = append(chain.now, group.Id)
	}

	if group.Dummy != nil {
		chain.groups.Put(generateKey(chain.count), group.Dummy)
		if !overWrite {
			chain.count++
		}
		chain.groups.Put([]byte(GROUP_STATUS_KEY), intToBytes(chain.count))
		fmt.Printf("[group]put dummy succ.count: %d, now:%d, overwrite: %t, dummy id:%x \n", chain.count, len(chain.now), overWrite, group.Dummy)
		return chain.groups.Put(group.Dummy, data)
	} else {
		chain.groups.Put(generateKey(chain.count), group.Id)
		chain.groups.Put([]byte(GROUP_STATUS_KEY), intToBytes(chain.count))
		if !overWrite {
			chain.count++
		}
		fmt.Printf("[group]put real one succ.count: %d, now:%d, overwrite: %t, id:%x \n", chain.count, len(chain.now), overWrite, group.Id)

		notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: *group,})
		return chain.groups.Put(group.Id, data)
	}
}

func (chain *GroupChain) GetAllGroupID() [][]byte {
	return chain.now
}
