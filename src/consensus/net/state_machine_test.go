package net

import (
	"testing"
	"network/p2p"
)

func sendMsg(code uint32, msg string)  {

}

func TestStateMachine_Input(t *testing.T) {
	machine := TimeSeq.GetStateMachine("m1", MTYPE_GROUP)
	machine.Input(p2p.KEY_PIECE_MSG, "share piece 1", func(msg interface{}) {
		t.Log("sharepiece", msg)
	})
	machine.Input(p2p.SIGN_PUBKEY_MSG, "pub key 1", func(msg interface{}) {
		t.Log("pubkey", msg)
	})
	machine.Input(p2p.GROUP_INIT_MSG, "init 1", func(msg interface{}) {
		t.Log("init", msg)
	})
	machine.Input(p2p.SIGN_PUBKEY_MSG, "pub key 2", func(msg interface{}) {
		t.Log("pubkey", msg)
	})
	machine.Input(p2p.KEY_PIECE_MSG, "share piece 2", func(msg interface{}) {
		t.Log("sharepiece", msg)
	})
	machine.Input(p2p.KEY_PIECE_MSG, "share piece 3", func(msg interface{}) {
		t.Log("sharepiece", msg)
	})
	machine.Input(p2p.SIGN_PUBKEY_MSG, "pub key 3", func(msg interface{}) {
		t.Log("pubkey", msg)
	})
	machine.Input(p2p.KEY_PIECE_MSG, "share piece 4", func(msg interface{}) {
		t.Log("sharepiece", msg)
	})
	machine.Input(p2p.GROUP_INIT_DONE_MSG, "done 1", func(msg interface{}) {
		t.Log("done", msg)
	})
	machine.Input(p2p.SIGN_PUBKEY_MSG, "pub key 4", func(msg interface{}) {
		t.Log("pubkey", msg)
	})
	machine.Input(p2p.GROUP_INIT_DONE_MSG, "done 2", func(msg interface{}) {
		t.Log("done", msg)
	})
}