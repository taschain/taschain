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

package vm

import (
	"math/big"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/trie"
)

type AccountDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	GetRefund() uint64

	GetData(common.Address, string) []byte
	SetData(common.Address, string, []byte)
	RemoveData(common.Address, string)
	DataIterator(common.Address, string) *trie.Iterator
	DataNext(iterator uintptr) string

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	Exist(common.Address) bool
	Empty(common.Address) bool

	RevertToSnapshot(int)
	Snapshot() int
}

type ChainReader interface {
	Height() uint64
	QueryTopBlock() *types.BlockHeader
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader
	QueryBlockHeaderByHeight(height uint64) *types.BlockHeader
	HasBlock(hash common.Hash) bool
	HasHeight(height uint64) bool
}
