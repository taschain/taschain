package notify

import (
	"github.com/taschain/taschain/middleware/types"
)

type BlockOnChainSuccMessage struct {
	Block *types.Block
}

func (m *BlockOnChainSuccMessage) GetRaw() []byte {
	return []byte{}
}
func (m *BlockOnChainSuccMessage) GetData() interface{} {
	return m.Block
}

//--------------------------------------------------------------------------------------------------------------------
type GroupMessage struct {
	Group *types.Group
}

func (m *GroupMessage) GetRaw() []byte {
	return []byte{}
}
func (m *GroupMessage) GetData() interface{} {
	return m.Group
}

type DefaultMessage struct {
	body            []byte
	source          string
	chainId         uint16
	protocalVersion uint16
}

func (m *DefaultMessage) GetRaw() []byte {
	panic("implement me")
}

func (m *DefaultMessage) GetData() interface{} {
	return m.Body
}

func (m *DefaultMessage) Body() []byte {
	return m.body
}

func (m *DefaultMessage) Source() string {
	return m.source
}
func (m *DefaultMessage) ChainId() uint16 {
	return m.chainId
}
func (m *DefaultMessage) ProtocalVersion() uint16 {
	return m.protocalVersion
}

func NewDefaultMessage(body []byte, from string, chainId, protocal uint16) *DefaultMessage {
	return &DefaultMessage{body: body, source: from, chainId: chainId, protocalVersion: protocal}
}

func AsDefault(message Message) *DefaultMessage {
	return message.(*DefaultMessage)
}
