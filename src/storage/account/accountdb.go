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
	"fmt"
	"math/big"
	"sort"
	"sync"

	"storage/trie"
	"storage/serialize"
	"golang.org/x/crypto/sha3"
	"common"
	"unsafe"
)

type revision struct {
	id           int
	journalIndex int
}

var (
	emptyData = sha3.Sum256(nil)

	emptyCode = sha3.Sum256(nil)
)

type AccountDB struct {
	db   AccountDatabase
	trie Trie

	//accountObjects      map[common.Address]*accountObject
	accountObjects      *sync.Map
	accountObjectsDirty map[common.Address]struct{}

	dbErr error

	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logSize      uint

	transitions    transition
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

func NewAccountDBWithMap(root common.Hash, db AccountDatabase, nodes map[string]*[]byte) (*AccountDB, error) {
	tr, err := db.OpenTrieWithMap(root, nodes)
	if err != nil {
		return nil, err
	}
	return &AccountDB{
		db:   db,
		trie: tr,
		//accountObjects:      make(map[common.Address]*accountObject),
		accountObjectsDirty: make(map[common.Address]struct{}),
	}, nil
}

func NewAccountDB(root common.Hash, db AccountDatabase) (*AccountDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	accountdb := &AccountDB{
		db:   db,
		trie: tr,
		//accountObjects:      make(map[common.Address]*accountObject),
		accountObjects:      new(sync.Map),
		accountObjectsDirty: make(map[common.Address]struct{}),
	}
	return accountdb, nil
}

func (self *AccountDB) GetTrie() Trie {
	return self.trie
}

func (self *AccountDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *AccountDB) Error() error {
	return self.dbErr
}

func (self *AccountDB) Reset(root common.Hash) error {
	tr, err := self.db.OpenTrie(root)
	if err != nil {
		return err
	}
	self.trie = tr
	self.accountObjects = new(sync.Map)
	self.accountObjectsDirty = make(map[common.Address]struct{})
	self.thash = common.Hash{}
	self.bhash = common.Hash{}
	self.txIndex = 0
	self.logSize = 0
	self.clearJournalAndRefund()
	return nil
}

func (self *AccountDB) AddRefund(gas uint64) {
	self.transitions = append(self.transitions, refundChange{prev: self.refund})
	self.refund += gas
}

func (self *AccountDB) Exist(addr common.Address) bool {
	return self.getAccountObject(addr) != nil
}

func (self *AccountDB) Empty(addr common.Address) bool {
	so := self.getAccountObject(addr)
	return so == nil || so.empty()
}

func (self *AccountDB) GetBalance(addr common.Address) *big.Int {
	accountObject := self.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.Balance()
	}
	return common.Big0
}

func (self *AccountDB) GetNonce(addr common.Address) uint64 {
	accountObject := self.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.Nonce()
	}

	return 0
}

func (self *AccountDB) GetCode(addr common.Address) []byte {
	stateObject := self.getAccountObject(addr)
	if stateObject != nil {
		return stateObject.Code(self.db)
	}
	return nil
}

func (self *AccountDB) GetCodeSize(addr common.Address) int {
	stateObject := self.getAccountObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}
	size, err := self.db.ContractCodeSize(stateObject.addrHash, common.BytesToHash(stateObject.CodeHash()))
	if err != nil {
		self.setError(err)
	}
	return size
}

func (self *AccountDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := self.getAccountObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (self *AccountDB) GetData(a common.Address, b string) []byte {
	stateObject := self.getAccountObject(a)
	if stateObject != nil {
		return stateObject.GetData(self.db, b)
	}
	return nil
}

func (self *AccountDB) RemoveData(a common.Address, b string) {
	self.SetData(a, b, nil)
}

func (self *AccountDB) Database() AccountDatabase {
	return self.db
}

func (self *AccountDB) StorageTrie(a common.Address) Trie {
	stateObject := self.getAccountObject(a)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(self, nil)
	return cpy.updateTrie(self.db)
}

func (self *AccountDB) HasSuicided(addr common.Address) bool {
	stateObject := self.getAccountObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

func (self *AccountDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (self *AccountDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (self *AccountDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (self *AccountDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (self *AccountDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetCode(sha3.Sum256(code), code)
	}
}

func (self *AccountDB) SetData(addr common.Address, key string, value []byte) {
	stateObject := self.GetOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetData(self.db, key, value)
	}
}

func (self *AccountDB) Suicide(addr common.Address) bool {
	stateObject := self.getAccountObject(addr)
	if stateObject == nil {
		return false
	}
	self.transitions = append(self.transitions, suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}

func (self *AccountDB) updateAccountObject(stateObject *accountObject) {
	addr := stateObject.Address()
	data, err := serialize.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't serialize object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

func (self *AccountDB) deleteAccountObject(stateObject *accountObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	self.setError(self.trie.TryDelete(addr[:]))
}

func (self *AccountDB) getAccountObjectFromTrie(addr common.Address) (stateObject *accountObject) {
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Account
	if err := serialize.DecodeBytes(enc, &data); err != nil {
		common.DefaultLogger.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}

	obj := newAccountObject(self, addr, data, self.MarkAccountObjectDirty)
	//self.setAccountObject(obj)
	return obj
}

func (self *AccountDB) getAccountObject(addr common.Address) (stateObject *accountObject) {

	if obj, ok := self.accountObjects.Load(addr); ok {
		obj2 := obj.(*accountObject)
		if obj2.deleted {
			return nil
		}
		return obj2
	}

	obj := self.getAccountObjectFromTrie(addr)
	if obj != nil {
		self.setAccountObject(obj)
	}
	return obj
}

func (self *AccountDB) setAccountObject(object *accountObject) {
	self.accountObjects.Store(object.Address(), object)
}

func (self *AccountDB) GetOrNewAccountObject(addr common.Address) *accountObject {
	stateObject := self.getAccountObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}

func (self *AccountDB) MarkAccountObjectDirty(addr common.Address) {
	self.accountObjectsDirty[addr] = struct{}{}
}

func (self *AccountDB) createObject(addr common.Address) (newobj, prev *accountObject) {
	prev = self.getAccountObject(addr)
	newobj = newAccountObject(self, addr, Account{}, self.MarkAccountObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	if prev == nil {
		self.transitions = append(self.transitions, createObjectChange{account: &addr})
	} else {
		self.transitions = append(self.transitions, resetObjectChange{prev: prev})
	}
	self.setAccountObject(newobj)
	return newobj, prev
}

func (self *AccountDB) CreateAccount(addr common.Address) {
	new, prev := self.createObject(addr)
	if prev != nil {
		new.setBalance(prev.data.Balance)
	}
}

func (self *AccountDB) DataIterator(addr common.Address, prefix string) *trie.Iterator {
	stateObject := self.getAccountObjectFromTrie(addr)
	if stateObject != nil {
		return stateObject.DataIterator(self.db, []byte(prefix))
	} else {
		return nil
	}
}

func (self *AccountDB) DataNext(iterator uintptr) string  {
	iter := (*trie.Iterator)(unsafe.Pointer(iterator))
	if iter == nil{
		return `{"key":"","value":"","hasValue":0}`
	}
	hasValue := 1
	var key string = ""
	var value string = ""
	if len(iter.Key) != 0 {
		key = string(iter.Key)
		value = string(iter.Value)
	}
	if !iter.Next(){//no data
		hasValue = 0
	}
	if key == "" {
		return fmt.Sprintf(`{"key":"","value":"","hasValue":%d}`, hasValue)
	}
	if len(value) > 0{
		valueType := value[0:1]
		if valueType == "0"{//this is map node
			hasValue = 2
		}else{
			value= value[1:]
		}
	}else{
		return `{"key":"","value":"","hasValue":0}`
	}
	return fmt.Sprintf(`{"key":"%s","value":%s,"hasValue":%d}`,key,value,hasValue)
}

func (self *AccountDB) Copy() *AccountDB {
	self.lock.Lock()
	defer self.lock.Unlock()

	state := &AccountDB{
		db:   self.db,
		trie: self.trie,
		//accountObjects:      make(map[common.Address]*accountObject, len(self.accountObjectsDirty)),
		accountObjectsDirty: make(map[common.Address]struct{}, len(self.accountObjectsDirty)),
		refund:              self.refund,
		logSize:             self.logSize,
	}

	for addr := range self.accountObjectsDirty {
		//state.accountObjects[addr] = self.accountObjects[addr].deepCopy(state, state.MarkAccountObjectDirty)
		state.accountObjectsDirty[addr] = struct{}{}
	}
	return state
}

func (self *AccountDB) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, len(self.transitions)})
	return id
}

func (self *AccountDB) RevertToSnapshot(revid int) {

	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex

	for i := len(self.transitions) - 1; i >= snapshot; i-- {
		self.transitions[i].undo(self)
	}
	self.transitions = self.transitions[:snapshot]

	self.validRevisions = self.validRevisions[:idx]
}

func (self *AccountDB) GetRefund() uint64 {
	return self.refund
}

func (s *AccountDB) Finalise(deleteEmptyObjects bool) {
	for addr := range s.accountObjectsDirty {
		object, _ := s.accountObjects.Load(addr)
		accountObject := object.(*accountObject)
		if accountObject.suicided || (deleteEmptyObjects && accountObject.empty()) {
			s.deleteAccountObject(accountObject)
		} else {
			accountObject.updateRoot(s.db)
			s.updateAccountObject(accountObject)
		}
	}

	s.clearJournalAndRefund()
}

func (s *AccountDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.Finalise(deleteEmptyObjects)
	return s.trie.Hash()
}

func (self *AccountDB) Prepare(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (s *AccountDB) DeleteSuicides() {
	s.clearJournalAndRefund()

	for addr := range s.accountObjectsDirty {
		object, _ := s.accountObjects.Load(addr)
		accountObject := object.(*accountObject)

		if accountObject.suicided {
			accountObject.deleted = true
		}
		delete(s.accountObjectsDirty, addr)
	}
}

func (s *AccountDB) clearJournalAndRefund() {
	s.transitions = nil
	s.validRevisions = s.validRevisions[:0]
	s.refund = 0
}

func (s *AccountDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer s.clearJournalAndRefund()
	var e *error
	s.accountObjects.Range(func(key, value interface{}) bool {
		//for addr, accountObject := range s.accountObjects {
		addr := key.(common.Address)
		_, isDirty := s.accountObjectsDirty[addr]
		accountObject := value.(*accountObject)
		switch {
		case accountObject.suicided || (isDirty && deleteEmptyObjects && accountObject.empty()):

			s.deleteAccountObject(accountObject)
		case isDirty:

			if accountObject.code != nil && accountObject.dirtyCode {
				s.db.TrieDB().Insert(common.BytesToHash(accountObject.CodeHash()), accountObject.code)
				accountObject.dirtyCode = false
			}

			if err := accountObject.CommitTrie(s.db); err != nil {
				e = &err
				return false
				//return common.Hash{}, err
			}

			s.updateAccountObject(accountObject)
		}
		delete(s.accountObjectsDirty, addr)
		return true
	})
	if e != nil {
		return common.Hash{}, *e
	}

	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := serialize.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyData {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	//s.db.PushTrie(root, s.trie)
	common.DefaultLogger.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}
