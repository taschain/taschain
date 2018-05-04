package core

import (
	"sync"
	"encoding/json"
	"fmt"
	"encoding/binary"
	"os"
	"common"
	"vm/ethdb"
	"core/datasource"
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
	groups ethdb.Database

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
		chain.save(genesisGroup())
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

func (chain *GroupChain) GetGroupsByHeight(height uint64, currentHash common.Hash) ([]*Group, error) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	if chain.count <= height {
		return nil, fmt.Errorf("exceed local height")
	}

	// todo: 校验currentHash

	result := make([]*Group, chain.count-height)
	for i := height; i < chain.count; i++ {
		group := chain.getGroupByHeight(i)
		if nil != group {
			result[i-height] = group
		}

	}
	return result, nil
}

func (chain *GroupChain) getGroupByHeight(height uint64) *Group {
	groupId, _ := chain.groups.Get(generateKey(height))
	if nil != groupId {
		return chain.getGroupById(groupId)
	}

	return nil
}

func (chain *GroupChain) GetGroupById(id []byte) *Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.getGroupById(id)
}

func (chain *GroupChain) getGroupById(id []byte) *Group {
	data, _ := chain.groups.Get(id)
	if nil == data || 0 == len(data) {
		return nil
	}

	var group *Group
	err := json.Unmarshal(data, &group)
	if err != nil {
		return nil
	}
	return group
}

func (chain *GroupChain) AddGroup(group *Group, sender []byte, signature []byte) error {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	if nil == group {
		return fmt.Errorf("nil group")
	}

	if nil != group.Parent {
		parent := chain.getGroupById(group.Parent)
		if nil == parent {
			return fmt.Errorf("parent is not existed")
		}
	}

	// todo: 通过父亲节点公钥校验本组的合法性

	return chain.save(group)
}

func (chain *GroupChain) save(group *Group) error {
	data, err := json.Marshal(group)
	if nil != err {
		return err
	}

	// todo: 半成品组，不能参与铸块
	if group.Id != nil {
		chain.now = append(chain.now, group.Id)
	}

	chain.groups.Put(generateKey(chain.count), group.Id)
	chain.groups.Put([]byte(GROUP_STATUS_KEY), intToBytes(chain.count))
	chain.count++
	return chain.groups.Put(group.Id, data)

}

func (chain *GroupChain) GetAllGroupID() [][]byte {
	return chain.now
}
