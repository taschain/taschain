package governance

import (
	"common"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:02
**  Description: 
*/

type AccountCreditManager interface {
	GetAccountCredit(ac common.Address) *AccountCredit
	SaveAccountCredit(ac *AccountCredit) bool
	StartEval() bool
}

type CreditPool struct {
	lock sync.RWMutex
	cache map[common.Address]AccountCredit
}

/**
* @Description: 读取一个账号的信用信息, todo: 从存储中读取部分 @鸠兹
* @Param:
* @return:
*/
func (p *CreditPool) GetAccountCredit(ac common.Address) (*AccountCredit) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cache == nil {
		return nil
	}
	if v, ok := p.cache[ac]; ok {
		return &v
	}
	return nil
}

/**
* @Description:  保存信用信息, todo: 持久化(@鸠兹)
* @Param:
* @return:
*/
func (p *CreditPool) SaveAccountCredit(credit *AccountCredit) (ok bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.cache == nil {
		p.cache = make(map[common.Address]AccountCredit)
	}

	p.cache[credit.Account] = *credit
	return true
}

/**
* @Description: 开始计算所有账户信用等级 todo: 等待链结构
* @Param:
* @return:
*/
func (p *CreditPool) StartEval() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return true
}



