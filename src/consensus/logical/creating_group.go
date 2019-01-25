package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"strconv"
	"sync"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/25 下午12:14
**  Description:
 */

const (
	PIECE_GROUP_NOTFOUND int8 = 0
	PIECE_NORMAL              = 1
	PIECE_THRESHOLD           = 2
	PIECE_DENY_RECOVERED      = 3
	PIECE_DENY_DUP            = 4
)

func PIECE_RESULT(ret int8) string {
	switch ret {
	case PIECE_GROUP_NOTFOUND:
		return "找不到组信息"
	case PIECE_NORMAL:
		return "正常签名分片"
	case PIECE_THRESHOLD:
		return "收到阈值个分片"
	case PIECE_DENY_RECOVERED:
		return "已恢复出组签名，拒绝分片"
	case PIECE_DENY_DUP:
		return "重复分片"
	default:
		return strconv.FormatInt(int64(ret), 10)
	}
}

type CreatingGroup struct {
	gInfo 			*model.ConsensusGroupInitInfo
	gSignGenerator *model.GroupSignGenerator
}

type CreatingGroups struct {
	groups sync.Map //string -> *CreatingGroup
	sentInitGroupHashs [10]common.Hash //已经发送init消息的组hash
	currIndex  int
	lock sync.RWMutex
}

func newCreateGroup(gInfo *model.ConsensusGroupInitInfo, threshold int) *CreatingGroup {
	cg := &CreatingGroup{
		gInfo: gInfo,
		//createGroup:    creator,
		gSignGenerator: model.NewGroupSignGenerator(threshold),
	}
	return cg
}

func (cg *CreatingGroup) getIDs() []groupsig.ID {
   return cg.gInfo.Mems
}

func (cg *CreatingGroup) acceptPiece(from groupsig.ID, sign groupsig.Signature) int8 {
	add, gen := cg.gSignGenerator.AddWitness(from, sign)
	if add {
		if gen {
			return PIECE_THRESHOLD
		} else {
			return PIECE_NORMAL
		}
	} else {
		if gen {
			return PIECE_DENY_RECOVERED
		} else {
			return PIECE_DENY_DUP
		}
	}
}

func (cgs *CreatingGroups) addCreatingGroup(group *CreatingGroup) {
	cgs.groups.Store(group.gInfo.GroupHash().Hex(), group)
}

func (cgs *CreatingGroups) acceptPiece(ghash common.Hash, from groupsig.ID, sign groupsig.Signature) int8 {
	if cg := cgs.getCreatingGroup(ghash); cg == nil {
		return PIECE_GROUP_NOTFOUND
	} else {
		return cg.acceptPiece(from, sign)
	}
}

func (cgs *CreatingGroups) getCreatingGroup(hash common.Hash) *CreatingGroup {
	if cg, ok := cgs.groups.Load(hash.Hex()); !ok {
		return nil
	} else {
		return cg.(*CreatingGroup)
	}
}

func (cgs *CreatingGroups) removeGroup(hash common.Hash) {
	cgs.groups.Delete(hash.Hex())
}

func (cgs *CreatingGroups) forEach(f func(cg *CreatingGroup) bool) {
	cgs.groups.Range(func(key, value interface{}) bool {
		return f(value.(*CreatingGroup))
	})
}

func (cgs *CreatingGroups) addSentHash(hash common.Hash)  {
    cgs.lock.Lock()
    defer cgs.lock.Unlock()
    cgs.sentInitGroupHashs[cgs.currIndex] = hash
    cgs.currIndex = (cgs.currIndex+1)%len(cgs.sentInitGroupHashs)
}

func (cgs *CreatingGroups) hasSentHash(hash common.Hash) bool {
	cgs.lock.RLock()
	defer cgs.lock.RUnlock()
	for _, h := range cgs.sentInitGroupHashs {
		if h == hash {
			return true
		}
	}
	return false
}