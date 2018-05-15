package net

import (
	"testing"
	"network/p2p"
	"fmt"
	"taslog"
)

func InputGroupMachine(code uint32, msg string, sourceId string, machine StateMachineTransform)  {
	machine.Transform(NewStateMsg(code, msg, sourceId, ""), func(m interface{}) {
		fmt.Println("msg:", m)
	})
}
func InputBlockMachine(code uint32, msg string, sourceId string, groupId string, height uint64, kingId string, machine StateMachineTransform)  {
	key := fmt.Sprintf("%s-%d-%s", groupId, height, kingId)
	machine.Transform(NewStateMsg(code, msg, sourceId, key), func(m interface{}) {
		fmt.Println("handle msg:", key, m)
	})
}
func outputMachine(m *StateMachine) {
	st := make([]uint32, 0)
	p := m.Head
	for p != nil {
		st = append(st, p.State.code)
		fmt.Println(p.State)
		p = p.Next
	}
	fmt.Println(st)
}

func TestTimeSequence_GetStateMachine(t *testing.T) {
	machine := TimeSeq.GetGroupStateMachine("g1")
	outputMachine(machine.(*StateMachine))
	machine = TimeSeq.GetBlockStateMachine([]byte("g2"), 1)
	//outputMachine(machine.(*StateMachine))
	t.Log(machine)
}

func TestStateMachine_GroupMachine(t *testing.T) {
	machine := TimeSeq.GetGroupStateMachine("m1")
	InputGroupMachine(p2p.GROUP_INIT_DONE_MSG, "done 1", "u2", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 5", "u5", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 52", "u5", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 534", "u5", machine)
	InputGroupMachine(p2p.KEY_PIECE_MSG, "sharepiece 1", "u1", machine)
	InputGroupMachine(p2p.KEY_PIECE_MSG, "sharepiece 2", "u2", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 1", "u1", machine)
	InputGroupMachine(p2p.KEY_PIECE_MSG, "sharepiece 5", "u5", machine)
	InputGroupMachine(p2p.KEY_PIECE_MSG, "sharepiece 4", "u4", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 4444", "u4", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 2", "u2", machine)
	InputGroupMachine(p2p.KEY_PIECE_MSG, "sharepiece 3", "u3", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 3", "u3", machine)
	InputGroupMachine(p2p.GROUP_INIT_MSG, "init 1", "u1", machine)
	InputGroupMachine(p2p.SIGN_PUBKEY_MSG, "pubkey 42", "u4", machine)

	outputMachine(machine.(*StateMachine))
	taslog.Close()
}

func TestStateMachine_BlockMachine(t *testing.T) {
	groupId := "block1"
	height := uint64(1)
	kingId := "king1"
	kingId2 := "king2"
	kingId3 := "king3"

	machine := TimeSeq.GetBlockStateMachine([]byte(groupId), height)
	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 1", "u1", groupId, height, kingId, machine)
	InputBlockMachine(p2p.CAST_VERIFY_MSG, "cast verify 1", "u1", groupId, height, kingId,  machine)
	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 2", "u3", groupId, height, kingId, machine)
	InputBlockMachine(p2p.NEW_BLOCK_MSG, "newblock 1", "u3", groupId, height, kingId2, machine)
	InputBlockMachine(p2p.NEW_BLOCK_MSG, "newblock 1", "u1", groupId, height, kingId, machine)

	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 1", "u4", groupId, height, kingId2, machine)
	InputBlockMachine(p2p.CAST_VERIFY_MSG, "cast verify 1", "u3", groupId, height, kingId2,  machine)
	InputBlockMachine(p2p.CAST_VERIFY_MSG, "cast verify 1", "u5", groupId, height, kingId3,  machine)
	InputBlockMachine(p2p.NEW_BLOCK_MSG, "newblock 1", "u5", groupId, height, kingId3, machine)
	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 2", "u6", groupId, height, kingId3, machine)

	InputBlockMachine(p2p.CURRENT_GROUP_CAST_MSG, "current 1", "u1", groupId, height, kingId, machine)
	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 1", "u5", groupId, height, kingId3, machine)
	InputBlockMachine(p2p.VARIFIED_CAST_MSG, "verified 2", "u3", groupId, height, kingId2, machine)

	taslog.Close()
}