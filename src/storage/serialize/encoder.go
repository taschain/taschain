package serialize

import (
	"io"
	"encoding/gob"
	"bytes"
)

type Encoder interface {
	Encode(io.Writer) error
}

func Encode(w io.Writer, val interface{}) error {
	switch value := val.(type) {
	case Encoder:
		value.Encode(w)
	default:
		encoder := gob.NewEncoder(w)
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