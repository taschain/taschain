package p2p

import (
	"utility"
	"taslog"
)

const (
	GROUP_INIT_MSG = 0x00

	KEY_PIECE_MSG = 0x01

	MEMBER_PUBKEY_MSG = 0x02

	GROUP_INIT_DONE_MSG = 0x03

	CURRENT_GROUP_CAST_MSG = 0x04

	CAST_VERIFY_MSG = 0x05

	VARIFIED_CAST_MSG = 0x06

	REQ_TRANSACTION_MSG = 0x07

	TRANSACTION_MSG = 0x08

	NEW_BLOCK_MSG = 0x09

	REQ_BLOCK_CHAIN_HEIGHT_MSG = 0x0a

	BLOCK_CHAIN_HEIGHT_MSG = 0x0b

	REQ_BLOCK_MSG = 0x0c

	BLOCK_MSG = 0x0d

	REQ_GROUP_CHAIN_HEIGHT_MSG = 0x0e

	GROUP_CHAIN_HEIGHT_MSG = 0x0f

	REQ_GROUP_MSG = 0x10

	GROUP_MSG = 0x11
)

type Serializer interface {
	Marshal() ([]byte, error)

	Unmarshal([]byte) error
}
type Message struct {
	code uint32

	sign Serializer

	entity Serializer
}

func MarshalMessage(m Message) ([]byte, error) {

	b1 := utility.UInt32ToByte(m.code)

	b2, e1 := m.sign.Marshal()
	if e1 != nil {
		taslog.P2pLogger.Error("Sign marshal error!\n" + e1.Error())
		return nil, e1
	}

	b3, e2 := m.entity.Marshal()
	if e2 != nil {
		taslog.P2pLogger.Error("Entity marshal error!\n" + e2.Error())
		return nil, e2
	}
	b := make([]byte, len(b1)+len(b2)+len(b3))
	copy(b, b1)
	copy(b[len(b1):len(b1)+len(b2)], b2)
	copy(b[len(b1)+len(b2):], b3)
	return b, nil
}
