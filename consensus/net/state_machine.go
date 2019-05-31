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
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/ticker"
	"github.com/taschain/taschain/network"
	"github.com/taschain/taschain/taslog"
	"sync"
	"time"
)

type stateHandleFunc func(msg interface{})

type stateNode struct {
	//state code, unique in a machine
	code uint32

	//The minimum number of repetitions that need to occur
	//in order to transit to the next state
	leastRepeat int32

	//The maximum number of repetitions that would occur
	//at the state
	mostRepeat int32

	//the state transit handler func
	handler stateHandleFunc
	next    *stateNode

	currentIdx int32
	execNum    int32

	//future state msgs cached in the queue
	queue []*StateMsg
	//lock       sync.RWMutex
}

type StateMsg struct {
	Code uint32
	Data interface{}
	Id   string
}

type StateMachine struct {
	Id      string
	Current *stateNode
	//Current atomic.Value
	Head *stateNode
	Time time.Time
	lock sync.Mutex
}

type StateMachines struct {
	name      string
	machines  *lru.Cache
	generator StateMachineGenerator
	ticker    *ticker.GlobalTicker
	//machines map[string]*StateMachine
}

var GroupInsideMachines StateMachines

//var GroupOutsideMachines StateMachines

var logger taslog.Logger

func InitStateMachines() {
	logger = taslog.GetLoggerByIndex(taslog.StateMachineLogConfig, common.GlobalConf.GetString("instance", "index", ""))

	GroupInsideMachines = StateMachines{
		name:      "GroupInsideMachines",
		generator: &groupInsideMachineGenerator{},
		machines:  common.MustNewLRUCache(50),
		ticker:    ticker.NewGlobalTicker("state_machine"),
	}

	GroupInsideMachines.startCleanRoutine()

}

func NewStateMsg(code uint32, data interface{}, id string) *StateMsg {
	return &StateMsg{
		Code: code,
		Data: data,
		Id:   id,
	}
}

func newStateNode(st uint32, lr, mr int, h stateHandleFunc) *stateNode {
	return &stateNode{
		code:        st,
		leastRepeat: int32(lr),
		mostRepeat:  int32(mr),
		queue:       make([]*StateMsg, 0),
		handler:     h,
	}
}

func newStateMachine(id string) *StateMachine {
	return &StateMachine{
		Id:   id,
		Time: time.Now(),
	}
}

func (n *stateNode) queueSize() int32 {
	//n.lock.RLock()
	//defer n.lock.RUnlock()
	return int32(len(n.queue))
}

func (n *stateNode) state() string {
	return fmt.Sprintf("%v[%v/%v]", n.code, n.currentIdx, n.leastRepeat)
}

func (n *stateNode) dataIndex(id string) int32 {
	//n.lock.RLock()
	//defer n.lock.RUnlock()
	for idx, d := range n.queue {
		if d.Id == id {
			return int32(idx)
		}
	}
	return -1
}

func (n *stateNode) addData(stateMsg *StateMsg) (int32, bool) {
	idx := n.dataIndex(stateMsg.Id)
	if idx >= 0 {
		return idx, false
	}
	//n.lock.Lock()
	//defer n.lock.Unlock()
	n.queue = append(n.queue, stateMsg)
	return int32(len(n.queue)) - 1, true
}

func (n *stateNode) leastFinished() bool {
	return n.currentIdx >= n.leastRepeat
}

func (n *stateNode) mostFinished() bool {
	return n.execNum >= n.mostRepeat
}

func (m *StateMachine) findTail() *stateNode {
	p := m.Head
	for p != nil && p.next != nil {
		p = p.next
	}
	return p
}

func (m *StateMachine) currentNode() *stateNode {
	return m.Current
}

func (m *StateMachine) setCurrent(node *stateNode) {
	m.Current = node
}

func (m *StateMachine) appendNode(node *stateNode) {
	if node == nil {
		panic("cannot add nil node to the state machine!")
	}

	tail := m.findTail()
	if tail == nil {
		m.setCurrent(node)
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
	current := m.currentNode()
	return current.next == nil && current.leastFinished()
}

func (m *StateMachine) allFinished() bool {
	for n := m.Head; n != nil; n = n.next {
		if !n.mostFinished() {
			return false
		}
	}
	return true
}

func (m *StateMachine) expire() bool {
	return int(time.Since(m.Time).Seconds()) >= model.Param.GroupInitMaxSeconds
}

func (m *StateMachine) transform() {
	node := m.currentNode()
	qs := node.queueSize()

	//node.lock.Lock()
	d := qs - node.currentIdx
	switch d {
	case 0:
		return
	case 1:
		msg := node.queue[node.currentIdx]
		node.handler(msg.Data)
		node.queue[node.currentIdx].Data = true //释放内存
		node.currentIdx++
		node.execNum++
		logger.Debugf("machine %v handling exec state %v, from %v", m.Id, node.state(), msg.Id)
	default:
		wg := sync.WaitGroup{}
		for node.currentIdx < qs {
			msg := node.queue[node.currentIdx]
			wg.Add(1)
			go func() {
				defer wg.Done()
				node.handler(msg.Data)
				msg.Data = true //释放内存
			}()
			node.currentIdx++
			node.execNum++
			logger.Debugf("machine %v handling exec state %v in parallel, from %v", m.Id, node.state(), msg.Id)
		}
		wg.Wait()
	}

	//node.lock.Unlock()

	if node.leastFinished() && node.next != nil {
		m.setCurrent(node.next)
		m.transform()
	}

}

func (m *StateMachine) Transform(msg *StateMsg) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	defer func() {
		if !m.finish() {
			curr := m.currentNode()
			logger.Debugf("machine %v waiting state %v[%v/%v]", m.Id, curr.code, curr.currentIdx, curr.leastRepeat)
		} else {
			logger.Debugf("machine %v finished", m.Id)
		}
	}()
	node := m.findNode(msg.Code)
	if node == nil {
		return false
	}
	if node.code < m.currentNode().code {
		logger.Debugf("machine %v handle pre state %v, exec state %v", m.Id, node.code, m.currentNode().state())
		node.handler(msg.Data)
		node.execNum++
	} else if node.code > m.currentNode().code {
		logger.Debugf("machine %v cache future state %v from %v, current state %v", m.Id, node.code, msg.Id, m.currentNode().state())
		node.addData(msg)
	} else {
		_, add := node.addData(msg)
		if !add {
			logger.Debugf("machine %v ignore redundant state %v, current state %v", m.Id, node.code, m.currentNode().state())
			return false
		}
		m.transform()
	}
	return true
}

type StateMachineGenerator interface {
	Generate(id string, cnt int) *StateMachine
}

type groupInsideMachineGenerator struct{}

func (m *groupInsideMachineGenerator) Generate(id string, cnt int) *StateMachine {
	machine := newStateMachine(id)
	memNum := cnt
	machine.appendNode(newStateNode(network.GroupInitMsg, 1, 1, func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInit(msg.(*model.ConsensusGroupRawMessage))
	}))
	machine.appendNode(newStateNode(network.KeyPieceMsg, memNum, memNum, func(msg interface{}) {
		MessageHandler.processor.OnMessageSharePiece(msg.(*model.ConsensusSharePieceMessage))
	}))
	machine.appendNode(newStateNode(network.SignPubkeyMsg, 1, memNum, func(msg interface{}) {
		MessageHandler.processor.OnMessageSignPK(msg.(*model.ConsensusSignPubKeyMessage))
	}))
	machine.appendNode(newStateNode(network.GroupInitDoneMsg, model.Param.GetGroupK(memNum), model.Param.GetGroupK(memNum), func(msg interface{}) {
		MessageHandler.processor.OnMessageGroupInited(msg.(*model.ConsensusGroupInitedMessage))
	}))
	return machine
}

func (stm *StateMachines) startCleanRoutine() {
	stm.ticker.RegisterPeriodicRoutine(stm.name, stm.cleanRoutine, 2)
	stm.ticker.StartTickerRoutine(stm.name, false)
}

func (stm *StateMachines) cleanRoutine() bool {
	for _, k := range stm.machines.Keys() {
		id := k.(string)
		value, ok := stm.machines.Get(id)
		if !ok {
			continue
		}
		m := value.(*StateMachine)
		if m.allFinished() {
			logger.Infof("%v state machine allFinished, id=%v", stm.name, m.Id)
			stm.machines.Remove(m.Id)
		}
		if m.expire() {
			logger.Infof("%v state machine expire, id=%v", stm.name, m.Id)
			stm.machines.Remove(m.Id)
		}
	}
	return true
}

func (stm *StateMachines) GetMachine(id string, cnt int) *StateMachine {
	if v, ok := stm.machines.Get(id); ok {
		return v.(*StateMachine)
	} else {
		m := stm.generator.Generate(id, cnt)
		contains, _ := stm.machines.ContainsOrAdd(id, m)
		if !contains {
			return m
		} else {
			if v, ok := stm.machines.Get(id); ok {
				return v.(*StateMachine)
			} else {
				panic("get machine fail, id " + id)
			}
		}
	}
}
