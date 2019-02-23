package logical

import (
	"consensus/groupsig"
	"core"
	"fmt"
	"consensus/model"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/

type GroupManager struct {
	groupChain     *core.GroupChain
	mainChain      core.BlockChain
	processor      *Processor
	//creatingGroups *CreatingGroups
	creatingGroupCtx *CreatingGroupContext
	checker          *GroupCreateChecker
}

func NewGroupManager(processor *Processor) *GroupManager {
	gm := &GroupManager{
		processor:      processor,
		mainChain:      processor.MainChain,
		groupChain:     processor.GroupChain,
		//creatingGroups: &CreatingGroups{},
		checker:        newGroupCreateChecker(processor),
	}
	return gm
}

func (gm *GroupManager) setCreatingGroupContext(baseCtx *createGroupBaseContext, kings []groupsig.ID, isKing bool)  {
	ctx := newCreateGroupContext(baseCtx, kings, isKing, gm.mainChain.Height())
	gm.creatingGroupCtx = ctx
}

func (gm *GroupManager) getContext() *CreatingGroupContext {
    return gm.creatingGroupCtx
}

func (gm *GroupManager) removeContext() {
	gm.creatingGroupCtx = nil
}

func (gm *GroupManager) CreateNextGroupRoutine() {
	top := gm.mainChain.QueryTopBlock()
	topHeight := top.Height

	gap := model.Param.GroupCreateGap
	if topHeight > gap {
		gm.checkReqCreateGroupSign(topHeight)

		pre := gm.mainChain.QueryBlockHeaderByHash(top.PreHash)
		if pre != nil {
			for h := top.Height; h > pre.Height && h > gap; h-- {
				baseHeight := h-gap
				if checkCreate(baseHeight) {
					gm.checkCreateGroupRoutine(baseHeight)
					break
				}
			}
		}
	}

}


func (gm *GroupManager) OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage) (bool, error) {
	blog := newBizLog("OMCGR")
	blog.log("gHash=%v, sender=%v", msg.GInfo.GI.GetHash().ShortS(), msg.SI.SignMember.ShortS())

	ctx := gm.getContext()
	if ctx.getStatus() == sendInit {
		return false, fmt.Errorf("has send inited")
	}
	top := gm.mainChain.Height()
	if ctx.readyTimeout(top) {
		return false, fmt.Errorf("ready timeout")
	}
	if !ctx.generateGroupInitInfo(top) {
		return false, fmt.Errorf("generate group init info fail")
	}
	if ctx.gInfo.GroupHash() != msg.GInfo.GroupHash() {
		blog.log("expect gh %+v, real gh %+v", ctx.gInfo.GI.GHeader, msg.GInfo.GI.GHeader)
		return false, fmt.Errorf("grouphash diff")
	}

	//if ok, err := gm.isGroupHeaderLegal(msg.GInfo.GI.GHeader); !ok {
	//	blog.log(err.Error())
	//	return false, err
	//}
	return true, nil

}

func (gm *GroupManager) OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage) (bool, error) {
	blog := newBizLog("OMCGS")
	blog.log("gHash=%v, sender=%v", msg.GHash.ShortS(), msg.SI.SignMember.ShortS())
	ctx := gm.getContext()
	if ctx == nil {
		return false, fmt.Errorf("context is nil")
	}

	height := gm.processor.MainChain.QueryTopBlock().Height
	if ctx.readyTimeout(height) {
		return false, fmt.Errorf("ready timeout")
	}
	if ctx.gInfo.GroupHash() != msg.GHash {
		return false, fmt.Errorf("gHash diff")
	}

	accept, recover := ctx.acceptPiece(msg.SI.GetID(), msg.SI.DataSign)
	blog.log("accept result %v %v", accept, recover)
	newHashTraceLog("OMCGS", msg.GHash, msg.SI.GetID()).log("OnMessageCreateGroupSign ret %v, %v", recover, ctx.gSignGenerator.Brief())
	if recover {
		ctx.gInfo.GI.Signature = ctx.gSignGenerator.GetGroupSign()
		return true, nil
	}
	return false, fmt.Errorf("waiting piece")
}

func (gm *GroupManager) AddGroupOnChain(sgi *StaticGroupInfo) {
	group := ConvertStaticGroup2CoreGroup(sgi)

	stdLogger.Infof("AddGroupOnChain height:%d,id:%s\n", group.GroupHeight, sgi.GroupID.ShortS())

	var err error
	defer func() {
		var s string
		if err != nil {
			s = err.Error()
		}
		newHashTraceLog("AddGroupOnChain", sgi.GInfo.GroupHash(), groupsig.ID{}).log("gid=%v, workHeight=%v, result %v", sgi.GroupID.ShortS(), group.Header.WorkHeight, s)
	}()

	if gm.groupChain.GetGroupById(group.Id) != nil {
		stdLogger.Debugf("group already onchain, accept, id=%v\n", sgi.GroupID.ShortS())
		gm.processor.acceptGroup(sgi)
		err = fmt.Errorf("group already onchain")
	} else {
		top := gm.processor.MainChain.Height()
		if !sgi.GetReadyTimeout(top) {
			err1 := gm.groupChain.AddGroup(group)
			if err1 != nil {
				stdLogger.Infof("ERROR:add group fail! hash=%v, gid=%v, err=%v\n", group.Header.Hash.ShortS(), sgi.GroupID.ShortS(), err1.Error())
				err = err1
				return
			}
			err = fmt.Errorf("success")
			gm.checker.addHeightCreated(group.Header.CreateHeight)
			stdLogger.Infof("AddGroupOnChain success, ID=%v, height=%v\n", sgi.GroupID.ShortS(), gm.groupChain.Count())
		} else {
			err = fmt.Errorf("ready timeout, currentHeight %v", top)
			stdLogger.Infof("AddGroupOnChain group ready timeout, gid %v, timeout height %v, top %v\n", sgi.GroupID.ShortS(), sgi.GInfo.GI.GHeader.ReadyHeight, top)
		}
	}


}

func (gm *GroupManager) onGroupAddSuccess(g *StaticGroupInfo)  {
	ctx := gm.getContext()
	if ctx != nil && ctx.gInfo != nil && ctx.gInfo.GroupHash() == g.GInfo.GroupHash() {
		top := gm.mainChain.Height()
		groupLogger.Infof("onGroupAddSuccess info=%v, gHash=%v, gid=%v, costHeight=%v", ctx.logString(), g.GInfo.GroupHash().ShortS(), g.GroupID.ShortS(), top-ctx.createTopHeight)
		gm.removeContext()
	}
}
