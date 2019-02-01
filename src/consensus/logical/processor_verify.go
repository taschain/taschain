package logical

import (
	"consensus/model"
	"sync"
	"common"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019/2/1 上午10:58
**  Description: 
*/


type verifyMsgCache struct {
	castMsg *model.ConsensusCastMessage
	verifyMsgs []*model.ConsensusVerifyMessage
	expire time.Time
	lock sync.RWMutex
}

func newVerifyMsgCache() *verifyMsgCache {
	return &verifyMsgCache{
		verifyMsgs: make([]*model.ConsensusVerifyMessage, 0),
		expire: time.Now().Add(30*time.Second),
	}
}

func (c *verifyMsgCache) expired() bool {
    return time.Now().After(c.expire)
}

func (c *verifyMsgCache) addVerifyMsg(msg *model.ConsensusVerifyMessage)  {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.verifyMsgs = append(c.verifyMsgs, msg)
}

func (c *verifyMsgCache) merge(msg *verifyMsgCache)  {
    c.lock.Lock()
    defer c.lock.Unlock()
	if msg.castMsg != nil && c.castMsg == nil {
		c.castMsg = msg.castMsg
	}
	if msg.verifyMsgs != nil {
		for _, m := range msg.verifyMsgs {
			c.verifyMsgs = append(c.verifyMsgs, m)
		}
	}
}

func (c *verifyMsgCache) getVerifyMsgs() []*model.ConsensusVerifyMessage {
    msgs := make([]*model.ConsensusVerifyMessage, len(c.verifyMsgs))
    c.lock.RLock()
    defer c.lock.RUnlock()
    copy(msgs, c.verifyMsgs)
    return msgs
}

func (c *verifyMsgCache) removeVerifyMsgs()  {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.verifyMsgs = make([]*model.ConsensusVerifyMessage, 0)
}

func (p *Processor) addVerifyCache(hash common.Hash, cache *verifyMsgCache)  {
	if ok, _ := p.verifyMsgCaches.ContainsOrAdd(hash, cache); ok {
		c := p.getVerifyMsgCache(hash)
		if c == nil {
			return
		}
		c.merge(cache)
	}
}

func (p *Processor) addVerifyMsgToCache(msg *model.ConsensusVerifyMessage)  {
	cache := p.getVerifyMsgCache(msg.BlockHash)
	if cache == nil {
		cache := newVerifyMsgCache()
		cache.addVerifyMsg(msg)
		p.addVerifyCache(msg.BlockHash, cache)
	} else {
		cache.addVerifyMsg(msg)
	}
}

func (p *Processor) addCastMsgToCache(msg *model.ConsensusCastMessage)  {
    cache := p.getVerifyMsgCache(msg.BH.Hash)
	if cache == nil {
		cache := newVerifyMsgCache()
		cache.castMsg = msg
		p.addVerifyCache(msg.BH.Hash, cache)
	} else {
		cache.castMsg = msg
	}
}

func (p *Processor) getVerifyMsgCache(hash common.Hash) *verifyMsgCache {
	v, ok := p.verifyMsgCaches.Get(hash)
	if !ok {
		return nil
	}
	return v.(*verifyMsgCache)
}

func (p *Processor) removeVerifyMsgCache(hash common.Hash)  {
	p.verifyMsgCaches.Remove(hash)
}
