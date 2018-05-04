package core

import "testing"

func TestGroupChain_AddGroup(t *testing.T) {
	ClearGroup(defaultGroupChainConfig())
	initGroupChain()

	first := genesisGroup()
	if nil==GroupChainImpl.getGroupById(first.Dummy){
		t.Fatalf("fail to put genesis")
	}
	if nil!=GroupChainImpl.getGroupById(first.Id){
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

	// 相同id
	group3 := &Group{
		Id:     id2,
		Parent: id1,
	}
	GroupChainImpl.AddGroup(group3, nil, nil)
	if 2 != GroupChainImpl.Count() {
		t.Fatalf("fail to add group2")
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

func TestGroupChain_init(t *testing.T)  {
	group := genesisGroup()
	if nil==group{
		t.Fatalf("fail to genesisGroup")
	}
}
