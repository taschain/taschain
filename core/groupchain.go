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
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/tasdb"
	"sync"
)

const groupStatusKey = "gcurrent"

var (
	errGroupExist = errors.New("group exist")
)

var GroupChainImpl *GroupChain

type GroupChainConfig struct {
	dbfile      string
	group       string
	groupHeight string
}

type GroupChain struct {
	config *GroupChainConfig

	// key id, value group
	// key number, value id
	groups       *tasdb.PrefixedDatabase
	groupsHeight *tasdb.PrefixedDatabase

	// 读写锁
	lock sync.RWMutex

	lastGroup      *types.Group
	genesisMembers []string

	topGroups *lru.Cache

	consensusHelper types.ConsensusHelper
}

type GroupIterator struct {
	current *types.Group
}

func (iterator *GroupIterator) Current() *types.Group {
	return iterator.current
}

func (iterator *GroupIterator) MovePre() *types.Group {
	iterator.current = GroupChainImpl.GetGroupByID(iterator.current.Header.PreGroup)
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
		dbfile:      "d_g",
		group:       "gid",
		groupHeight: "gh",
	}
}

func getGroupChainConfig() *GroupChainConfig {
	defaultConfig := defaultGroupChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}
	return &GroupChainConfig{
		dbfile:      common.GlobalConf.GetString(configSec, "db_groups", defaultConfig.dbfile) + common.GlobalConf.GetString("instance", "index", ""),
		group:       defaultConfig.group,
		groupHeight: defaultConfig.groupHeight,
	}
}

func initGroupChain(genesisInfo *types.GenesisInfo, consensusHelper types.ConsensusHelper) error {
	chain := &GroupChain{
		config:          getGroupChainConfig(),
		consensusHelper: consensusHelper,
		topGroups:       common.MustNewLRUCache(10),
		//preCache: new(sync.Map),
	}

	var err error
	ds, err := tasdb.NewDataSource(chain.config.dbfile)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return err
	}
	chain.groups, err = ds.NewPrefixDatabase(chain.config.group)
	if nil != err {
		return err
	}
	chain.groupsHeight, err = ds.NewPrefixDatabase(chain.config.groupHeight)
	if nil != err {
		return err
	}

	build(chain, genesisInfo)

	GroupChainImpl = chain
	return nil
}

func build(chain *GroupChain, genesisInfo *types.GenesisInfo) {
	var lastGroup = chain.loadLastGroup()
	if lastGroup != nil {
		chain.lastGroup = lastGroup
	} else {
		lastGroup = &genesisInfo.Group
		lastGroup.GroupHeight = 0
		e := chain.commitGroup(lastGroup)
		if e != nil {
			panic("Add genesis group on chain failed:" + e.Error())
		}
	}
	genesisGroup := &genesisInfo.Group
	mems := make([]string, 0)
	for _, mem := range genesisGroup.Members {
		mems = append(mems, common.Bytes2Hex(mem))
	}
	chain.genesisMembers = mems
}

func (chain *GroupChain) Height() uint64 {
	if chain.lastGroup == nil {
		return 0
	}
	return chain.lastGroup.GroupHeight
}
func (chain *GroupChain) Close() {
	chain.groups.Close()
}

func (chain *GroupChain) GetGroupsAfterHeight(height uint64, limit int) []*types.Group {
	return chain.getGroupsAfterHeight(height, limit)
}

func (chain *GroupChain) GetGroupByHeight(height uint64) *types.Group {
	return chain.getGroupByHeight(height)
}

func (chain *GroupChain) GetGroupByID(id []byte) *types.Group {
	if v, ok := chain.topGroups.Get(common.Bytes2Hex(id)); ok {
		return v.(*types.Group)
	}
	return chain.getGroupByID(id)
}

func (chain *GroupChain) AddGroup(group *types.Group) (err error) {
	defer func() {
		Logger.Debugf("add group id=%v, groupHeight=%v, err=%v", common.ToHex(group.ID), group.GroupHeight, err)
	}()
	if chain.hasGroup(group.ID) {
		//notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: group,})
		return errGroupExist
	}

	//CheckGroup会调用groupchain的接口，需要在加锁前调用
	ok, err := chain.consensusHelper.CheckGroup(group)
	if !ok {
		if err == common.ErrCreateBlockNil {
			Logger.Infof("Add group failed:  depend on block!")
		}
		return err
	}

	chain.lock.Lock()
	defer chain.lock.Unlock()

	if !bytes.Equal(group.Header.PreGroup, chain.lastGroup.ID) {
		err = fmt.Errorf("preGroup not equal to lastGroup")
		return
	}

	if !chain.hasGroup(group.Header.PreGroup) {
		err = fmt.Errorf("pre group not exist")
		return
	}
	if !chain.hasGroup(group.Header.Parent) {
		err = fmt.Errorf("prarent group not exist")
		return
	}
	group.GroupHeight = chain.lastGroup.GroupHeight + 1

	if err = chain.commitGroup(group); err != nil {
		Logger.Errorf("commit Group fail ,err=%v, height=%v", err, group.GroupHeight)
		return
	}
	chain.topGroups.Add(common.Bytes2Hex(group.ID), group)
	notify.BUS.Publish(notify.GroupAddSucc, &notify.GroupMessage{Group: group})

	return nil
}

func (chain *GroupChain) GenesisMember() map[string]byte {
	mems := make(map[string]byte)
	for _, mem := range chain.genesisMembers {
		mems[mem] = 1
	}
	return mems
}

func (chain *GroupChain) WhetherMemberInActiveGroup(id []byte, currentHeight uint64) bool {
	iter := chain.NewIterator()
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		//解散，后面的组，除了创世组外也都解散了
		if g.Header.DismissedAt(currentHeight) {
			//直接跳到创世组检查
			genisGroup := chain.getGroupByHeight(0)
			if genisGroup.MemberExist(id) {
				return true
			}
			break
		} else { //有效组
			if g.MemberExist(id) {
				return true
			}
		}
	}

	return false
}