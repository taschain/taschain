package net

import (
	"network/p2p"
	"consensus/logical"
	"sync"
	"taslog"
	"vm/common/math"
	"fmt"
	"common"
)


/**
** 	1. 每个组内或全网广播的消息是否都会给自己发
**	2. 消息时序确认, 全网广播的消息是否不需要前序消息先到达? 如收到NewBlock消息,是否不需要Current, verify, verified消息
 */
const (
	MTYPE_GROUP = 1
	MTYPE_BLOCK = 2
)

type StateMachineTransform interface {
	Transform(msg *StateMsg, transformFunc StateHandleFunc) bool
}

type StateMachine struct {
	Id 	string
	Current *StateNode
	Head *StateNode
	lock sync.Mutex
}

type BlockStateMachines struct {
	height uint64
	currentMsgNode *StateNode
	kingMachines map[string]*StateMachine
}

type StateMsg struct {
	code uint32
	msg interface{}
	sid string
	key string
}

type StateHandleFunc func(msg interface{})

type StateNode struct {
	State   *StateMsg
	Handler StateHandleFunc
	Next    *StateNode
}

type TimeSequence struct {
	groupMachines map[string]*StateMachine
	blockMachines map[string]*BlockStateMachines
	finishCh chan string
	lock sync.Mutex
}

var TimeSeq TimeSequence

var logger taslog.Logger

func init() {
	logger = taslog.GetLoggerByName("consensus")

	TimeSeq = TimeSequence{
		groupMachines: make(map[string]*StateMachine),
		blockMachines: make(map[string]*BlockStateMachines),
		finishCh: make(chan string),
	}

	go func() {
		for {
			id := <- TimeSeq.finishCh
			logger.Info("state machine finished, id=", id)
			delete(TimeSeq.groupMachines, id)
			for _, ms := range TimeSeq.blockMachines {
				delete(ms.kingMachines, id)
			}
			for gid, ms := range TimeSeq.blockMachines {
				if len(ms.kingMachines) == 0 && ms.currentMsgNode == nil {
					delete(TimeSeq.blockMachines, gid)
					logger.Info("cacscade delete block machines, id=", gid)
				}
			}
		}
	}()
}

func NewStateMsg(code uint32, msg interface{}, sid string, key string) *StateMsg {
	return &StateMsg{
		code: code,
		msg: msg,
		sid: sid,
		key: key,
	}
}
func newStateNode(st uint32) *StateNode {
	return &StateNode{
		State: NewStateMsg(st, nil, "", ""),
	}
}

func newStateNodeEx(msg *StateMsg) *StateNode {
	return &StateNode{
		State: msg,
	}
}

func newStateMachine(id string) *StateMachine {
	init := newStateNode(math.MaxUint32)
	return &StateMachine{Id: id, Current:init, Head:init}
}

/** 
* @Description: 组创建组外状态机 
* @Param:  
* @return:  
*/
func newOutsideGroupCreateStateMachine(dummyId string) *StateMachine {
	machine := newStateMachine(dummyId + "-outsidegroup")
	machine.addNode(newStateNode(p2p.GROUP_INIT_MSG), 1)
	machine.addNode(newStateNode(p2p.GROUP_INIT_DONE_MSG), logical.GetGroupK())
	return machine
}

/** 
* @Description: 组创建组内状态机 
* @Param:  
* @return:  
*/ 
func newInsideGroupCreateStateMachine(dummyId string) *StateMachine {
	machine := newStateMachine(dummyId)
	machine.addNode(newStateNode(p2p.GROUP_INIT_MSG), 1)
	machine.addNode(newStateNode(p2p.KEY_PIECE_MSG), logical.GetGroupMemberNum())
	machine.addNode(newStateNode(p2p.SIGN_PUBKEY_MSG), logical.GetGroupMemberNum())
	machine.addNode(newStateNode(p2p.GROUP_INIT_DONE_MSG), 1)
	return machine
}

/** 
* @Description: 组内某个king铸块状态机 
* @Param:  
* @return:
*/ 
func newBlockCastStateMachine(id string) *StateMachine {
	machine := newStateMachine(id)
	machine.addNode(newStateNode(p2p.CURRENT_GROUP_CAST_MSG), 1)
	machine.addNode(newStateNode(p2p.CAST_VERIFY_MSG), 1)
	machine.addNode(newStateNode(p2p.VARIFIED_CAST_MSG), logical.GetGroupK() - 1)
	machine.addNode(newStateNode(p2p.NEW_BLOCK_MSG), 1)
	return machine
}

func (m *StateMachine) Transform(msg *StateMsg, handleFunc StateHandleFunc) bool {
	state := newStateNodeEx(msg)
	state.Handler = handleFunc

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.canTransform(state) {	//状态可以转换
		logger.Debugf("machine %v transform and handle %v", m.Id, state.State)
		m.prepareNext(state)
		m.transform() //执行状态转换
		return true
	} else if future, st := m.futureState(state); future && st != nil {
		logger.Debugf("machine %v future state received, cached %v", m.Id, state.State)
		st.State = state.State // 后续消息先到达,不能转换, 消息先缓存
		st.Handler = handleFunc
		return false
	} else {
		logger.Debugf("machine %v reducdant state, handle %v", m.Id, state.State)
		handleFunc(msg) //重复消息或者是某些超过门限后的消息, 怎么处理?
		return false
	}
}

func (bsm *BlockStateMachines) Transform(msg *StateMsg, handleFunc StateHandleFunc) bool {
	if msg.code == p2p.CURRENT_GROUP_CAST_MSG {
		bsm.currentMsgNode = newStateNodeEx(msg)
		bsm.currentMsgNode.Handler = handleFunc
		for _, m := range bsm.kingMachines {
			m.Transform(msg, handleFunc)
		}
	} else {
		var machine *StateMachine
		if m, ok := bsm.kingMachines[msg.key]; !ok {
			machine = newBlockCastStateMachine(msg.key)
			bsm.kingMachines[msg.key] = machine
		} else {
			machine = m
		}
		if bsm.currentMsgNode != nil {
			if future, _ := machine.futureState(bsm.currentMsgNode); future {
				machine.Transform(bsm.currentMsgNode.State, bsm.currentMsgNode.Handler)
			}
		}
		machine.Transform(msg, handleFunc)
	}
	return true
}

func (m *StateMachine) futureState(state *StateNode) (bool, *StateNode) {
	p := m.Head
	future := false
	for p != nil && p.State.code != state.State.code {
		if p == m.Current {
			future = true
		}
		p = p.Next
	}
	future = future && p != nil

	for p != nil && p.State.code == state.State.code && p.State.msg != nil && p.State.sid != state.State.sid {
		p = p.Next
	}
	if p == nil {
		logger.Warn("illegal msg found! ignored!", state.State.msg, state.State.sid)
	}

	return future, p
}

func (m *StateMachine) canTransform(state *StateNode) bool {
	if m.Finish() {
		return false
	}
	return state.State.code == m.Current.Next.State.code
}

func (m *StateMachine) prepareNext(state *StateNode) {
	m.Current.Next.State = state.State
	m.Current.Next.Handler = state.Handler
}

func (m *StateMachine) transform() bool {
	for !m.Finish() && m.Current.Next.State.msg != nil {
		m.Current = m.Current.Next
		if m.Current.State.msg != nil {
			logger.Debugf("machine %v handle state %v", m.Id, m.Current.State)
			m.Current.Handler(m.Current.State.msg)
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
	for repeat > 0 {
		tmp := *node
		tail.Next = &tmp
		tail = &tmp
		repeat--
	}
}

func (this *TimeSequence) GetInsideGroupStateMachine(dummyId string) StateMachineTransform {
	this.lock.Lock()
	defer this.lock.Unlock()

	if m, ok := this.groupMachines[dummyId]; ok {
		return m
	} else {
		m = newInsideGroupCreateStateMachine(dummyId)
		this.groupMachines[dummyId] = m
		return m
	}

}

func (this *TimeSequence) GetOutsideGroupStateMachine(dummyId string) StateMachineTransform {
	this.lock.Lock()
	defer this.lock.Unlock()

	if m, ok := this.groupMachines[dummyId]; ok {
		return m
	} else {
		m = newOutsideGroupCreateStateMachine(dummyId)
		this.groupMachines[dummyId] = m
		return m
	}

}

func GenerateBlockMachineKey(groupId []byte, height uint64, kingId []byte) string {
	return fmt.Sprintf("%s-%d-%s", common.Bytes2Hex(groupId), height, common.Bytes2Hex(kingId))
}

func (this *TimeSequence) GetBlockStateMachine(groupId []byte, height uint64) StateMachineTransform {
	this.lock.Lock()
	defer this.lock.Unlock()

	id := fmt.Sprintf("%s-%d", common.Bytes2Hex(groupId), height)
	if ms, ok := this.blockMachines[id]; ok {
		return ms
		//if m, ok2 := ms.kingMachines[key]; ok2 {
		//	machine = m
		//} else {
		//	machine = newBlockCastStateMachine(key)
		//	ms.kingMachines[key] = machine
		//}
	} else {
		ms = &BlockStateMachines{
			height: height,
			kingMachines: make(map[string]*StateMachine),
		}
		//machine = newBlockCastStateMachine(key)
		//ms.kingMachines[key] = machine
		this.blockMachines[id] = ms
		return ms
	}
}