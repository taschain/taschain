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

package serialize

import (
	"io"
	"bytes"
	"github.com/vmihailenco/msgpack"
)

type Encoder interface {
	Encode(io.Writer) error
}

func Encode(w io.Writer, val interface{}) error {
	switch value := val.(type) {
	case Encoder:
		value.Encode(w)
	default:
		encoder := msgpack.NewEncoder(w)
		if err := encoder.Encode(val); err != nil {
			return err
		}
	}

	return nil
}

func EncodeBytes(b []byte, val interface{}) error {
	return Encode(bytes.NewBuffer(b), val)
}

func EncodeToBytes(val interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := Encode(buf, val)
	return buf.Bytes(), err
}