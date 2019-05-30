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

package net

//
//import (
//	"testing"
//	"fmt"
//	"taslog"
//	"network"
//)
//
//func InputGroupMachine(code uint32, msg string, sourceId string, machine StateMachineTransform)  {
//	machine.Transform(NewStateMsg1(code, msg, sourceId, ""), func(m interface{}) {
//		fmt.Println("msg:", m)
//	})
//}
//func InputBlockMachine(code uint32, msg string, sourceId string, groupId string, height uint64, kingId string, machine StateMachineTransform)  {
//	key := fmt.Sprintf("%s-%d-%s", groupId, height, kingId)
//	machine.Transform(NewStateMsg1(code, msg, sourceId, key), func(m interface{}) {
//		fmt.Println("handle msg:", key, m)
//	})
//}
//func outputMachine(m *StateMachine1) {
//	st := make([]uint32, 0)
//	p := m.Head
//	for p != nil {
//		st = append(st, p.State.code)
//		fmt.Println(p.State)
//		p = p.Next
//	}
//	fmt.Println(st)
//}
//
//func TestTimeSequence_GetStateMachine(t *testing.T) {
//	machine := TimeSeq.GetInsideGroupStateMachine("g1")
//	outputMachine(machine.(*StateMachine1))
//	//outputMachine(machine.(*StateMachine1))
//	t.Log(machine)
//}
//
//func TestStateMachine_GroupMachine(t *testing.T) {
//	machine := TimeSeq.GetInsideGroupStateMachine("m1")
//	InputGroupMachine(network.GROUP_INIT_DONE_MSG, "done 1", "u2", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 5", "u5", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 52", "u5", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 534", "u5", machine)
//	InputGroupMachine(network.KEY_PIECE_MSG, "sharepiece 1", "u1", machine)
//	InputGroupMachine(network.KEY_PIECE_MSG, "sharepiece 2", "u2", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 1", "u1", machine)
//	InputGroupMachine(network.KEY_PIECE_MSG, "sharepiece 5", "u5", machine)
//	InputGroupMachine(network.KEY_PIECE_MSG, "sharepiece 4", "u4", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 4444", "u4", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 2", "u2", machine)
//	InputGroupMachine(network.KEY_PIECE_MSG, "sharepiece 3", "u3", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 3", "u3", machine)
//	InputGroupMachine(network.GROUP_INIT_MSG, "init 1", "u1", machine)
//	InputGroupMachine(network.SIGN_PUBKEY_MSG, "pubkey 42", "u4", machine)
//
//	outputMachine(machine.(*StateMachine1))
//	taslog.Close()
//}
//
////func TestStateMachine_BlockMachine(t *testing.T) {
////	groupId := "block1"
////	height := uint64(1)
////	kingId := "king1"
////	kingId2 := "king2"
////	kingId3 := "king3"
////
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 1", "u1", groupId, height, kingId, machine)
////	//InputBlockMachine(network.CAST_VERIFY_MSG, "cast verify 1", "u1", groupId, height, kingId,  machine)
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 2", "u3", groupId, height, kingId, machine)
////	//InputBlockMachine(network.NEW_BLOCK_MSG, "newblock 1", "u3", groupId, height, kingId2, machine)
////	//InputBlockMachine(network.NEW_BLOCK_MSG, "newblock 1", "u1", groupId, height, kingId, machine)
////	//
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 1", "u4", groupId, height, kingId2, machine)
////	//InputBlockMachine(network.CAST_VERIFY_MSG, "cast verify 1", "u3", groupId, height, kingId2,  machine)
////	//InputBlockMachine(network.CAST_VERIFY_MSG, "cast verify 1", "u5", groupId, height, kingId3,  machine)
////	//InputBlockMachine(network.NEW_BLOCK_MSG, "newblock 1", "u5", groupId, height, kingId3, machine)
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 2", "u6", groupId, height, kingId3, machine)
////	//
////	//InputBlockMachine(network.CURRENT_GROUP_CAST_MSG, "exec 1", "u1", groupId, height, kingId, machine)
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 1", "u5", groupId, height, kingId3, machine)
////	//InputBlockMachine(network.VARIFIED_CAST_MSG, "verified 2", "u3", groupId, height, kingId2, machine)
////	//
////	//taslog.Close()
////}
