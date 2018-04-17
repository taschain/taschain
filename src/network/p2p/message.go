package p2p

import (
	"pb"
	"github.com/gogo/protobuf/proto"
	"taslog"
)

type Message struct {
	Code uint32

	Sign []byte

	Body []byte
}

func MarshalMessage(m Message) ([]byte, error) {
	message := tas_pb.Message{Code: &m.Code, Signature: m.Sign, Body: m.Body}
	return proto.Marshal(&message)
}

func UnMarshalMessage(b []byte) (*Message, error) {
	message := new(tas_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		taslog.P2pLogger.Errorf("Unmarshal message error:%s\n", e.Error())
		return nil, e
	}
	m := Message{Code: *message.Code, Sign: message.Signature, Body: message.Body}
	return &m, nil
}
