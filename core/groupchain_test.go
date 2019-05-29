////   Copyright (C) 2018 TASChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/taslog"
	"testing"
)

//
//import (
//	"testing"
//	"middleware/types"
//	"middleware"
//)
//
//func TestGroupChain_Add(t *testing.T)  {
//	ClearGroup(defaultGroupChainConfig())
//	initGroupChain()
//	middleware.InitMiddleware()
//	id1 := genHash("test1")
//	group1 := &types.Group{
//		Id: id1,
//	}
//	GroupChainImpl.AddGroup(group1, nil, nil)
//
//	if 1 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group1")
//	}
//
//
//
//	id2 := genHash("test2")
//	group2 := &types.Group{
//		Id:     id2,
//		Parent: id1,
//		PreGroup: id1,
//	}
//
//	//if 2 != GroupChainImpl.Height() {
//	//	t.Fatalf("fail to add group2")
//	//}
//
//	id4 := genHash("test3")
//	group4 := &types.Group{
//		Id:     id4,
//		Parent: id1,
//		PreGroup: id2,
//	}
//
//	GroupChainImpl.AddGroup(group4, nil, nil)
//	GroupChainImpl.AddGroup(group2, nil, nil)
//
//	// 相同id，测试覆盖
//	group3 := &types.Group{
//		Id:        id2,
//		Parent:    id1,
//		PreGroup: id2,
//		Signature: []byte{1, 2},
//	}
//	GroupChainImpl.AddGroup(group3, nil, nil)
//	if 3 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group4")
//	}
//}
//
//func TestGroupChain_AddGroup(t *testing.T) {
//	ClearGroup(defaultGroupChainConfig())
//	initGroupChain()
//
//	first := types.GenesisGroup()
//	if nil == GroupChainImpl.getGroupById(first.Dummy) {
//		t.Fatalf("fail to put genesis")
//	}
//	if nil != GroupChainImpl.getGroupById(first.Id) {
//		t.Fatalf("fail to put genesis")
//	}
//
//	id1 := genHash("test1")
//	group1 := &types.Group{
//		Id: id1,
//	}
//	GroupChainImpl.AddGroup(group1, nil, nil)
//	if 1 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group1")
//	}
//
//	id2 := genHash("test2")
//	group2 := &types.Group{
//		Id:     id2,
//		Parent: id1,
//	}
//	GroupChainImpl.AddGroup(group2, nil, nil)
//	if 2 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group2")
//	}
//
//	// 相同id，测试覆盖
//	group3 := &types.Group{
//		Id:        id2,
//		Parent:    id1,
//		Signature: []byte{1, 2},
//	}
//	GroupChainImpl.AddGroup(group3, nil, nil)
//	if 2 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group3")
//	}
//	check := GroupChainImpl.getGroupById(id2)
//	if nil == check || check.Signature == nil || check.Signature[0] != 1 {
//		t.Fatalf("fail to overwrite by id")
//	}
//
//	// 相同Dummy，测试覆盖
//	group4 := &types.Group{
//		Dummy:  []byte{1, 2, 3, 4, 5},
//		Parent: id1,
//	}
//	GroupChainImpl.AddGroup(group4, nil, nil)
//	if 3 != GroupChainImpl.Height() {
//		t.Fatalf("fail to add group4")
//	}
//	group4.Signature = []byte{6, 7}
//	GroupChainImpl.AddGroup(group4, nil, nil)
//	if 3 != GroupChainImpl.Height() {
//		t.Fatalf("fail to overwrite group4")
//	}
//	check = GroupChainImpl.getGroupById([]byte{1, 2, 3, 4, 5})
//	if nil == check || check.Signature == nil || check.Signature[0] != 6 {
//		t.Fatalf("fail to overwrite by dummyid")
//	}
//
//	//now := GroupChainImpl.GetAllGroupID()
//	//if nil == now {
//	//	t.Fatalf("fail to get all groupID")
//	//}
//
//	//fmt.Printf("len now: %d\n",len(now))
//	group := GroupChainImpl.GetGroupById(id2)
//	if nil == group {
//		t.Fatalf("fail to GetGroupById2")
//	}
//
//	group = GroupChainImpl.GetGroupById(id2)
//	if nil == group {
//		t.Fatalf("fail to GetGroupById2")
//	}
//
//}
//
//func TestGroupChain_init(t *testing.T) {
//	group := types.GenesisGroup()
//	if nil == group {
//		t.Fatalf("fail to genesisGroup")
//	}
//}
func TestQueryGroupAfter(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/test9/tas9.ini")
	middleware.InitMiddleware()
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	initGroupChain(&types.GenesisInfo{}, nil)

	//lg := GroupChainImpl.LastGroup()
	//t.Log(lg.GroupHeight, lg.Id)
	chain := GroupChainImpl
	iter := chain.groupsHeight.NewIterator()
	defer iter.Release()

	limit := 100
	for iter.Next() {
		gid := iter.Value()
		g := chain.getGroupById(gid)
		if g != nil {
			t.Log(g.GroupHeight, iter.Key(), g.Id)
			limit--
		}
	}

	//gs := GroupChainImpl.GetGroupsAfterHeight(0, 20)
	//for _, g := range gs {
	//	t.Log(g.GroupHeight, g.Id)
	//}
}
