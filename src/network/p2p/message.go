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
