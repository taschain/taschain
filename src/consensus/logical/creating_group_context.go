package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"sync"
	"time"
	"middleware/types"
	"consensus/base"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/6/25 下午12:14
**  Description:
 */


const (
	waitingPong = 1
	waitingSign = 2
	sendInit = 3
)

type createGroupBaseContext struct {
	parentInfo     *StaticGroupInfo
	baseBH         *types.BlockHeader
	baseGroup      *types.Group
	candidates     []groupsig.ID
}

type CreatingGroupContext struct {
	createGroupBaseContext

	gSignGenerator *model.GroupSignGenerator

	kings 			[]groupsig.ID
	pingID 		string
	createTime 	time.Time
	createTopHeight uint64

	gInfo 			*model.ConsensusGroupInitInfo
	pongMap 	map[string]byte
	memMask 	[]byte
	status 		int8
	bKing		bool
	lock 			sync.RWMutex
}

func newCreateGroupBaseContext(sgi *StaticGroupInfo, baseBH *types.BlockHeader, baseG *types.Group, cands []groupsig.ID) *createGroupBaseContext {
	return &createGroupBaseContext{
		parentInfo: sgi,
		baseBH: baseBH,
		baseGroup: baseG,
		candidates: cands,
	}
}

func newCreateGroupContext(baseCtx *createGroupBaseContext, kings []groupsig.ID, isKing bool, top uint64) *CreatingGroupContext {
	pingIdBytes := baseCtx.baseBH.Hash.Bytes()
	pingIdBytes = append(pingIdBytes, baseCtx.baseGroup.Id...)
	cg := &CreatingGroupContext{
		//gInfo: gInfo,
		//createGroup:    creator,
		createGroupBaseContext: *baseCtx,
		kings: 			kings,
		status:         waitingPong,
		createTime: 	time.Now(),
		bKing:			isKing,
		createTopHeight: top,
		pingID: 		base.Data2CommonHash(pingIdBytes).Hex(),
		pongMap: 		make(map[string]byte, 0),
		gSignGenerator: model.NewGroupSignGenerator(model.Param.GetGroupK(baseCtx.parentInfo.GetMemberCount())),
	}

	return cg
}

func (ctx  *createGroupBaseContext) hasCandidate(uid groupsig.ID) bool {
	for _, id := range ctx.candidates {
		if id.IsEqual(uid) {
			return true
		}
	}
	return false
}

func (ctx *createGroupBaseContext) readyHeight() uint64 {
    return ctx.baseBH.Height + model.Param.GroupReadyGap
}

func (ctx *createGroupBaseContext) readyTimeout(h uint64) bool {
    return h >= ctx.readyHeight()
}

func (ctx *createGroupBaseContext) recoverMemberSet(mask []byte) (ids []groupsig.ID) {
	ids = make([]groupsig.ID, 0)
	for i, id := range ctx.candidates {
		b := mask[i/8]
		if (b & (1 << byte(i%8))) != 0 {
			ids = append(ids, id)
		}
	}
	return
}


func (ctx *createGroupBaseContext) createGroupHeader(memIds []groupsig.ID) *types.GroupHeader {
	pid := ctx.parentInfo.GroupID
	theBH := ctx.baseBH
	gn := fmt.Sprintf("%s-%v", pid.GetHexString(), theBH.Height)
	extends := fmt.Sprintf("baseBlock:%v|%v|%v", theBH.Hash.Hex(), theBH.CurTime, theBH.Height)

	gh := &types.GroupHeader{
		Parent: ctx.parentInfo.GroupID.Serialize(),
		PreGroup: ctx.baseGroup.Id,
		Name: gn,
		Authority: 777,
		BeginTime: theBH.CurTime,
		CreateHeight: theBH.Height,
		ReadyHeight: ctx.readyHeight(),
		WorkHeight: theBH.Height + model.Param.GroupWorkGap,
		MemberRoot: model.GenMemberRootByIds(memIds),
		Extends: extends,
	}
	gh.DismissHeight = gh.WorkHeight + model.Param.GroupworkDuration

	gh.Hash = gh.GenHash()
	return gh
}


func (ctx *createGroupBaseContext) createGroupInitInfo(mask []byte) *model.ConsensusGroupInitInfo {
	memIds := ctx.recoverMemberSet(mask)
	gh := ctx.createGroupHeader(memIds)
	return &model.ConsensusGroupInitInfo{
		GI:   model.ConsensusGroupInitSummary{GHeader: gh},
		Mems: memIds,
	}
}

func (ctx *CreatingGroupContext) pongDeadline(h uint64) bool {
	return h >= ctx.baseBH.Height + model.Param.GroupWaitPongGap
}

func (ctx *CreatingGroupContext) isKing() bool {
	return ctx.bKing
}

func (ctx *CreatingGroupContext) addPong(h uint64, uid groupsig.ID) (add bool, size int) {
	if ctx.pongDeadline(h) {
		return false, ctx.pongSize()
	}
	ctx.lock.Lock()
	defer ctx.lock.Unlock()


	if ctx.hasCandidate(uid) {
		ctx.pongMap[uid.GetHexString()] = 1
		add = true
	}
	size = len(ctx.pongMap)
	return
}

func (ctx *CreatingGroupContext) pongSize() int {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return len(ctx.pongMap)
}

func (ctx *CreatingGroupContext) getStatus() int8 {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return ctx.status
}

func (ctx *CreatingGroupContext) setStatus(st int8) {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	ctx.status = st
}

func (ctx *CreatingGroupContext) generateMemberMask() (mask []byte) {
	mask = make([]byte, (len(ctx.candidates)+7)/8)

	for i, id := range ctx.candidates {
		b := mask[i/8]
		if _, ok := ctx.pongMap[id.GetHexString()]; ok {
			b |= 1 << byte(i%8)
			mask[i/8] = b
		}
	}
	return
}

func (ctx *CreatingGroupContext) generateGroupInitInfo(h uint64) bool {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	if ctx.gInfo != nil {
		return true
	}
	if len(ctx.pongMap) == len(ctx.candidates) || ctx.pongDeadline(h) {
		mask := ctx.generateMemberMask()
		gInfo := ctx.createGroupInitInfo(mask)
		ctx.gInfo = gInfo
		ctx.memMask = mask
		return true
	}

	return false
}

func (ctx *CreatingGroupContext) acceptPiece(from groupsig.ID, sign groupsig.Signature) (accept, recover bool) {
	accept, recover = ctx.gSignGenerator.AddWitness(from, sign)
	return
}

func (ctx *CreatingGroupContext) logString() string {
	return fmt.Sprintf("baseHeight=%v, topHeight=%v, candidates=%v, isKing=%v, parentGroup=%v, pongs=%v, elapsed=%v",
		ctx.baseBH.Height, ctx.createTopHeight, len(ctx.candidates), ctx.isKing(), ctx.parentInfo.GroupID.ShortS(), ctx.pongSize(), time.Since(ctx.createTime).String())

}