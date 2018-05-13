package net

import (
	"network/p2p"
	"consensus/logical"
	"sync"
	"taslog"
)


/**
** 	1. 每个组内或全网广播的消息是否都会给自己发
**	2. 消息时序确认, 全网广播的消息是否不需要前序消息先到达? 如收到NewBlock消息,是否不需要Current, verify, verified消息
 */
const (
	MTYPE_GROUP = 1
	MTYPE_BLOCK = 2
)

type StateMachine struct {
	Id 	string
	Current *StateNode
	Head *StateNode
	lock sync.Mutex
}

type StateTransformFunc func(msg interface{})

type StateNode struct {
	State uint32
	Msg interface{}
	Transform StateTransformFunc
	Next *StateNode
}

type TimeSequence struct {
	machines map[string]*StateMachine
	finishCh chan string
	lock sync.Mutex
}

var TimeSeq TimeSequence

var logger taslog.Logger

func init() {
	logger = taslog.GetLoggerByName("consensus")

	TimeSeq = TimeSequence{
		machines: make(map[string]*StateMachine),
		finishCh: make(chan string),
	}

	go func() {
		for {
			id := <- TimeSeq.finishCh
			delete(TimeSeq.machines, id)
			logger.Info("state machine finished, id=", id)
		}
	}()
}

func newStateNode(st uint32) *StateNode {
	return &StateNode{
		State:st,
	}
}

func newStateMachine(id string) *StateMachine {
	init := newStateNode(-1)
	return &StateMachine{Id: id, Current:init, Head:init}
}

func newGroupCreateStateMachine(dummyId string) *StateMachine {
	machine := newStateMachine(dummyId)
	machine.addNode(newStateNode(p2p.GROUP_INIT_MSG), 1)
	machine.addNode(newStateNode(p2p.KEY_PIECE_MSG), logical.GetGroupMemberNum() - 1)
	machine.addNode(newStateNode(p2p.SIGN_PUBKEY_MSG), logical.GetGroupMemberNum() - 1)
	machine.addNode(newStateNode(p2p.GROUP_INIT_DONE_MSG), logical.GetGroupK() - 1)
	return machine
}

func newBlockCastStateMachine(groupId string) *StateMachine {
	machine := newStateMachine(groupId)
	machine.addNode(newStateNode(p2p.CURRENT_GROUP_CAST_MSG), 1)
	machine.addNode(newStateNode(p2p.CAST_VERIFY_MSG), 1)
	machine.addNode(newStateNode(p2p.VARIFIED_CAST_MSG), logical.GetGroupK() - 1)
	machine.addNode(newStateNode(p2p.NEW_BLOCK_MSG), 1)
	return machine
}

func (m *StateMachine) Input(code uint32, msg interface{}, transformFunc StateTransformFunc) bool {
	state := newStateNode(code)

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.canTransform(state) {	//状态可以转换
		transformFunc(msg)	//先处理本次消息
		m.transformNext()	//执行状态转换
		return true
	} else if future, st := m.futureState(state); future {
		st.Msg = msg	// 后续消息先到达,不能转换, 消息先缓存
		return false
	} else {
		transformFunc(msg)	//重复消息, 怎么处理? 重新处理?
		return false
	}
}

func (m *StateMachine) futureState(state *StateNode) (bool, *StateNode) {
	p := m.Head
	future := false
	for p != nil && p.State != state.State {
		if p == m.Current {
			future = true
		}
		p = p.Next
	}
	return future, p
}

func (m *StateMachine) canTransform(state *StateNode) bool {
	if m.Finish() {
		return false
	}
	return state.State == m.Current.Next.State
}

func (m *StateMachine) transformNext() bool {
	for !m.Finish() {
		m.Current = m.Current.Next
		if m.Current.Msg != nil {
			m.Current.Transform(m.Current.Msg)
		}
	}
	return true
}

func (m *StateMachine) Finish() bool {
    ret := m.Current.Next == nil
	if ret {
		TimeSeq.finishCh <- m.Id
	}
	return ret
}

func (m *StateMachine) addRepeat(tail *StateNode, node *StateNode, repeat int) {
	for repeat > 0 {
		tail.Next = node
		tail = node
		repeat--
	}
}

func (m *StateMachine) findTail() *StateNode {
    p := m.Head
	for p.Next != nil {
		p = p.Next
	}
	return p
}

func (m *StateMachine) addNode(node *StateNode, repeat int) {
	if node == nil {
		panic("cannot add nil node to the state machine!")
	}
	tail := m.findTail()
	if tail == nil {
		m.Head = node
		tail = m.Head
		m.Current = m.Head
		repeat--
	}
	m.addRepeat(tail, node, repeat)

}

func (this *TimeSequence) GetStateMachine(id string, mtype int) *StateMachine {
	this.lock.Lock()
	defer this.lock.Unlock()

	if m, ok := this.machines[id]; ok {
		return m
	} else {
		if mtype == MTYPE_GROUP {
			m = newGroupCreateStateMachine(id)
		} else {
			m = newBlockCastStateMachine(id)
		}
		this.machines[id] = m
		return m
	}
}