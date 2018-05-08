package core

import "testing"

func TestGroupChain_AddGroup(t *testing.T) {
	ClearGroup(defaultGroupChainConfig())
	initGroupChain()

	first := genesisGroup()
	if nil == GroupChainImpl.getGroupById(first.Dummy) {
		t.Fatalf("fail to put genesis")
	}
	if nil != GroupChainImpl.getGroupById(first.Id) {
		t.Fatalf("fail to put genesis")
	}

	id1 := genHash("test1")
	group1 := &Group{
		Id: id1,
	}
	GroupChainImpl.AddGroup(group1, nil, nil)
	if 1 != GroupChainImpl.Count() {
		t.Fatalf("fail to add group1")
	}

	id2 := genHash("test2")
	group2 := &Group{
		Id:     id2,
		Parent: id1,
	}
	GroupChainImpl.AddGroup(group2, nil, nil)
	if 2 != GroupChainImpl.Count() {
		t.Fatalf("fail to add group2")
	}

	// 相同id，测试覆盖
	group3 := &Group{
		Id:        id2,
		Parent:    id1,
		Signature: []byte{1, 2},
	}
	GroupChainImpl.AddGroup(group3, nil, nil)
	if 2 != GroupChainImpl.Count() {
		t.Fatalf("fail to add group3")
	}
	check := GroupChainImpl.getGroupById(id2)
	if nil == check || check.Signature == nil || check.Signature[0] != 1 {
		t.Fatalf("fail to overwrite by id")
	}

	// 相同Dummy，测试覆盖
	group4 := &Group{
		Dummy:  []byte{1, 2, 3, 4, 5},
		Parent: id1,
	}
	GroupChainImpl.AddGroup(group4, nil, nil)
	if 3 != GroupChainImpl.Count() {
		t.Fatalf("fail to add group4")
	}
	group4.Signature = []byte{6, 7}
	GroupChainImpl.AddGroup(group4, nil, nil)
	if 3 != GroupChainImpl.Count() {
		t.Fatalf("fail to overwrite group4")
	}
	check = GroupChainImpl.getGroupById([]byte{1, 2, 3, 4, 5})
	if nil == check || check.Signature == nil || check.Signature[0] != 6 {
		t.Fatalf("fail to overwrite by dummyid")
	}

	now := GroupChainImpl.GetAllGroupID()
	if nil == now {
		t.Fatalf("fail to get all groupID")
	}

	group := GroupChainImpl.GetGroupById(id2)
	if nil == group {
		t.Fatalf("fail to GetGroupById2")
	}

	group = GroupChainImpl.GetGroupById(id2)
	if nil == group {
		t.Fatalf("fail to GetGroupById2")
	}
}

func TestGroupChain_init(t *testing.T) {
	group := genesisGroup()
	if nil == group {
		t.Fatalf("fail to genesisGroup")
	}
}
