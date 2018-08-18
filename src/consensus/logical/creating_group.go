package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"strconv"
	"sync"
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
	gis            *model.ConsensusGroupInitSummary
	createGroup    *StaticGroupInfo
	ids            []groupsig.ID
	gSignGenerator *model.GroupSignGenerator
}

type CreatingGroups struct {
	groups sync.Map //string -> *CreatingGroup
	//groups map[string]*CreatingGroup
	//lock sync.RWMutex
}

func newCreateGroup(gis *model.ConsensusGroupInitSummary, ids []groupsig.ID, creator *StaticGroupInfo) *CreatingGroup {
	cg := &CreatingGroup{
		gis:            gis,
		ids:            ids,
		createGroup:    creator,
		gSignGenerator: model.NewGroupSignGenerator(model.Param.GetGroupK(creator.MemberCount())),
	}
	return cg
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
	cgs.groups.Store(group.gis.DummyID.GetHexString(), group)
}

func (cgs *CreatingGroups) acceptPiece(dummyId groupsig.ID, from groupsig.ID, sign groupsig.Signature) int8 {
	if cg := cgs.getCreatingGroup(dummyId); cg == nil {
		return PIECE_GROUP_NOTFOUND
	} else {
		return cg.acceptPiece(from, sign)
	}
}

func (cgs *CreatingGroups) getCreatingGroup(dummyId groupsig.ID) *CreatingGroup {
	if cg, ok := cgs.groups.Load(dummyId.GetHexString()); !ok {
		return nil
	} else {
		return cg.(*CreatingGroup)
	}
}

func (cgs *CreatingGroups) removeGroup(dummyId groupsig.ID) {
	cgs.groups.Delete(dummyId.GetHexString())
}

func (cgs *CreatingGroups) forEach(f func(cg *CreatingGroup) bool) {
	cgs.groups.Range(func(key, value interface{}) bool {
		return f(value.(*CreatingGroup))
	})
}
