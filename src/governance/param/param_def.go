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


type ParamMeta struct {
	Value       interface{}
	version     uint32
	gmtModified time.Time
	VoteAddr    common.Address
	Block   	uint64
	ValidBlock  uint64
}

type ParamDef struct {
	mu        sync.Mutex
	Current   *ParamMeta
	Futures   []*ParamMeta
	Histories []*ParamMeta
	Validate	ValidateFunc
	update	chan int
}

type ParamDefs struct {
	defs []*ParamDef
}

func NewMeta(v interface{}) *ParamMeta {
	return &ParamMeta{
		Value:      v,
		version:    DEFAULT_VERSION,
		ValidBlock: 0,
	}
}

func newParamDef(init interface{}, validator ValidateFunc) *ParamDef {
	return &ParamDef{
		Current:  NewMeta(init),
		Validate: validator,
		update:   make(chan int),
	}
}
func newParamDefs() *ParamDefs {
	return &ParamDefs{
		defs: make([]*ParamDef,0, PARAM_CNT),

	}
}


func (p *ParamDef ) CurrentValue() interface{} {
	p.tryApplyFutureMeta()
    return p.Current.Value
}

func (p *ParamDef) findPos4Future(meta* ParamMeta) int {
	i := 0
	for ; i < len(p.Futures) && p.Futures[i].ValidBlock < meta.ValidBlock; i++ {}
	return i
}

func (p *ParamDef) removePreFuture(idx int)  {
	if len(p.Futures)-1 == idx {
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

func (p *ParamDef) notifyUpdate()  {
	p.update<- 1
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
			p.notifyUpdate()
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
	l := len(p.Futures)
	p.Futures = append(p.Futures, meta)

	if pos < l {
		copy(p.Futures[pos+1:], p.Futures[pos: l])
		p.Futures[pos] = meta
	}

	p.notifyUpdate()
}


func buildUint64ParamDef(def *DefaultValueDef) *ParamDef {
	return newParamDef(
		def.init,
		getValidateFunc(def))
}


func (p *ParamDefs) AddParam(def *ParamDef)  {
	p.defs = append(p.defs, def)
}

func (p *ParamDefs) GetParamByIndex(index int) *ParamDef {
	if index > p.Size() {
		return nil
	}
	return p.defs[index]
}

func (p *ParamDefs) Size() int {
	return len(p.defs)
}

func (p *ParamDefs) UpdateParam(index int, def *ParamDef) error {
	if p.Size() <= index {
		return fmt.Errorf("error index %v", index)
	}
	if err := def.Validate(def.Current.Value); err != nil {
		return err
	}
	p.defs[index] = def
	return nil
}

func initParamDefs() *ParamDefs {

	param := newParamDefs()

	gasPriceMin := buildUint64ParamDef(getDefaultValueDefs(IDX_GASPRICE_MIN))
	blockFixAward := buildUint64ParamDef(getDefaultValueDefs(IDX_BLOCK_FIX_AWARD))
	voteCntMin := buildUint64ParamDef(getDefaultValueDefs(IDX_VOTER_CNT_MIN))
	voteDepositMin := buildUint64ParamDef(getDefaultValueDefs(IDX_VOTER_DEPOSIT_MIN))
	voteTotalDepositMin := buildUint64ParamDef(getDefaultValueDefs(IDX_VOTER_TOTAL_DEPOSIT_MIN))

	param.AddParam(gasPriceMin)
	param.AddParam(blockFixAward)
	param.AddParam(voteCntMin)
	param.AddParam(voteDepositMin)
	param.AddParam(voteTotalDepositMin)
	return param
}
