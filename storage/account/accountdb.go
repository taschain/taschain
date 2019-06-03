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

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/storage/serialize"
	"github.com/taschain/taschain/storage/trie"
	"golang.org/x/crypto/sha3"
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
	nextRevisionID int

	lock sync.Mutex
}

//func NewAccountDBWithMap(root common.Hash, db AccountDatabase, nodes map[string]*[]byte) (*AccountDB, error) {
//	tr, err := db.OpenTrieWithMap(root, nodes)
//	if err != nil {
//		return nil, err
//	}
//	return &AccountDB{
//		db:   db,
//		trie: tr,
//		//accountObjects:      make(map[common.Address]*accountObject),
//		accountObjectsDirty: make(map[common.Address]struct{}),
//	}, nil
//}

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

func (adb *AccountDB) GetTrie() Trie {
	return adb.trie
}

func (adb *AccountDB) setError(err error) {
	if adb.dbErr == nil {
		adb.dbErr = err
	}
}

func (adb *AccountDB) Error() error {
	return adb.dbErr
}

func (adb *AccountDB) Reset(root common.Hash) error {
	tr, err := adb.db.OpenTrie(root)
	if err != nil {
		return err
	}
	adb.trie = tr
	adb.accountObjects = new(sync.Map)
	adb.accountObjectsDirty = make(map[common.Address]struct{})
	adb.thash = common.Hash{}
	adb.bhash = common.Hash{}
	adb.txIndex = 0
	adb.logSize = 0
	adb.clearJournalAndRefund()
	return nil
}

func (adb *AccountDB) AddRefund(gas uint64) {
	adb.transitions = append(adb.transitions, refundChange{prev: adb.refund})
	adb.refund += gas
}

func (adb *AccountDB) Exist(addr common.Address) bool {
	return adb.getAccountObject(addr) != nil
}

func (adb *AccountDB) Empty(addr common.Address) bool {
	so := adb.getAccountObject(addr)
	return so == nil || so.empty()
}

func (adb *AccountDB) GetBalance(addr common.Address) *big.Int {
	accountObject := adb.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.Balance()
	}
	return common.Big0
}

func (adb *AccountDB) GetNonce(addr common.Address) uint64 {
	accountObject := adb.getAccountObject(addr)
	if accountObject != nil {
		return accountObject.Nonce()
	}

	return 0
}

func (adb *AccountDB) GetCode(addr common.Address) []byte {
	stateObject := adb.getAccountObject(addr)
	if stateObject != nil {
		return stateObject.Code(adb.db)
	}
	return nil
}

func (adb *AccountDB) GetCodeSize(addr common.Address) int {
	stateObject := adb.getAccountObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}
	size, err := adb.db.ContractCodeSize(stateObject.addrHash, common.BytesToHash(stateObject.CodeHash()))
	if err != nil {
		adb.setError(err)
	}
	return size
}

func (adb *AccountDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := adb.getAccountObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (adb *AccountDB) GetData(a common.Address, b string) []byte {
	stateObject := adb.getAccountObject(a)
	if stateObject != nil {
		return stateObject.GetData(adb.db, b)
	}
	return nil
}

func (adb *AccountDB) RemoveData(a common.Address, b string) {
	adb.SetData(a, b, nil)
}

func (adb *AccountDB) Database() AccountDatabase {
	return adb.db
}

func (adb *AccountDB) StorageTrie(a common.Address) Trie {
	stateObject := adb.getAccountObject(a)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(adb, nil)
	return cpy.updateTrie(adb.db)
}

func (adb *AccountDB) HasSuicided(addr common.Address) bool {
	stateObject := adb.getAccountObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

func (adb *AccountDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

func (adb *AccountDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(amount)
	}
}

func (adb *AccountDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
	}
}

func (adb *AccountDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (adb *AccountDB) SetCode(addr common.Address, code []byte) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetCode(sha3.Sum256(code), code)
	}
}

func (adb *AccountDB) SetData(addr common.Address, key string, value []byte) {
	stateObject := adb.getOrNewAccountObject(addr)
	if stateObject != nil {
		stateObject.SetData(adb.db, key, value)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's account object is still available until the account is committed,
// getAccountObject will return a non-nil account after Suicide.
func (adb *AccountDB) Suicide(addr common.Address) bool {
	stateObject := adb.getAccountObject(addr)
	if stateObject == nil {
		return false
	}
	adb.transitions = append(adb.transitions, suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}

func (adb *AccountDB) updateAccountObject(stateObject *accountObject) {
	addr := stateObject.Address()
	data, err := serialize.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't serialize object at %x: %v", addr[:], err))
	}
	adb.setError(adb.trie.TryUpdate(addr[:], data))
}

func (adb *AccountDB) deleteAccountObject(stateObject *accountObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	adb.setError(adb.trie.TryDelete(addr[:]))
}

func (adb *AccountDB) getAccountObjectFromTrie(addr common.Address) (stateObject *accountObject) {
	enc, err := adb.trie.TryGet(addr[:])
	if len(enc) == 0 {
		adb.setError(err)
		return nil
	}
	var data Account
	if err := serialize.DecodeBytes(enc, &data); err != nil {
		common.DefaultLogger.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}

	obj := newAccountObject(adb, addr, data, adb.MarkAccountObjectDirty)
	//adb.setAccountObject(obj)
	return obj
}

func (adb *AccountDB) getAccountObject(addr common.Address) (stateObject *accountObject) {

	if obj, ok := adb.accountObjects.Load(addr); ok {
		obj2 := obj.(*accountObject)
		if obj2.deleted {
			return nil
		}
		return obj2
	}

	obj := adb.getAccountObjectFromTrie(addr)
	if obj != nil {
		adb.setAccountObject(obj)
	}
	return obj
}

func (adb *AccountDB) setAccountObject(object *accountObject) {
	adb.accountObjects.Store(object.Address(), object)
}

func (adb *AccountDB) getOrNewAccountObject(addr common.Address) *accountObject {
	stateObject := adb.getAccountObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = adb.createObject(addr)
	}
	return stateObject
}

func (adb *AccountDB) MarkAccountObjectDirty(addr common.Address) {
	adb.accountObjectsDirty[addr] = struct{}{}
}

func (adb *AccountDB) createObject(addr common.Address) (newobj, prev *accountObject) {
	prev = adb.getAccountObject(addr)
	newobj = newAccountObject(adb, addr, Account{}, adb.MarkAccountObjectDirty)
	newobj.setNonce(0) // sets the object to dirty
	if prev == nil {
		adb.transitions = append(adb.transitions, createObjectChange{account: &addr})
	} else {
		adb.transitions = append(adb.transitions, resetObjectChange{prev: prev})
	}
	adb.setAccountObject(newobj)
	return newobj, prev
}

func (adb *AccountDB) CreateAccount(addr common.Address) {
	new, prev := adb.createObject(addr)
	if prev != nil {
		new.setBalance(prev.data.Balance)
	}
}

func (adb *AccountDB) DataIterator(addr common.Address, prefix string) *trie.Iterator {
	stateObject := adb.getAccountObjectFromTrie(addr)
	if stateObject != nil {
		return stateObject.DataIterator(adb.db, []byte(prefix))
	}
	return nil
}

func (adb *AccountDB) DataNext(iterator uintptr) string {
	iter := (*trie.Iterator)(unsafe.Pointer(iterator))
	if iter == nil {
		return `{"key":"","value":"","hasValue":0}`
	}
	hasValue := 1
	var key string
	var value string
	if len(iter.Key) != 0 {
		key = string(iter.Key)
		value = string(iter.Value)
	}
	if !iter.Next() { //no data
		hasValue = 0
	}
	if key == "" {
		return fmt.Sprintf(`{"key":"","value":"","hasValue":%d}`, hasValue)
	}
	if len(value) > 0 {
		valueType := value[0:1]
		if valueType == "0" { //this is map node
			hasValue = 2
		} else {
			value = value[1:]
		}
	} else {
		return `{"key":"","value":"","hasValue":0}`
	}
	return fmt.Sprintf(`{"key":"%s","value":%s,"hasValue":%d}`, key, value, hasValue)
}

func (adb *AccountDB) Copy() *AccountDB {
	adb.lock.Lock()
	defer adb.lock.Unlock()

	state := &AccountDB{
		db:   adb.db,
		trie: adb.trie,
		//accountObjects:      make(map[common.Address]*accountObject, len(adb.accountObjectsDirty)),
		accountObjectsDirty: make(map[common.Address]struct{}, len(adb.accountObjectsDirty)),
		refund:              adb.refund,
		logSize:             adb.logSize,
	}

	for addr := range adb.accountObjectsDirty {
		//state.accountObjects[addr] = adb.accountObjects[addr].deepCopy(state, state.MarkAccountObjectDirty)
		state.accountObjectsDirty[addr] = struct{}{}
	}
	return state
}

func (adb *AccountDB) Snapshot() int {
	id := adb.nextRevisionID
	adb.nextRevisionID++
	adb.validRevisions = append(adb.validRevisions, revision{id, len(adb.transitions)})
	return id
}

func (adb *AccountDB) RevertToSnapshot(revid int) {

	idx := sort.Search(len(adb.validRevisions), func(i int) bool {
		return adb.validRevisions[i].id >= revid
	})
	if idx == len(adb.validRevisions) || adb.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := adb.validRevisions[idx].journalIndex

	for i := len(adb.transitions) - 1; i >= snapshot; i-- {
		adb.transitions[i].undo(adb)
	}
	adb.transitions = adb.transitions[:snapshot]

	adb.validRevisions = adb.validRevisions[:idx]
}

func (adb *AccountDB) GetRefund() uint64 {
	return adb.refund
}

func (adb *AccountDB) Finalise(deleteEmptyObjects bool) {
	for addr := range adb.accountObjectsDirty {
		object, _ := adb.accountObjects.Load(addr)
		accountObject := object.(*accountObject)
		if accountObject.suicided || (deleteEmptyObjects && accountObject.empty()) {
			adb.deleteAccountObject(accountObject)
		} else {
			accountObject.updateRoot(adb.db)
			adb.updateAccountObject(accountObject)
		}
	}

	adb.clearJournalAndRefund()
}

func (adb *AccountDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	adb.Finalise(deleteEmptyObjects)
	return adb.trie.Hash()
}

func (adb *AccountDB) Prepare(thash, bhash common.Hash, ti int) {
	adb.thash = thash
	adb.bhash = bhash
	adb.txIndex = ti
}

func (adb *AccountDB) DeleteSuicides() {
	adb.clearJournalAndRefund()

	for addr := range adb.accountObjectsDirty {
		object, _ := adb.accountObjects.Load(addr)
		accountObject := object.(*accountObject)

		if accountObject.suicided {
			accountObject.deleted = true
		}
		delete(adb.accountObjectsDirty, addr)
	}
}

func (adb *AccountDB) clearJournalAndRefund() {
	adb.transitions = nil
	adb.validRevisions = adb.validRevisions[:0]
	adb.refund = 0
}

func (adb *AccountDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer adb.clearJournalAndRefund()
	var e *error
	adb.accountObjects.Range(func(key, value interface{}) bool {
		//for addr, accountObject := range adb.accountObjects {
		addr := key.(common.Address)
		_, isDirty := adb.accountObjectsDirty[addr]
		accountObject := value.(*accountObject)
		switch {
		case accountObject.suicided || (isDirty && deleteEmptyObjects && accountObject.empty()):

			adb.deleteAccountObject(accountObject)
		case isDirty:

			if accountObject.code != nil && accountObject.dirtyCode {
				adb.db.TrieDB().InsertBlob(common.BytesToHash(accountObject.CodeHash()), accountObject.code)
				accountObject.dirtyCode = false
			}

			if err := accountObject.CommitTrie(adb.db); err != nil {
				e = &err
				return false
				//return common.Hash{}, err
			}

			adb.updateAccountObject(accountObject)
		}
		delete(adb.accountObjectsDirty, addr)
		return true
	})
	if e != nil {
		return common.Hash{}, *e
	}

	root, err = adb.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := serialize.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyData {
			adb.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			adb.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	//adb.db.PushTrie(root, adb.trie)
	return root, err
}
