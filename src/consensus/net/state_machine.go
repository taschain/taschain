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
	"network"
	"sync"
	"taslog"
	"common"
	"consensus/model"
	"time"
	"consensus/ticker"
	"fmt"
)


type stateHandleFunc func(msg interface{})

type stateNode struct {
	code    uint32
	repeat  int
	current int
	data    []*StateMsg
	handler stateHandleFunc
	next    *stateNode
}

type StateMsg struct {
	Code uint32
	Data interface{}
	Id 	string
}

type StateMachine struct {
	Id 	string
	Current *stateNode
	Head *stateNode
	Time time.Time
	lock sync.Mutex
}

type StateMachines struct {
	name 	string
	machines sync.Map
	generator StateMachineGenerator
	//machines map[string]*StateMachine
}

var GroupInsideMachines StateMachines
var GroupOutsideMachines StateMachines

var logger taslog.Logger

func InitStateMachines() {
	logger = taslog.GetLoggerByName("state_machine" + common.GlobalConf.GetString("instance", "index", ""))

	GroupInsideMachines = StateMachines{
		name: "GroupInsideMachines",
		generator: &groupInsideMachineGenerator{},
	}

	GroupOutsideMachines = StateMachines{
		name: "GroupOutsideMachines",
		generator: &groupOutsideMachineGenerator{},
	}

	GroupInsideMachines.startCleanRoutine()
	GroupOutsideMachines.startCleanRoutine()
}

func NewStateMsg(code uint32, data interface{}, id string) *StateMsg {
	return &StateMsg{
		Code:code,
		Data:data,
		Id:id,
	}
}

func newStateNode(st uint32, r int, h stateHandleFunc) *stateNode {
	return &stateNode{
		code: st,
		repeat: r,
		data: make([]*StateMsg, 0),
		handler:h,
	}
}

func newStateMachine(id string) *StateMachine {
	return &StateMachine{
		Id: id,
		Time: time.Now(),
	}
}

func (n *stateNode) notEmpty() bool {
    return len(n.data) > 0
}

func (n *stateNode) state() string {
    return fmt.Sprintf("%v[%v/%v]", n.code, n.current, n.repeat)
}

func (n *stateNode) dataIndex(id string) int {
	for idx, d := range n.data {
		if d.Id == id {
			return idx
		}
	}
    return -1
}

func (n *stateNode) addData(stateMsg *StateMsg) (int, bool) {
    idx := n.dataIndex(stateMsg.Id)
	if idx >= 0 {
		return idx, false
	}
	n.data = append(n.data, stateMsg)
	return len(n.data)-1, true
}

func (n *stateNode) finished() bool {
    return n.current >= n.repeat
}


func (m *StateMachine) findTail() *stateNode {
	p := m.Head
	for p != nil && p.next != nil {
		p = p.next
	}
	return p
}

func (m *StateMachine) appendNode(node *stateNode) {
	if node == nil {
		panic("cannot add nil node to the state machine!")
	}

	tail := m.findTail()
	if tail == nil {
		m.Current = node
		m.Head = node
	} else {
		tail.next = node
	}
}


func (m *StateMachine) findNode(code uint32) *stateNode {
    p := m.Head
	for p != nil && p.code != code {
		p = p.next
	}
	return p
}


func (m *StateMachine) finish() bool {
	return m.Current.next == nil && m.Current.finished()
}

func (m *StateMachine) expire() bool {
    return int(time.Since(m.Time).Seconds()) >= model.Param.GroupInitMaxSeconds
}

func (m *StateMachine) transform() {
	node := m.Current

	for node.current < len(node.data) {
		node.handler(node.data[node.current].Data)
		node.data[node.current].Data = true	//释放内存
		node.current++
		logger.Debugf("machine %v handling current state %v", m.Id, node.state())
	}

	if m.Current.finished() && m.Current.next != nil {
		m.Current = m.Current.next
		if len(m.Current.data) > 0 {
			m.transform()
		}
	}
}

func (m *StateMachine) Transform(msg *StateMsg) bool {
	node := m.findNode(msg.Code)
	if node == nil {
		return false
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	defer func() {
		if !m.finish() {
			logger.Debugf("machine %v waiting state %v[%v/%v]", m.Id, m.Current.code, m.Current.current, m.Current.repeat)
		}
	}()

	if m.finish() {
		return false
	}

	if node.code < m.Current.code {	//已经执行过的状态
		logger.Debugf("machine %v handle pre state %v, current state %v", m.Id, node.code, m.Current.state())
		node.handler(msg.Data)
	} else if node.code == m.Current.code {	//进行中的状态
		idx, _ := node.addData(msg)
		if idx < node.current {
			logger.Debugf("machine %v ignore redundant state %v, current state %v", m.Id, node.code, m.Current.state())
			return false
		}
		m.transform()
	} else {	//未来的状态
		logger.Debugf("machine %v cache future state %v, current state %v", m.Id, node.code, m.Current.state())
		node.addData(msg)
	}
	return true
}

type StateMachineGenerator interface {
	Generate(id string) *StateMachine
}

type groupInsideMachineGenerator struct {}
type groupOutsideMachineGenerator struct {}

func (m *groupOutsideMachineGenerator) Generate(id string) *StateMachine {
	machine := newStateMachine(id)
	machine.appendNode(newStateNode(network.GroupInitMsg, 1, func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInit(msg.(*model.ConsensusGroupRawMessage))
	}))
	machine.appendNode(newStateNode(network.GroupInitDoneMsg, model.Param.GetThreshold(), func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInited(msg.(*model.ConsensusGroupInitedMessage))
	}))
	return machine
}

func (m *groupInsideMachineGenerator) Generate(id string) *StateMachine {
	machine := newStateMachine(id)
	machine.appendNode(newStateNode(network.GroupInitMsg, 1, func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInit(msg.(*model.ConsensusGroupRawMessage))
	}))
	machine.appendNode(newStateNode(network.KeyPieceMsg, model.Param.GetGroupMemberNum(), func(msg interface{}) {
		MessageHandler.processor.OnMessageSharePiece(msg.(*model.ConsensusSharePieceMessage))
	}))
	machine.appendNode(newStateNode(network.SignPubkeyMsg, model.Param.GetGroupMemberNum(), func(msg interface{}) {
		MessageHandler.processor.OnMessageSignPK(msg.(*model.ConsensusSignPubKeyMessage))
	}))
	machine.appendNode(newStateNode(network.GroupInitDoneMsg, 1, func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInited(msg.(*model.ConsensusGroupInitedMessage))
	}))
	return machine
}

func (stm *StateMachines) startCleanRoutine()  {
    ticker.GetTickerInstance().RegisterRoutine(stm.name, stm.cleanRoutine, 2)
    ticker.GetTickerInstance().StartTickerRoutine(stm.name, false)
}

func (stm *StateMachines) cleanRoutine() bool {
	stm.machines.Range(func(key, value interface{}) bool {
		m := value.(*StateMachine)
		if m.finish() {
			logger.Infof("%v state machine finished, id=%v", stm.name, m.Id)
			stm.machines.Delete(m.Id)
		}
		if m.expire() {
			logger.Infof("%v state machine expire, id=%v", stm.name, m.Id)
			stm.machines.Delete(m.Id)
		}
		return true
	})
	return true
}


func (stm *StateMachines) GetMachine(id string) *StateMachine {
	if v, ok := stm.machines.Load(id); ok {
		return v.(*StateMachine)
	} else {
		m := stm.generator.Generate(id)
		v, _ = stm.machines.LoadOrStore(id, m)
		return v.(*StateMachine)
	}
}
