package logical

import (
	"consensus/groupsig"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/6/25 下午12:14
**  Description: 
*/

const (
	PIECE_GROUP_NOTFOUND int8 = 0
	PIECE_NORMAL = 1
	PIECE_THRESHOLD = 2
)

type CreatingGroup struct {
	gis *ConsensusGroupInitSummary
	ids []groupsig.ID
	pieces map[string]groupsig.Signature
}

type CreateGroupContext struct {
	groups map[string]*CreatingGroup
	lock sync.RWMutex
}

func newCreateGroup(gis *ConsensusGroupInitSummary, ids []groupsig.ID) *CreatingGroup {
	return &CreatingGroup{
		gis: gis,
		ids: ids,
		pieces: make(map[string]groupsig.Signature),
	}
}

func (cg *CreatingGroup) acceptPiece(from groupsig.ID, sign groupsig.Signature) bool {
	if s, ok := cg.pieces[from.GetHexString()]; ok {
		if !sign.IsEqual(s) {
			panic("sign diff!")
		}
	} else {
		cg.pieces[from.GetHexString()] = sign
	}
	return true
}

func (cg *CreatingGroup) reachThresholdPiece() bool {
    return len(cg.pieces) >= cg.threshold()
}

func (cg *CreatingGroup) threshold() int {
	return GetGroupK(int(cg.gis.Members))
}

func (ctx *CreateGroupContext) addCreatingGroup(group *CreatingGroup)  {
    ctx.lock.Lock()
    defer ctx.lock.Unlock()
    ctx.groups[group.gis.DummyID.GetHexString()] = group
}

func (ctx *CreateGroupContext) acceptPiece(dummyId groupsig.ID, from groupsig.ID, sign groupsig.Signature) int8 {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()

	if cg, ok := ctx.groups[dummyId.GetHexString()]; !ok {
		return PIECE_GROUP_NOTFOUND
	} else {
		cg.acceptPiece(from, sign)
		if cg.reachThresholdPiece() {
			return PIECE_THRESHOLD
		} else {
			return PIECE_NORMAL
		}
	}
}

func (ctx *CreateGroupContext) getCreatingGroup(dummyId groupsig.ID) *CreatingGroup {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()

	if cg, ok := ctx.groups[dummyId.GetHexString()]; !ok {
		return nil
	} else {
		return cg
	}
}

func (ctx *CreateGroupContext) removeGroup(dummyId groupsig.ID)  {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	delete(ctx.groups, dummyId.GetHexString())
}