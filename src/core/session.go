package core

import (
	"vm/core/vm"

	c "common"

	"math/big"
	"errors"
	"vm/core"
	"vm/common"
)

type Session struct {
	state vm.StateDB

	// 交易相关参数
	nonce    uint64
	source   *c.Address
	target   *c.Address
	value    *big.Int
	data     []byte
	gasLimit uint64
	gasPrice *big.Int

	// 当前剩余gas数量
	gas        uint64
	initialGas uint64

	gp *core.GasPool
}

func NewSession(state vm.StateDB, tx *Transaction, gp *core.GasPool, realdata []byte) *Session {
	session := &Session{
		state:    state,
		nonce:    tx.Nonce,
		source:   tx.Source,
		target:   tx.Target,
		value:    new(big.Int).SetUint64(tx.Value),
		data:     tx.Data,
		gasLimit: tx.GasLimit,
		gasPrice: new(big.Int).SetUint64(tx.GasPrice),
		gp:       gp,
	}

	// todo: 为问勤定制的需求，日后要用接口暴露出去
	if nil != realdata && len(realdata) > 0 {
		session.data = realdata
	}
	return session
}

var (
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

func (st *Session) preCheck() error {
	sender := st.from()
	nonce := st.state.GetNonce(sender.Address())
	if nonce < st.nonce {
		return ErrNonceTooHigh
	} else if nonce > st.nonce {
		return ErrNonceTooLow
	}
	return st.buyGas()
}

func (st *Session) from() vm.AccountRef {
	f := common.BytesToAddress(st.source.Bytes())
	if !st.state.Exist(f) {
		st.state.CreateAccount(f)
	}
	return vm.AccountRef(f)
}

func (st *Session) buyGas() error {
	var (
		state  = st.state
		sender = st.from()
	)
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.gasLimit), st.gasPrice)
	if state.GetBalance(sender.Address()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	if err := st.gp.SubGas(st.gasLimit); err != nil {
		return err
	}
	st.gas += st.gasLimit
	st.initialGas = st.gasLimit
	state.SubBalance(sender.Address(), mgval)
	return nil
}

func (st *Session) gasUsed() uint64 {
	return st.initialGas - st.gas
}

func (st *Session) refundGas() {
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	sender := st.from()

	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	st.state.AddBalance(sender.Address(), remaining)

	st.gp.AddGas(st.gas)
}

func (st *Session) Run(evm *vm.EVM) (ret []byte, usedGas uint64, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	sender := st.from() // err checked in preCheck

	// Pay intrinsic gas
	//gas, err := IntrinsicGas(st.data, contractCreation, homestead)
	//if err != nil {
	//	return nil, 0, false, err
	//}
	//if err = st.useGas(gas); err != nil {
	//	return nil, 0, false, err
	//}

	var vmerr error

	contractCreation := st.target == nil
	if contractCreation {
		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(sender.Address(), st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = evm.Call(sender, common.BytesToAddress(st.target.Bytes()), st.data, st.gas, st.value)
	}
	if vmerr != nil {
		//log.Debug("VM returned with error", "err", vmerr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return nil, 0, false, vmerr
		}
	}

	st.refundGas()

	//todo: 矿工分钱
	// st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))

	return ret, st.gasUsed(), vmerr != nil, err
}
