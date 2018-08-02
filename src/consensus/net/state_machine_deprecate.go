package net
//
//import (
//	"network"
//	"sync"
//	"taslog"
//	"vm/common/math"
//	"fmt"
//	"common"
//	"consensus/model"
//)
//
//
///**
//** 	1. 每个组内或全网广播的消息是否都会给自己发
//**	2. 消息时序确认, 全网广播的消息是否不需要前序消息先到达? 如收到NewBlock消息,是否不需要Current, verify, verified消息
// */
//const (
//	MTYPE_GROUP = 1
//	MTYPE_BLOCK = 2
//)
//
//type StateMachineTransform interface {
//	Transform(msg *StateMsg1, handlerFunc StateHandleFunc) bool
//}
//
//type StateMachine1 struct {
//	Id 	string
//	Current *StateNode
//	Head *StateNode
//	lock sync.Mutex
//}
//
////type BlockStateMachines struct {
////	height uint64
////	currentMsgNode *StateNode
////	kingMachines map[string]*StateMachine1
////	lock sync.Mutex
////}
//
//type StateMsg1 struct {
//	code uint32
//	msg interface{}
//	sid string
//	key string
//}
//
//type StateHandleFunc func(msg interface{})
//
//type StateNode struct {
//	State   *StateMsg1
//	Handler StateHandleFunc
//	Next    *StateNode
//}
//
//type TimeSequence struct {
//	groupMachines map[string]*StateMachine1
//	//blockMachines map[string]*BlockStateMachines
//	finishCh chan string
//	lock sync.Mutex
//}
//
//var TimeSeq TimeSequence
//
//var logger1 taslog.Logger
//
//func InitStateMachine1() {
//	logger1 = taslog.GetLoggerByName("state_machine" + common.GlobalConf.GetString("instance", "index", ""))
//
//	TimeSeq = TimeSequence{
//		groupMachines: make(map[string]*StateMachine1),
//		//blockMachines: make(map[string]*BlockStateMachines),
//		finishCh: make(chan string),
//	}
//
//	go func() {
//		for {
//			id := <- TimeSeq.finishCh
//			logger.Info("state machine finished, id=", id)
//			TimeSeq.lock.Lock()
//
//			delete(TimeSeq.groupMachines, id)
//			//for _, ms := range TimeSeq.blockMachines {
//			//	delete(ms.kingMachines, id)
//			//}
//			//for gid, ms := range TimeSeq.blockMachines {
//			//	if len(ms.kingMachines) == 0 && ms.currentMsgNode == nil {
//			//		delete(TimeSeq.blockMachines, gid)
//			//		logger.Info("cacscade delete block machines, id=", gid)
//			//	}
//			//}
//
//			TimeSeq.lock.Unlock()
//		}
//	}()
//}
//
//func NewStateMsg1(code uint32, msg interface{}, sid string, key string) *StateMsg1 {
//	return &StateMsg1{
//		code: code,
//		msg: msg,
//		sid: sid,
//		key: key,
//	}
//}
//func newStateNode1(st uint32) *StateNode {
//	return &StateNode{
//		State: NewStateMsg1(st, nil, "", ""),
//	}
//}
//
//func newStateNodeEx(msg *StateMsg1) *StateNode {
//	return &StateNode{
//		State: msg,
//	}
//}
//
//func newStateMachine1(id string) *StateMachine1 {
//	init := newStateNode1(math.MaxUint32)
//	return &StateMachine1{Id: id, Current:init, Head:init}
//}
//
///**
//* @Description: 组创建组外状态机
//* @Param:
//* @return:
//*/
//func newOutsideGroupCreateStateMachine(dummyId string) *StateMachine1 {
//	machine := newStateMachine1(dummyId + "-outsidegroup")
//	machine.addNode(newStateNode1(network.GROUP_INIT_MSG), 1)
//	machine.addNode(newStateNode1(network.GROUP_INIT_DONE_MSG), model.Param.GetThreshold())
//	return machine
//}
//
///**
//* @Description: 组创建组内状态机
//* @Param:
//* @return:
//*/
//func newInsideGroupCreateStateMachine(dummyId string) *StateMachine1 {
//	machine := newStateMachine1(dummyId)
//	machine.addNode(newStateNode1(network.GROUP_INIT_MSG), 1)
//	machine.addNode(newStateNode1(network.KEY_PIECE_MSG), model.Param.GetGroupMemberNum())
//	machine.addNode(newStateNode1(network.SIGN_PUBKEY_MSG), model.Param.GetGroupMemberNum())
//	machine.addNode(newStateNode1(network.GROUP_INIT_DONE_MSG), 1)
//	return machine
//}
//
/////**
////* @Description: 组内某个king铸块状态机
////* @Param:
////* @return:
////*/
////func newBlockCastStateMachine(id string) *StateMachine1 {
////	machine := newStateMachine(id)
////	//machine.addNode(newStateNode(p2p.CURRENT_GROUP_CAST_MSG), 1)
////	machine.addNode(newStateNode(network.CAST_VERIFY_MSG), 1)
////	machine.addNode(newStateNode(network.VARIFIED_CAST_MSG), logical.GetGroupK(logical.GetGroupMemberNum()) - 1)
////	machine.addNode(newStateNode(network.NEW_BLOCK_MSG), 1)
////	return machine
////}
//
//func (m *StateMachine1) Transform(msg *StateMsg1, handleFunc StateHandleFunc) bool {
//	state := newStateNodeEx(msg)
//	state.Handler = handleFunc
//
//	m.lock.Lock()
//	defer m.lock.Unlock()
//	defer func() {
//		if !m.Finish() {
//			logger.Debugf("machine %v wating state %v", m.Id, m.Current.Next.State.code)
//		}
//	}()
//
//	if m.canTransform(state) {	//状态可以转换
//		m.prepareNext(state)
//		m.transform() //执行状态转换
//		return true
//	} else if future, st := m.futureState(state); future {
//		if st == nil {
//			logger.Debugf("machine %v future reducdant state received, ingored! %v", m.Id, state.State)
//		} else {
//			logger.Debugf("machine %v future state received, cached! %v", m.Id, state.State)
//			st.State = state.State // 后续消息先到达,不能转换, 消息先缓存
//			st.Handler = handleFunc
//		}
//		return false
//	} else {
//		logger.Debugf("machine %v prev state received, handle %v", m.Id, state.State)
//		handleFunc(msg.msg) //重复消息或者是某些超过门限后的消息, 怎么处理?
//		return false
//	}
//}
//
////func (bsm *BlockStateMachines) getMachineByKey(key string) *StateMachine1 {
////    bsm.lock.Lock()
////    defer bsm.lock.Unlock()
////	if m, ok := bsm.kingMachines[key]; !ok {
////		m = newBlockCastStateMachine(key)
////		bsm.kingMachines[key] = m
////		return m
////	} else {
////		return m
////	}
////}
////
////func (bsm *BlockStateMachines) setCurrentMsgNode(msg *StateMsg1, handlerFunc StateHandleFunc)  {
////    bsm.lock.Lock()
////    defer bsm.lock.Unlock()
////	bsm.currentMsgNode = newStateNodeEx(msg)
////	bsm.currentMsgNode.Handler = handlerFunc
////}
////
////func (bsm *BlockStateMachines) Transform(msg *StateMsg1, handleFunc StateHandleFunc) bool {
////	if msg.code == network.CURRENT_GROUP_CAST_MSG {
////		if bsm.currentMsgNode == nil {
////			bsm.setCurrentMsgNode(msg, handleFunc)
////			bsm.lock.Lock()
////			defer bsm.lock.Unlock()
////			for _, m := range bsm.kingMachines {
////				m.Transform(msg, handleFunc)
////			}
////		}
////	} else {
////		machine := bsm.getMachineByKey(msg.key)
////
////		if bsm.currentMsgNode != nil {
////			if future := machine.future(bsm.currentMsgNode); future {
////				machine.Transform(bsm.currentMsgNode.State, bsm.currentMsgNode.Handler)
////			}
////		}
////		machine.Transform(msg, handleFunc)
////	}
////	return true
////}
//
//func (m *StateMachine1) future(node *StateNode) bool {
//    m.lock.Lock()
//    defer m.lock.Unlock()
//    if ok, _ := m.futureState(node); ok {
//    	return true
//	}
//	return false
//}
//
//func (m *StateMachine1) futureState(state *StateNode) (bool, *StateNode) {
//	p := m.Head
//	future := false
//	for p != nil && p.State.code != state.State.code {
//		if p == m.Current {
//			future = true
//		}
//		p = p.Next
//	}
//	if p == nil {
//		logger.Warnf("illegal msg found! current state %v, found state %v, msg %v", m.Current.State.code, state.State.code, state.State)
//	}
//	future = future && p != nil
//
//	for p != nil && p.State.code == state.State.code && p.State.msg != nil && p.State.sid != state.State.sid {
//		p = p.Next
//	}
//
//	return future, p
//}
//
//func (m *StateMachine1) canTransform(state *StateNode) bool {
//	if m.Finish() {
//		return false
//	}
//	return state.State.code == m.Current.Next.State.code
//}
//
//func (m *StateMachine1) prepareNext(state *StateNode) {
//	m.Current.Next.State = state.State
//	m.Current.Next.Handler = state.Handler
//}
//
//func (m *StateMachine1) transform() bool {
//	for !m.Finish() && m.Current.Next.State.msg != nil {
//		m.Current = m.Current.Next
//		if m.Current.State.msg != nil {
//			logger.Debugf("machine %v handle state %v %v", m.Id, m.Current.State)
//			m.Current.Handler(m.Current.State.msg)
//		}
//	}
//	return true
//}
//
//func (m *StateMachine1) Finish() bool {
//    ret := m.Current.Next == nil
//	if ret {
//		TimeSeq.finishCh <- m.Id
//	}
//	return ret
//}
//
//func (m *StateMachine1) findTail() *StateNode {
//    p := m.Head
//	for p.Next != nil {
//		p = p.Next
//	}
//	return p
//}
//
//func (m *StateMachine1) addNode(node *StateNode, repeat int) {
//	if node == nil {
//		panic("cannot add nil node to the state machine!")
//	}
//	tail := m.findTail()
//	for repeat > 0 {
//		tmp := *node
//		tail.Next = &tmp
//		tail = &tmp
//		repeat--
//	}
//}
//
//func (this *TimeSequence) GetInsideGroupStateMachine(dummyId string) StateMachineTransform {
//	this.lock.Lock()
//	defer this.lock.Unlock()
//
//	if m, ok := this.groupMachines[dummyId]; ok {
//		return m
//	} else {
//		m = newInsideGroupCreateStateMachine(dummyId)
//		this.groupMachines[dummyId] = m
//		return m
//	}
//
//}
//
//func (this *TimeSequence) GetOutsideGroupStateMachine(dummyId string) StateMachineTransform {
//	this.lock.Lock()
//	defer this.lock.Unlock()
//
//	if m, ok := this.groupMachines[dummyId]; ok {
//		return m
//	} else {
//		m = newOutsideGroupCreateStateMachine(dummyId)
//		this.groupMachines[dummyId] = m
//		return m
//	}
//
//}
//
//func GenerateBlockMachineKey(groupId []byte, height uint64, kingId []byte) string {
//	return fmt.Sprintf("%s-%d-%s", common.Bytes2Hex(groupId), height, common.Bytes2Hex(kingId))
//}
//
////func (this *TimeSequence) GetBlockStateMachine(groupId []byte, height uint64) StateMachineTransform {
////	id := fmt.Sprintf("%s-%d", common.Bytes2Hex(groupId), height)
////	this.lock.Lock()
////	defer this.lock.Unlock()
////
////	if ms, ok := this.blockMachines[id]; ok {
////		return ms
////		//if m, ok2 := ms.kingMachines[key]; ok2 {
////		//	machine = m
////		//} else {
////		//	machine = newBlockCastStateMachine(key)
////		//	ms.kingMachines[key] = machine
////		//}
////	} else {
////		ms = &BlockStateMachines{
////			height: height,
////			kingMachines: make(map[string]*StateMachine1),
////		}
////		//machine = newBlockCastStateMachine(key)
////		//ms.kingMachines[key] = machine
////		this.blockMachines[id] = ms
////		return ms
////	}
////}