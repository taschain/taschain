package logical

import (
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
	time2 "github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"sync"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019/2/1 上午10:58
**  Description:
 */
type castedBlock struct {
	height  uint64
	preHash common.Hash
}
type verifyMsgCache struct {
	verifyMsgs []*model.ConsensusVerifyMessage
	expire     time.Time
	lock       sync.RWMutex
}

type proposedBlock struct {
	block            *types.Block
	responseCount    uint
	maxResponseCount uint
}

func newVerifyMsgCache() *verifyMsgCache {
	return &verifyMsgCache{
		verifyMsgs: make([]*model.ConsensusVerifyMessage, 0),
		expire:     time.Now().Add(30 * time.Second),
	}
}

func (c *verifyMsgCache) expired() bool {
	return time.Now().After(c.expire)
}

func (c *verifyMsgCache) addVerifyMsg(msg *model.ConsensusVerifyMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.verifyMsgs = append(c.verifyMsgs, msg)
}

func (c *verifyMsgCache) getVerifyMsgs() []*model.ConsensusVerifyMessage {
	msgs := make([]*model.ConsensusVerifyMessage, len(c.verifyMsgs))
	c.lock.RLock()
	defer c.lock.RUnlock()
	copy(msgs, c.verifyMsgs)
	return msgs
}

func (c *verifyMsgCache) removeVerifyMsgs() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.verifyMsgs = make([]*model.ConsensusVerifyMessage, 0)
}

type castBlockContexts struct {
	proposed        *lru.Cache //hash -> *Block
	heightVctxs     *lru.Cache //height -> *VerifyContext
	hashVctxs       *lru.Cache //hash -> *VerifyContext
	reservedVctx    *lru.Cache //uint64 -> *VerifyContext 存储已经有签出块的verifyContext，待广播
	verifyMsgCaches *lru.Cache //hash -> *verifyMsgCache 缓存验证消息
	recentCasted    *lru.Cache //height -> *castedBlock
	chain           core.BlockChain
}

func newCastBlockContexts(chain core.BlockChain) *castBlockContexts {
	return &castBlockContexts{
		proposed:        common.MustNewLRUCache(20),
		heightVctxs:     common.MustNewLRUCacheWithEvitCB(20, heightVctxEvitCallback),
		hashVctxs:       common.MustNewLRUCache(200),
		reservedVctx:    common.MustNewLRUCache(100),
		verifyMsgCaches: common.MustNewLRUCache(200),
		recentCasted:    common.MustNewLRUCache(200),
		chain:           chain,
	}
}

func heightVctxEvitCallback(k, v interface{}) {
	ctx := v.(*VerifyContext)
	stdLogger.Debugf("evitVctx: ctx.castHeight=%v, ctx.prevHash=%v, signedMaxQN=%v, signedNum=%v, verifyNum=%v, aggrNum=%v\n", ctx.castHeight, ctx.prevBH.Hash.ShortS(), ctx.signedMaxWeight, ctx.signedNum, ctx.verifyNum, ctx.aggrNum)
}

func (bctx *castBlockContexts) removeReservedVctx(height uint64) {
	bctx.reservedVctx.Remove(height)
}

func (bctx *castBlockContexts) addReservedVctx(vctx *VerifyContext) bool {
	_, load := bctx.reservedVctx.ContainsOrAdd(vctx.castHeight, vctx)
	return !load
}

func (bctx *castBlockContexts) forEachReservedVctx(f func(vctx *VerifyContext) bool) {
	for _, k := range bctx.reservedVctx.Keys() {
		v, ok := bctx.reservedVctx.Peek(k)
		if ok {
			if !f(v.(*VerifyContext)) {
				break
			}
		}
	}
}

func (bctx *castBlockContexts) addProposed(b *types.Block) {
	pb := proposedBlock{block: b}
	bctx.proposed.Add(b.Header.Hash, &pb)
}

func (bctx *castBlockContexts) getProposed(hash common.Hash) *proposedBlock {
	if v, ok := bctx.proposed.Peek(hash); ok {
		return v.(*proposedBlock)
	}
	return nil
}

func (bctx *castBlockContexts) removeProposed(hash common.Hash) {
	bctx.proposed.Remove(hash)
}

func (bctx *castBlockContexts) isHeightCasted(height uint64, pre common.Hash) (cb *castedBlock, casted bool) {
	v, ok := bctx.recentCasted.Peek(height)
	if ok {
		cb := v.(*castedBlock)
		return cb, cb.preHash == pre
	}
	return
}

func (bctx *castBlockContexts) addCastedHeight(height uint64, pre common.Hash) {
	if _, ok := bctx.isHeightCasted(height, pre); !ok {
		bctx.recentCasted.Add(height, &castedBlock{height: height, preHash: pre})
	}
}

func (bctx *castBlockContexts) getVctxByHeight(height uint64) *VerifyContext {
	if v, ok := bctx.heightVctxs.Peek(height); ok {
		return v.(*VerifyContext)
	}
	return nil
}

func (bctx *castBlockContexts) addVctx(vctx *VerifyContext) {
	bctx.heightVctxs.Add(vctx.castHeight, vctx)
}

func (bctx *castBlockContexts) attachVctx(bh *types.BlockHeader, vctx *VerifyContext) {
	bctx.hashVctxs.Add(bh.Hash, vctx)
}

func (bctx *castBlockContexts) getVctxByHash(hash common.Hash) *VerifyContext {
	if v, ok := bctx.hashVctxs.Peek(hash); ok {
		return v.(*VerifyContext)
	}
	return nil
}

func (bctx *castBlockContexts) replaceVerifyCtx(group *StaticGroupInfo, height uint64, expireTime time2.TimeStamp, preBH *types.BlockHeader) *VerifyContext {
	vctx := newVerifyContext(group, height, expireTime, preBH)
	bctx.addVctx(vctx)
	return vctx
}

func (bctx *castBlockContexts) getOrNewVctx(group *StaticGroupInfo, height uint64, expireTime time2.TimeStamp, preBH *types.BlockHeader) *VerifyContext {
	var vctx *VerifyContext
	blog := newBizLog("getOrNewVctx")

	//若该高度还没有verifyContext， 则创建一个
	if vctx = bctx.getVctxByHeight(height); vctx == nil {
		vctx = newVerifyContext(group, height, expireTime, preBH)
		bctx.addVctx(vctx)
		blog.log("add vctx expire %v", expireTime)
	} else {
		// hash不一致的情况下，
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, height=%v, existHash=%v, commingHash=%v", height, vctx.prevBH.Hash.ShortS(), preBH.Hash.ShortS())
			preOld := bctx.chain.QueryBlockHeaderByHash(vctx.prevBH.Hash)
			//原来的preBH可能被分叉调整干掉了，则此vctx已无效， 重新用新的preBH
			if preOld == nil {
				vctx = bctx.replaceVerifyCtx(group, height, expireTime, preBH)
				return vctx
			}
			preNew := bctx.chain.QueryBlockHeaderByHash(preBH.Hash)
			//新的preBH不存在了，也可能被分叉干掉了，此处直接返回nil
			if preNew == nil {
				return nil
			}
			//新旧preBH都非空， 取高度高的preBH？
			if preOld.Height < preNew.Height {
				vctx = bctx.replaceVerifyCtx(group, height, expireTime, preNew)
			}
		} else {
			if height == 1 && expireTime.After(vctx.expireTime) {
				vctx.expireTime = expireTime
			}
			blog.log("get exist vctx height %v, expire %v", height, vctx.expireTime)
		}
	}
	return vctx
}

func (bctx *castBlockContexts) getOrNewVerifyContext(group *StaticGroupInfo, bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	deltaHeightByTime := DeltaHeightByTime(bh, preBH)

	expireTime := GetCastExpireTime(preBH.CurTime, deltaHeightByTime, bh.Height)

	vctx := bctx.getOrNewVctx(group, bh.Height, expireTime, preBH)
	return vctx
}

func (bctx *castBlockContexts) cleanVerifyContext(height uint64) {
	for _, h := range bctx.heightVctxs.Keys() {
		v, ok := bctx.heightVctxs.Peek(h)
		if !ok {
			continue
		}
		ctx := v.(*VerifyContext)
		bRemove := ctx.shouldRemove(height)
		if bRemove {
			for _, slot := range ctx.GetSlots() {
				bctx.hashVctxs.Remove(slot.BH.Hash)
			}
			ctx.Clear()
			bctx.removeReservedVctx(ctx.castHeight)
			bctx.heightVctxs.Remove(h)
			stdLogger.Debugf("cleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v, signedMaxQN=%v, signedNum=%v, verifyNum=%v, aggrNum=%v\n", ctx.castHeight, ctx.prevBH.Hash.ShortS(), ctx.signedMaxWeight, ctx.signedNum, ctx.verifyNum, ctx.aggrNum)
		}
	}
}

func (bctx *castBlockContexts) addVerifyMsg(msg *model.ConsensusVerifyMessage) {
	if v, ok := bctx.verifyMsgCaches.Get(msg.BlockHash); ok {
		c := v.(*verifyMsgCache)
		c.addVerifyMsg(msg)
	} else {
		c := newVerifyMsgCache()
		c.addVerifyMsg(msg)
		bctx.verifyMsgCaches.ContainsOrAdd(msg.BlockHash, c)
	}
}

func (bctx *castBlockContexts) getVerifyMsgCache(hash common.Hash) *verifyMsgCache {
	v, ok := bctx.verifyMsgCaches.Peek(hash)
	if !ok {
		return nil
	}
	return v.(*verifyMsgCache)
}

func (bctx *castBlockContexts) removeVerifyMsgCache(hash common.Hash) {
	bctx.verifyMsgCaches.Remove(hash)
}
