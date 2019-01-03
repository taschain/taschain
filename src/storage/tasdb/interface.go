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

package tasdb

import "github.com/syndtr/goleveldb/leveldb/iterator"

const IdealBatchSize = 100 * 1024

type Putter interface {
	Put(key []byte, value []byte) error
}

type Batch interface {
	Putter
	ValueSize() int // amount of data in the batch
	Write() error
	// Reset resets the batch for reuse
	Reset()
}


type Database interface {
	Putter
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error
	Close()
	NewBatch() Batch
	NewIterator() iterator.Iterator
	NewIteratorWithPrefix(prefix []byte) iterator.Iterator
}