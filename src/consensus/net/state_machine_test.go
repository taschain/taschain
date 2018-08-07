package net

import (
	"testing"
	"taslog"
	"log"
)

type TestMachineGenerator struct {

}

func (t *TestMachineGenerator) Generate(id string) *StateMachine {
	machine := newStateMachine(id)
	machine.appendNode(newStateNode(1, 1, func(msg interface{}) {
		log.Println(1, msg)
	}))
	machine.appendNode(newStateNode(2, 4, func(msg interface{}) {
		log.Println(2, msg)
	}))
	machine.appendNode(newStateNode(3, 4, func(msg interface{}) {
		log.Println(3, msg)
	}))
	machine.appendNode(newStateNode(4, 1, func(msg interface{}) {
		log.Println(4, msg)
	}))
	return machine
}

func TestStateMachine_GroupMachine(t *testing.T) {
	logger = taslog.GetLoggerByName("test_machine.log")
	testMachines := StateMachines{
		name: "GroupOutsideMachines",
		generator: &TestMachineGenerator{},
	}

	machine := testMachines.GetMachine("abc")
	machine.Transform(NewStateMsg(2, "sharepiece 1", "u1"))
	machine.Transform(NewStateMsg(2, "sharepiece 2", "u2"))
	machine.Transform(NewStateMsg(2, "sharepiece 4", "u4"))
	machine.Transform(NewStateMsg(2, "sharepiece 5", "u5"))
	machine.Transform(NewStateMsg(2, "sharepiece 3", "u3"))
	machine.Transform(NewStateMsg(3, "pub 4", "u4"))
	machine.Transform(NewStateMsg(3, "pub 3", "u3"))
	machine.Transform(NewStateMsg(1, "init 2", "u4"))
	machine.Transform(NewStateMsg(3, "pub 2", "u2"))
	machine.Transform(NewStateMsg(1, "init 1", "u2"))
	machine.Transform(NewStateMsg(3, "pub 1", "u1"))
	machine.Transform(NewStateMsg(4, "done 1", "u1"))

	log.Println("finished ", machine.finish())

	taslog.Close()
}
