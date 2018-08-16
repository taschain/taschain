package notify

import "middleware/types"

type BlockMessage struct {
	Block types.Block
}

func (m *BlockMessage) GetRaw() []byte {
	return []byte{}
}
func (m *BlockMessage) GetData() interface{} {
	return m.Block
}

type GroupMessage struct {
	Group types.Group
}

func (m *GroupMessage) GetRaw() []byte {
	return []byte{}
}
func (m *GroupMessage) GetData() interface{} {
	return m.Group
}

type BlockHeaderNotifyMessage struct {
	HeaderByte []byte

	Peer string
}

func (m *BlockHeaderNotifyMessage) GetRaw() []byte {
	return nil
}

func (m *BlockHeaderNotifyMessage) GetData() interface{} {
	return m
}

type BlockBodyReqMessage struct {
	BlockHashByte []byte

	Peer string
}

func (m *BlockBodyReqMessage) GetRaw() []byte {
	return nil
}

func (m *BlockBodyReqMessage) GetData() interface{} {
	return m
}

type BlockBodyNotifyMessage struct {
	BodyByte []byte

	Peer string
}

func (m *BlockBodyNotifyMessage) GetRaw() []byte {
	return nil
}
func (m *BlockBodyNotifyMessage) GetData() interface{} {
	return m
}
