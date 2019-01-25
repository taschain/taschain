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

import (
	"testing"
	"taslog"
	"log"
	"github.com/hashicorp/golang-lru"
)

type TestMachineGenerator struct {

}

func (t *TestMachineGenerator) Generate(id string) *StateMachine {
	machine := newStateMachine(id)
	machine.appendNode(newStateNode(1, 1, 1, func(msg interface{}) {
		log.Println(1, msg)
	}))
	machine.appendNode(newStateNode(2, 4, 4, func(msg interface{}) {
		log.Println(2, msg)
	}))
	machine.appendNode(newStateNode(3, 1, 4, func(msg interface{}) {
		log.Println(3, msg)
	}))
	machine.appendNode(newStateNode(4, 1, 1, func(msg interface{}) {
		log.Println(4, msg)
	}))
	return machine
}

func TestStateMachine_GroupMachine(t *testing.T) {
	logger = taslog.GetLoggerByName("test_machine.log")
	cache, err := lru.New(2)
	if err != nil {
		panic("new lru cache fail, err:" + err.Error())
	}
	testMachines := StateMachines{
		name: "GroupOutsideMachines",
		generator: &TestMachineGenerator{},
		machines: cache,
	}

	machine := testMachines.GetMachine("abc")
	machine.Transform(NewStateMsg(2, "sharepiece 1", "u1"))
	machine.Transform(NewStateMsg(2, "sharepiece 2", "u2"))
	machine.Transform(NewStateMsg(2, "sharepiece 4", "u4"))
	machine.Transform(NewStateMsg(4, "done 1", "u1"))
	machine.Transform(NewStateMsg(2, "sharepiece 5", "u5"))
	machine.Transform(NewStateMsg(2, "sharepiece 3", "u3"))
	machine.Transform(NewStateMsg(3, "pub 4", "u4"))
	machine.Transform(NewStateMsg(3, "pub 3", "u3"))
	machine.Transform(NewStateMsg(1, "init 2", "u4"))
	machine.Transform(NewStateMsg(3, "pub 2", "u2"))
	machine.Transform(NewStateMsg(1, "init 1", "u2"))
	machine.Transform(NewStateMsg(3, "pub 1", "u1"))

	log.Println("leastFinished ", machine.finish())
	log.Println("mostFinisehd ", machine.allFinished())

	taslog.Close()
}
