package param

import (
	"time"
	"common"
	"sync"
	"fmt"
	"governance"
)

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/

const DEFAULT_VERSION = 0


//参数校验函数
type ValidateFunc func(input interface{}) error


type ParamMeta struct {
	Value       interface{}
	version     uint32
	gmtModified time.Time
	voteAddr    common.Address
	blockHash   common.Hash256
	ValidBlock  uint64
}

type ParamDef struct {
	mu        sync.Mutex
	Current   *ParamMeta
	Futures   []*ParamMeta
	Histories []*ParamMeta
	Validate	ValidateFunc
}

func newMeta(v interface{}) *ParamMeta {
	return &ParamMeta{
		Value:      v,
		version:    DEFAULT_VERSION,
		ValidBlock: 0,
	}
}

func newParamDef(init interface{}, validator ValidateFunc) *ParamDef {
	return &ParamDef{
		Current: newMeta(init),
		Validate: validator,
	}
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
	if p.Histories == nil {
		p.Histories = []*ParamMeta{}
	}
	p.Histories = append(p.Histories, futures...)
}

func (p *ParamDef) tryApplyFutureMeta()  {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := len(p.Futures) - 1; i >= 0; i-- {
		f := p.Futures[i]
		if f.ValidBlock <= governance.CurrentBlock() {
			p.Histories = append(p.Histories, p.Current)
			p.Current = f
			p.Current.version ++
			p.copyFuture2History(p.Futures[:i])
			p.removePreFuture(i)
			break
		}
	}

}

func out(a []*ParamMeta) {
	for idx, meta := range a {
		fmt.Println(idx, meta.ValidBlock, meta.Value)
	}
}

func (p *ParamDef) printFuture()  {
    out(p.Futures)
}

func (p *ParamDef) printHistory()  {
	out(p.Histories)
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



