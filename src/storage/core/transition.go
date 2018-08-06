package core

import (
	"math/big"
	"common"
)

type transitionEntry interface {
	undo(*AccountDB)
}

type transition []transitionEntry

type (
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *accountObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool
		prevbalance *big.Int
	}

	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key string
		prevalue []byte
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *AccountDB) {
	delete(s.accountObjects, *ch.account)
	delete(s.accountObjectsDirty, *ch.account)
}

func (ch resetObjectChange) undo(s *AccountDB) {
	s.setAccountObject(ch.prev)
}

func (ch suicideChange) undo(s *AccountDB) {
	obj := s.getAccountObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *AccountDB) {
	if !ch.prev && *ch.account != ripemd {
		s.getAccountObject(*ch.account).touched = ch.prev
		if !ch.prevDirty {
			delete(s.accountObjectsDirty, *ch.account)
		}
	}
}

func (ch balanceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setBalance(ch.prev)
}

func (ch nonceChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setNonce(ch.prev)
}

func (ch codeChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch storageChange) undo(s *AccountDB) {
	s.getAccountObject(*ch.account).setData(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *AccountDB) {
	s.refund = ch.prev
}
