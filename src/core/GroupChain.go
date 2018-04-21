package core

import (
	"core/datasource"
	"sync"
	"encoding/json"
	"fmt"
	"vm/common/hexutil"
)

type GroupChainConfig struct {
	group       string
	groupCache  int
	groupHandle int
}

type GroupChain struct {
	config *GroupChainConfig

	// key id, value group
	groups datasource.Database

	// 读写锁
	lock sync.RWMutex

	//已上链的最新块
	latest *Group
}

func defaultGroupChainConfig() *GroupChainConfig {
	return &GroupChainConfig{
		group:       "group",
		groupCache:  128,
		groupHandle: 1024,
	}
}

func NewGroupChain() *GroupChain {
	chain := &GroupChain{
		config: defaultGroupChainConfig(),
	}

	var err error

	chain.groups, err = datasource.NewLDBDatabase(chain.config.group, chain.config.groupCache, chain.config.groupHandle)
	if nil != err {
		return nil
	}

	chain.latest = chain.GetGroupById([]byte(STATUS_KEY))
	return chain
}

func (chain *GroupChain) GetGroupById(id []byte) *Group {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

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

	if nil != chain.latest && hexutil.Encode(chain.latest.Id) != hexutil.Encode(group.Id) {
		return fmt.Errorf("parent not existed")
	}

	error := chain.save(group)
	if nil == error {
		chain.latest = group
	}
	return error
}

func (chain *GroupChain) save(group *Group) error {
	data, err := json.Marshal(group)
	if nil != err {
		return err
	}

	chain.groups.Put([]byte(STATUS_KEY), data)
	return chain.groups.Put(group.Id, data)

}
