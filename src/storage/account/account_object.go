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

	"storage/serialize"
	"io"
	"golang.org/x/crypto/sha3"
	"common"
	"storage/trie"
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

	cachedLock	  sync.RWMutex
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

func (self *accountObject) SetBalance(amount *big.Int) {
	self.db.transitions = append(self.db.transitions, balanceChange{
		account: &self.address,
		prev:    new(big.Int).Set(self.data.Balance),
	})
	self.setBalance(amount)
}

func (self *accountObject) setBalance(amount *big.Int) {
	self.data.Balance = amount
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (c *accountObject) ReturnGas(gas *big.Int) {}

func (self *accountObject) deepCopy(db *AccountDB, onDirty func(addr common.Address)) *accountObject {
	accountObject := newAccountObject(db, self.address, self.data, onDirty)
	if self.trie != nil {
		accountObject.trie = db.db.CopyTrie(self.trie)
	}
	accountObject.code = self.code
	accountObject.dirtyStorage = self.dirtyStorage.Copy()
	accountObject.cachedStorage = self.dirtyStorage.Copy()
	accountObject.suicided = self.suicided
	accountObject.dirtyCode = self.dirtyCode
	accountObject.deleted = self.deleted
	return accountObject
}

func (c *accountObject) Address() common.Address {
	return c.address
}

func (self *accountObject) Code(db AccountDatabase) []byte {
	if self.code != nil {
		return self.code
	}
	if bytes.Equal(self.CodeHash(), emptyCodeHash[:]) {
		return nil
	}
	code, err := db.ContractCode(self.addrHash, common.BytesToHash(self.CodeHash()))
	if err != nil {
		self.setError(fmt.Errorf("can't load code hash %x: %v", self.CodeHash(), err))
	}
	self.code = code
	return code
}

func (self *accountObject) DataIterator(db AccountDatabase, prefix []byte) *trie.Iterator {
	if self.trie == nil {
		self.getTrie(db)
	}
	return trie.NewIterator(self.trie.NodeIterator([]byte(prefix)))
}

func (self *accountObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := self.Code(self.db.db)
	self.db.transitions = append(self.db.transitions, codeChange{
		account:  &self.address,
		prevhash: self.CodeHash(),
		prevcode: prevcode,
	})
	self.setCode(codeHash, code)
}

func (self *accountObject) setCode(codeHash common.Hash, code []byte) {
	self.code = code
	self.data.CodeHash = codeHash[:]
	self.dirtyCode = true
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) SetNonce(nonce uint64) {
	self.db.transitions = append(self.db.transitions, nonceChange{
		account: &self.address,
		prev:    self.data.Nonce,
	})
	self.setNonce(nonce)
}

func (self *accountObject) setNonce(nonce uint64) {
	self.data.Nonce = nonce
	if self.onDirty != nil {
		self.onDirty(self.Address())
		self.onDirty = nil
	}
}

func (self *accountObject) CodeHash() []byte {
	return self.data.CodeHash
}

func (self *accountObject) Balance() *big.Int {
	return self.data.Balance
}

func (self *accountObject) Nonce() uint64 {
	return self.data.Nonce
}

func (self *accountObject) Value() *big.Int {
	panic("Value on accountObject should never be called")
}

func (self *accountObject) fstring() string {
	if self.trie == nil {
		self.trie = self.getTrie(self.db.db)
	}
	return self.trie.Fstring()
}
