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
	PIECE_DENY_DIFF = 3
	PIECE_DENY_DUP = 4
)


type CreatingGroup struct {
	gis *ConsensusGroupInitSummary
	creator *StaticGroupInfo
	ids []groupsig.ID
	threshold int
	pieces map[string]groupsig.Signature
	lock sync.RWMutex
}

type CreatingGroups struct {
	groups sync.Map		//string -> *CreatingGroup
	//groups map[string]*CreatingGroup
	//lock sync.RWMutex
}

func newCreateGroup(gis *ConsensusGroupInitSummary, ids []groupsig.ID, creator *StaticGroupInfo) *CreatingGroup {
	return &CreatingGroup{
		gis: gis,
		ids: ids,
		pieces: make(map[string]groupsig.Signature),
		creator: creator,
		threshold: GetGroupK(len(creator.Members)),
	}
}

func (cg *CreatingGroup) getPieces() map[string]groupsig.Signature {
    cg.lock.RLock()
    defer cg.lock.RUnlock()
    return cg.pieces
}

func (cg *CreatingGroup) acceptPiece(from groupsig.ID, sign groupsig.Signature) int8 {
	cg.lock.Lock()
	defer cg.lock.Unlock()

	if s, ok := cg.pieces[from.GetHexString()]; ok {
		if !sign.IsEqual(s) {
			panic("sign diff!")
			return PIECE_DENY_DIFF
		} else {
			return PIECE_DENY_DUP
		}
	} else {
		cg.pieces[from.GetHexString()] = sign
		if cg.reachThresholdPiece() {
			return PIECE_THRESHOLD
		} else {
			return PIECE_NORMAL
		}
	}
}

func (cg *CreatingGroup) reachThresholdPiece() bool {
    return len(cg.pieces) >= cg.threshold
}

func (cgs *CreatingGroups) addCreatingGroup(group *CreatingGroup)  {
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

func (cgs *CreatingGroups) removeGroup(dummyId groupsig.ID)  {
	cgs.groups.Delete(dummyId.GetHexString())
}