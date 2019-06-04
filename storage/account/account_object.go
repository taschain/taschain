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

func (c Code) String() string {
	return string(c) //strings.Join(Disassemble(c), " ")
}

type Storage map[string][]byte

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
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

func (ao *accountObject) empty() bool {
	return ao.data.Nonce == 0 && ao.data.Balance.Sign() == 0 && bytes.Equal(ao.data.CodeHash, emptyCodeHash[:])
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

func (ao *accountObject) Encode(w io.Writer) error {
	return serialize.Encode(w, ao.data)
}

func (ao *accountObject) setError(err error) {
	if ao.dbErr == nil {
		ao.dbErr = err
	}
}

func (ao *accountObject) markSuicided() {
	ao.suicided = true
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) touch() {
	ao.db.transitions = append(ao.db.transitions, touchChange{
		account:   &ao.address,
		prev:      ao.touched,
		prevDirty: ao.onDirty == nil,
	})
	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
	ao.touched = true
}

func (ao *accountObject) getTrie(db AccountDatabase) Trie {
	if ao.trie == nil {
		var err error
		ao.trie, err = db.OpenStorageTrie(ao.addrHash, ao.data.Root)
		if err != nil {
			ao.trie, _ = db.OpenStorageTrie(ao.addrHash, common.Hash{})
			ao.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return ao.trie
}

func (ao *accountObject) GetData(db AccountDatabase, key string) []byte {
	ao.cachedLock.RLock()
	value, exists := ao.cachedStorage[key]
	ao.cachedLock.RUnlock()
	if exists {
		return value
	}

	value, err := ao.getTrie(db).TryGet([]byte(key))
	if err != nil {
		ao.setError(err)
		return nil
	}

	if value != nil {
		ao.cachedLock.Lock()
		ao.cachedStorage[key] = value
		ao.cachedLock.Unlock()
	}
	return value
}

func (ao *accountObject) SetData(db AccountDatabase, key string, value []byte) {
	ao.db.transitions = append(ao.db.transitions, storageChange{
		account:  &ao.address,
		key:      key,
		prevalue: ao.GetData(db, key),
	})
	ao.setData(key, value)
}

func (ao *accountObject) RemoveData(db AccountDatabase, key string) {
	ao.SetData(db, key, nil)
}

func (ao *accountObject) setData(key string, value []byte) {
	ao.cachedLock.Lock()
	ao.cachedStorage[key] = value
	ao.cachedLock.Unlock()
	ao.dirtyStorage[key] = value

	if ao.onDirty != nil {
		ao.onDirty(ao.Address())
		ao.onDirty = nil
	}
}

func (ao *accountObject) updateTrie(db AccountDatabase) Trie {
	tr := ao.getTrie(db)
	for key, value := range ao.dirtyStorage {
		delete(ao.dirtyStorage, key)
		if value == nil {
			ao.setError(tr.TryDelete([]byte(key)))
			continue
		}

		ao.setError(tr.TryUpdate([]byte(key), value[:]))
	}
	return tr
}

func (ao *accountObject) updateRoot(db AccountDatabase) {
	ao.updateTrie(db)
	ao.data.Root = ao.trie.Hash()
}

func (ao *accountObject) CommitTrie(db AccountDatabase) error {
	ao.updateTrie(db)
	if ao.dbErr != nil {
		return ao.dbErr
	}
	root, err := ao.trie.Commit(nil)
	if err == nil {
		ao.data.Root = root
	}
	return err
}

func (ao *accountObject) AddBalance(amount *big.Int) {

	if amount.Sign() == 0 {
		if ao.empty() {
			ao.touch()
		}

		return
	}
	ao.SetBalance(new(big.Int).Add(ao.Balance(), amount))
}

func (ao *accountObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	ao.SetBalance(new(big.Int).Sub(ao.Balance(), amount))
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

func (ao *accountObject) ReturnGas(gas *big.Int) {}

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

func (ao *accountObject) Address() common.Address {
	return ao.address
}

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
	prevCode := ao.Code(ao.db.db)
	ao.db.transitions = append(ao.db.transitions, codeChange{
		account:  &ao.address,
		prevhash: ao.CodeHash(),
		prevcode: prevCode,
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
