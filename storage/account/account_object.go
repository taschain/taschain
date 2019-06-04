//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package account

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/storage/serialize"
	"github.com/taschain/taschain/storage/trie"
	"golang.org/x/crypto/sha3"
	"io"
)

var emptyCodeHash = sha3.Sum256(nil)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[string][]byte

func (self Storage) String() (str string) {
	for key, value := range self {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

type accountObject struct {
	address  common.Address
	addrHash common.Hash
	data     Account
	db       *AccountDB

	dbErr error

	trie Trie
	code Code

	cachedLock    sync.RWMutex
	cachedStorage Storage
	dirtyStorage  Storage

	dirtyCode bool
	suicided  bool
	touched   bool
	deleted   bool
	onDirty   func(addr common.Address)
}

func (s *accountObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash[:])
}

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

func newAccountObject(db *AccountDB, address common.Address, data Account, onDirty func(addr common.Address)) *accountObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash[:]
	}
	return &accountObject{
		db:            db,
		address:       address,
		addrHash:      sha3.Sum256(address[:]),
		data:          data,
		cachedStorage: make(Storage),
		dirtyStorage:  make(Storage),
		onDirty:       onDirty,
	}
}

func (c *accountObject) Encode(w io.Writer) error {
	return serialize.Encode(w, c.data)
}

func (self *accountObject) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *accountObject) markSuicided() {
	self.suicided = true
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (c *accountObject) touch() {
	c.db.transitions = append(c.db.transitions, touchChange{
		account:   &c.address,
		prev:      c.touched,
		prevDirty: c.onDirty == nil,
	})
	if c.onDirty != nil {
		c.onDirty(c.Address())
		c.onDirty = nil
	}
	c.touched = true
}

func (c *accountObject) getTrie(db AccountDatabase) Trie {
	if c.trie == nil {
		var err error
		c.trie, err = db.OpenStorageTrie(c.addrHash, c.data.Root)
		if err != nil {
			c.trie, _ = db.OpenStorageTrie(c.addrHash, common.Hash{})
			c.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return c.trie
}

func (self *accountObject) GetData(db AccountDatabase, key string) []byte {
	self.cachedLock.RLock()
	value, exists := self.cachedStorage[key]
	self.cachedLock.RUnlock()
	if exists {
		return value
	}

	value, err := self.getTrie(db).TryGet([]byte(key))
	if err != nil {
		self.setError(err)
		return nil
	}

	if value != nil {
		self.cachedLock.Lock()
		self.cachedStorage[key] = value
		self.cachedLock.Unlock()
	}
	return value
}

func (self *accountObject) SetData(db AccountDatabase, key string, value []byte) {
	self.db.transitions = append(self.db.transitions, storageChange{
		account:  &self.address,
		key:      key,
		prevalue: self.GetData(db, key),
	})
	self.setData(key, value)
}

func (self *accountObject) RemoveData(db AccountDatabase, key string) {
	self.SetData(db, key, nil)
}

func (self *accountObject) setData(key string, value []byte) {
	self.cachedLock.Lock()
	self.cachedStorage[key] = value
	self.cachedLock.Unlock()
	self.dirtyStorage[key] = value

	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) updateTrie(db AccountDatabase) Trie {
	tr := self.getTrie(db)
	for key, value := range self.dirtyStorage {
		delete(self.dirtyStorage, key)
		if value == nil {
			self.setError(tr.TryDelete([]byte(key)))
			continue
		}

		self.setError(tr.TryUpdate([]byte(key), value[:]))
	}
	return tr
}

func (self *accountObject) updateRoot(db AccountDatabase) {
	self.updateTrie(db)
	self.data.Root = self.trie.Hash()
}

func (self *accountObject) CommitTrie(db AccountDatabase) error {
	self.updateTrie(db)
	if self.dbErr != nil {
		return self.dbErr
	}
	root, err := self.trie.Commit(nil)
	if err == nil {
		self.data.Root = root
		//self.db.db.PushTrie(root, self.trie)
	}
	return err
}

func (c *accountObject) AddBalance(amount *big.Int) {

	if amount.Sign() == 0 {
		if c.empty() {
			c.touch()
		}

		return
	}
	c.SetBalance(new(big.Int).Add(c.Balance(), amount))
}

func (c *accountObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	c.SetBalance(new(big.Int).Sub(c.Balance(), amount))
}

func (ao *accountObject) SetBalance(amount *big.Int) {
	ao.db.transitions = append(ao.db.transitions, balanceChange{
		account: &ao.address,
		prev:    new(big.Int).Set(ao.data.Balance),
	})
	ao.setBalance(amount)
}

func (ao *accountObject) setBalance(amount *big.Int) {
	ao.data.Balance = amount
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (c *accountObject) ReturnGas(gas *big.Int) {}

func (ao *accountObject) deepCopy(db *AccountDB, onDirty func(addr common.Address)) *accountObject {
	accountObject := newAccountObject(db, ao.address, ao.data, onDirty)
	if ao.trie != nil {
		accountObject.trie = db.db.CopyTrie(ao.trie)
	}
	accountObject.code = ao.code
	accountObject.dirtyStorage = ao.dirtyStorage.Copy()
	accountObject.cachedStorage = ao.dirtyStorage.Copy()
	accountObject.suicided = ao.suicided
	accountObject.dirtyCode = ao.dirtyCode
	accountObject.deleted = ao.deleted
	return accountObject
}

func (c *accountObject) Address() common.Address {
	return c.address
}

// Code returns the contract code associated with this object, if any.
func (ao *accountObject) Code(db AccountDatabase) []byte {
	if ao.code != nil {
		return ao.code
	}
	if bytes.Equal(ao.CodeHash(), emptyCodeHash[:]) {
		return nil
	}
	code, err := db.ContractCode(ao.addrHash, common.BytesToHash(ao.CodeHash()))
	if err != nil {
		ao.setError(fmt.Errorf("can't load code hash %x: %v", ao.CodeHash(), err))
	}
	ao.code = code
	return code
}

func (ao *accountObject) DataIterator(db AccountDatabase, prefix []byte) *trie.Iterator {
	if ao.trie == nil {
		ao.getTrie(db)
	}
	return trie.NewIterator(ao.trie.NodeIterator([]byte(prefix)))
}

func (ao *accountObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := ao.Code(ao.db.db)
	ao.db.transitions = append(ao.db.transitions, codeChange{
		account:  &ao.address,
		prevhash: ao.CodeHash(),
		prevcode: prevcode,
	})
	ao.setCode(codeHash, code)
}

func (ao *accountObject) setCode(codeHash common.Hash, code []byte) {
	ao.code = code
	ao.data.CodeHash = codeHash[:]
	ao.dirtyCode = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) SetNonce(nonce uint64) {
	ao.db.transitions = append(ao.db.transitions, nonceChange{
		account: &ao.address,
		prev:    ao.data.Nonce,
	})
	ao.setNonce(nonce)
}

func (ao *accountObject) setNonce(nonce uint64) {
	ao.data.Nonce = nonce
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) CodeHash() []byte {
	return ao.data.CodeHash
}

func (ao *accountObject) Balance() *big.Int {
	return ao.data.Balance
}

func (ao *accountObject) Nonce() uint64 {
	return ao.data.Nonce
}

func (ao *accountObject) Value() *big.Int {
	panic("Value on accountObject should never be called")
}
