package core

import "testing"

func TestGroupChain_AddGroup(t *testing.T) {
	ClearGroup(defaultGroupChainConfig())
	InitGroupChain()

	id1 := genHash("test1")
	group1 := &Group{
		Id: id1,
	}
	GroupChainImpl.AddGroup(group1)
	if 1 != GroupChainImpl.count {
		t.Fatalf("fail to add group1")
	}

	id2 := genHash("test2")
	group2 := &Group{
		Id:     id2,
		Parent: id1,
	}
	GroupChainImpl.AddGroup(group2)
	if 2 != GroupChainImpl.count {
		t.Fatalf("fail to add group2")
	}

	now := GroupChainImpl.GetAllGroupID()
	if nil == now {
		t.Fatalf("fail to get all groupID")
	}

	group := GroupChainImpl.GetGroupById(id2)
	if nil == group{
		t.Fatalf("fail to GetGroupById2")
	}

	GroupChainImpl.Close()
	InitGroupChain()

	group = GroupChainImpl.GetGroupById(id2)
	if nil == group{
		t.Fatalf("fail to GetGroupById2")
	}
}
