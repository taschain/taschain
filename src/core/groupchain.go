package core

import (
	"core/datasource"
	"sync"
	"encoding/json"
	"fmt"
	"bytes"
	"encoding/binary"
	"os"
)

var GroupChainImpl *GroupChain

const PREFIX = "p"

type GroupChainConfig struct {
	group       string
	groupCache  int
	groupHandle int
}

type GroupChain struct {
	config *GroupChainConfig

	// key id, value group
	// key number, value id
	groups datasource.Database

	// cache
	now [][]byte

	count uint64

	// 读写锁
	lock sync.RWMutex
}

func defaultGroupChainConfig() *GroupChainConfig {
	return &GroupChainConfig{
		group:       "groupldb",
		groupCache:  10,
		groupHandle: 10,
	}
}

func ClearGroup(config *GroupChainConfig) {
	os.RemoveAll(config.group)
}

func initGroupChain() error {
	chain := &GroupChain{
		config: defaultGroupChainConfig(),
		now:    *new([][]byte),
	}

	var err error

	chain.groups, err = datasource.NewLDBDatabase(chain.config.group, chain.config.groupCache, chain.config.groupHandle)
	if nil != err {
		return err
	}

	buildCache(chain)

	GroupChainImpl = chain
	return nil
}

func buildCache(chain *GroupChain) {
	count, _ := chain.groups.Get([]byte(STATUS_KEY))
	if nil == count {
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
	bytesBuffer := bytes.NewBuffer([]byte(PREFIX))
	bytesBuffer.Write(intToBytes(i))
	return bytesBuffer.Bytes()
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

func (chain *GroupChain) AddGroup(group *Group) error {
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
	chain.groups.Put([]byte(STATUS_KEY), intToBytes(chain.count))
	chain.count++
	return chain.groups.Put(group.Id, data)

}

func (chain *GroupChain) GetAllGroupID() [][]byte {
	return chain.now
}
