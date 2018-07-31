package serialize

import (
	"bytes"
	"encoding/gob"
	"io"
)

func Decode(r io.Reader, val interface{}) error {
	decoder := gob.NewDecoder(r)
	if err := decoder.Decode(val); err != nil {
		return err
	}
	return nil
}

func DecodeBytes(b []byte, val interface{}) error {
	return Decode(bytes.NewBuffer(b), val)
}
