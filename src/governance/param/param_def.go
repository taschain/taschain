package param

import (
	"time"
	"common"
	"sync"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/

const DEFAULT_VERSION = 0

const (
	DEFAULT_GASPRICE_MIN            = 10000 //gasprice min
	DEFAULT_BLOCK_FIX_AWARD         = 5     //区块固定奖励
	DEFAULT_VOTER_CNT_MIN           = 100   //最小参与投票人
	DEFAULT_VOTER_DEPOSIT_MIN       = 1     //每个投票人最低缴纳的保证金
	DEFAULT_VOTER_TOTAL_DEPOSIT_MIN = 100   //所有投票人最低保证金总和
)

type ParamMeta struct {
	Value       interface{}
	version     uint32
	gmtModified time.Time
	voteAddr    common.Address
	blockHash   common.Hash256
	ValidBlock  uint64
}

type ParamDef struct {
	mu			*sync.Mutex
	Current 	*ParamMeta
	Futures 	[]*ParamMeta
	Historys	[]*ParamMeta
	//RangeCheck
}

func newMeta(v interface{}) *ParamMeta {
	return &ParamMeta{
		Value:      v,
		version:    DEFAULT_VERSION,
		ValidBlock: 0,
	}
}

func newParamDef(current *ParamMeta) *ParamDef {
	return &ParamDef{
		Current: current,
		mu: new(sync.Mutex),
	}
}

func currentBlock() uint64 {
	//TODO: 获取当前区块高度
	return 2
}

func (p *ParamDef ) CurrentValue() interface{} {
	p.tryApplyFutureMeta()
    return p.Current.Value
}

func (p *ParamDef) findPos4Future(meta* ParamMeta) uint {
	i := 0
	for ; i < len(p.Futures) && p.Futures[i].ValidBlock < meta.ValidBlock; i++ {}
	return uint(i)
}

func (p *ParamDef) removePreFuture(idx int)  {
	if len(p.Futures) == idx {
		p.Futures = []*ParamMeta{}
	} else {
		p.Futures = p.Futures[idx+1:]
	}
}

func (p *ParamDef) copyFuture2History(futures []*ParamMeta)  {
	if p.Historys == nil {
		p.Historys = []*ParamMeta{}
	}
	p.Historys = append(p.Historys, futures...)
}

func (p *ParamDef) tryApplyFutureMeta()  {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := len(p.Futures) - 1; i >= 0; i-- {
		f := p.Futures[i]
		if f.ValidBlock <= currentBlock() {
			p.Historys = append(p.Historys, p.Current)
			p.Current = f
			p.Current.version ++
			p.copyFuture2History(p.Futures[:i])
			p.removePreFuture(i)
			break
		}
	}

}

func print(a []*ParamMeta) {
	for idx, meta := range a {
		fmt.Println(idx, meta.ValidBlock, meta.Value)
	}
}

func (p *ParamDef) printFuture()  {
    print(p.Futures)
}

func (p *ParamDef) printHistory()  {
	print(p.Historys)
}

func (p *ParamDef) AddFuture(meta *ParamMeta) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Futures == nil {
		p.Futures = []*ParamMeta{}
	}

	pos := p.findPos4Future(meta)
	p.Futures = append(p.Futures, meta)

	copy(p.Futures[pos+1:], p.Futures[pos: len(p.Futures)-1])
	p.Futures[pos] = meta

}



