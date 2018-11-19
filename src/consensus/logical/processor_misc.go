package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"log"
	"strings"
	"middleware/types"
	"consensus/base"
	"github.com/vmihailenco/msgpack"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/6/12 下午6:12
**  Description:
 */

func (p *Processor) genBelongGroupStoreFile() string {
	storeFile := consensusConfManager.GetString("joined_group_store", "")
	if strings.TrimSpace(storeFile) == "" {
		storeFile = "joined_group.config." + common.GlobalConf.GetString("instance", "index", "")
	}
	return storeFile
}

//后续如有全局定时器，从这个函数启动
func (p *Processor) Start() bool {
	p.Ticker.RegisterRoutine(p.getCastCheckRoutineName(), p.checkSelfCastRoutine, 1)
	p.Ticker.RegisterRoutine(p.getReleaseRoutineName(), p.releaseRoutine, 2)
	p.Ticker.StartTickerRoutine(p.getReleaseRoutineName(), false)
	p.triggerCastCheck()
	p.prepareMiner()
	p.ready = true
	return true
}

//预留接口
func (p *Processor) Stop() {
	return
}

func (p *Processor) prepareMiner() {

	topHeight := p.MainChain.QueryTopBlock().Height

	storeFile := p.genBelongGroupStoreFile()

	belongs := NewBelongGroups(storeFile)
	if !belongs.load() {

	}

	log.Printf("prepareMiner get groups from groupchain, belongGroup len=%v\n", belongs.groupSize())
	iterator := p.GroupChain.NewIterator()
	groups := make([]*StaticGroupInfo, 0)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre(){
		log.Printf("get group from core, id=%v", coreGroup.Id)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		needBreak := false
		sgi := NewSGIFromCoreGroup(coreGroup)
		if sgi.Dismissed(topHeight) {
			needBreak = true
			genesis := p.GroupChain.GetGroupByHeight(0)
			if coreGroup == nil {
				panic("get genesis group nil")
			}
			sgi = NewSGIFromCoreGroup(genesis)
		}
		groups = append(groups, sgi)
		log.Printf("load group=%v, beginHeight=%v, topHeight=%v\n", sgi.GroupID.ShortS(), sgi.BeginHeight, topHeight)
		if sgi.MemExist(p.GetMinerID()) {
			jg := belongs.getJoinedGroup(sgi.GroupID)
			if jg == nil {
				log.Printf("prepareMiner get join group fail, gid=%v\n", sgi.GroupID.ShortS())
			} else {
				p.joinGroup(jg, true)
			}
		}
		if needBreak {
			break
		}
	}
	for i := len(groups)-1; i >=0; i-- {
		p.acceptGroup(groups[i])
	}
	log.Printf("prepare finished")
}

func (p *Processor) Ready() bool {
	return p.ready
}

func (p *Processor) GetAvailableGroupsAt(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetAvailableGroups(height)
}

func (p *Processor) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	return p.globalGroups.GetCastQualifiedGroups(height)
}

func (p *Processor) Finalize() {
	if p.belongGroups != nil {
		p.belongGroups.commit()
	}
}

func (p *Processor) releaseRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height
	if topHeight <= model.Param.CreateGroupInterval {
		return true
	}
	//在当前高度解散的组不应立即从缓存删除，延缓一个建组周期删除。保证改组解散前夕建的块有效
	ids := p.globalGroups.DismissGroups(topHeight - model.Param.CreateGroupInterval)
	if len(ids) == 0 {
		return true
	}
	log.Printf("releaseRoutine: clean group %v\n", len(ids))
	p.globalGroups.RemoveGroups(ids)
	p.blockContexts.removeContexts(ids)
	p.belongGroups.leaveGroups(ids)
	for _, gid := range ids {
		log.Println("releaseRoutine DissolveGroupNet staticGroup gid ", gid.ShortS())
		p.NetServer.ReleaseGroupNet(gid)
	}

	//释放超时未建成组的组网络和相应的dummy组
	p.joiningGroups.forEach(func(gc *GroupContext) bool {
		if gc.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from joutils.GetngGroups gid ", gc.gis.DummyID.ShortS())
			p.NetServer.ReleaseGroupNet(gc.gis.DummyID)
			p.joiningGroups.RemoveGroup(gc.gis.DummyID)
		}
		return true
	})
	p.groupManager.creatingGroups.forEach(func(cg *CreatingGroup) bool {
		if cg.gis.ReadyTimeout(topHeight) {
			log.Println("releaseRoutine DissolveGroupNet dummyGroup from creatingGroups gid ", cg.gis.DummyID.ShortS())
			p.NetServer.ReleaseGroupNet(cg.gis.DummyID)
			p.groupManager.creatingGroups.removeGroup(cg.gis.DummyID)
		}
		return true
	})
	return true
}

func (p *Processor) getVrfWorker() *vrfWorker {
	if v := p.vrf.Load(); v != nil {
		return v.(*vrfWorker)
	}
	return nil
}

func (p *Processor) setVrfWorker(vrf *vrfWorker) {
	p.vrf.Store(vrf)
}

func (p *Processor) GetSelfMinerDO() *model.SelfMinerDO {
	md := p.minerReader.getProposeMiner(p.GetMinerID())
	if md != nil {
		p.mi.MinerDO = *md
	}
	return p.mi
}

func (p *Processor) canProposalAt(h uint64) bool {
	miner := p.minerReader.getProposeMiner(p.GetMinerID())
	if miner == nil {
		return false
	}
   	return miner.CanCastAt(h)
}

func (p *Processor) GetJoinedWorkGroupNums() (work, avail int) {
	h := p.MainChain.QueryTopBlock().Height
	groups := p.globalGroups.GetAvailableGroups(h)
	for _, g := range groups {
		if !g.MemExist(p.GetMinerID()) {
			continue
		}
		if g.CastQualified(h) {
			work++
		}
		avail++
	}
	return
}

func (p *Processor) CalcBlockHeaderQN(bh *types.BlockHeader) uint64 {
	pi := base.VRFProve(bh.ProveValue.Bytes())
	castor := groupsig.DeserializeId(bh.Castor)
	miner := p.minerReader.getProposeMiner(castor)
	if miner == nil {
		log.Printf("CalcBHQN getMiner nil id=%v, bh=%v", castor.ShortS(), bh.Hash.ShortS())
		return 0
	}
	totalStake := p.minerReader.getTotalStake(bh.Height)
	_, qn := vrfSatisfy(pi, miner.Stake, totalStake)
	return qn
}

func marshalBlock(b types.Block) ([]byte, error) {
	if b.Transactions != nil && len(b.Transactions) == 0 {
		b.Transactions = nil
	}
	if b.Header.Transactions != nil && len(b.Header.Transactions) == 0 {
		b.Header.Transactions = nil
	}
	return msgpack.Marshal(&b)
}

func (p *Processor) GenVerifyHash(b *types.Block, id groupsig.ID) common.Hash {
	buf, err := marshalBlock(*b)
	if err != nil {
		panic(fmt.Sprintf("marshal block error, hash=%v, err=%v", b.Header.Hash.ShortS(), err))
	}
	//header := &b.Header
	//log.Printf("GenVerifyHash aaa bufHash=%v, buf %v", base.Data2CommonHash(buf).ShortS(), buf)
	//log.Printf("GenVerifyHash aaa headerHash=%v, genHash=%v", b.Header.Hash.ShortS(), b.Header.GenHash().ShortS())

	//headBuf, _ := msgpack.Marshal(header)
	//log.Printf("GenVerifyHash aaa headerBufHash=%v, headerBuf=%v", base.Data2CommonHash(headBuf).ShortS(), headBuf)

	//log.Printf("GenVerifyHash height:%v,id:%v,%v, bbbbbuf %v", b.Header.Height,id.ShortS(), b.Transactions == nil, buf)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf ids %v", b.Header.Height,id.ShortS(),id.Serialize())
	buf = append(buf, id.Serialize()...)
	//log.Printf("GenVerifyHash height:%v,id:%v,bbbbbuf after %v", b.Header.Height,id.ShortS(),buf)
	h := base.Data2CommonHash(buf)
	log.Printf("GenVerifyHash height:%v,id:%v,bh:%v,vh:%v", b.Header.Height,id.ShortS(),b.Header.Hash.ShortS(), h.ShortS())
	return h
}
