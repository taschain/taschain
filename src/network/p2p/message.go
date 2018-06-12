package p2p

import (
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
	"log"
)

type Message struct {
	Code uint32

	Sign []byte

	Body []byte
}

func MarshalMessage(m Message) ([]byte, error) {
	message := tas_middleware_pb.Message{Code: &m.Code, Signature: m.Sign, Body: m.Body}
	return proto.Marshal(&message)
}

func UnMarshalMessage(b []byte) (*Message, error) {
	message := new(tas_middleware_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		log.Printf("Unmarshal message error:%s", e.Error())
		return nil, e
	}
	m := Message{Code: *message.Code, Sign: message.Signature, Body: message.Body}
	return &m, nil
}
