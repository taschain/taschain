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

type StateInfoReqMessage struct {
	StateInfoReqByte []byte

	Peer string
}

func (m *StateInfoReqMessage) GetRaw() []byte {
	return nil
}
func (m *StateInfoReqMessage) GetData() interface{} {
	return m
}

type StateInfoMessage struct {
	StateInfoByte []byte

	Peer string
}

func (m *StateInfoMessage) GetRaw() []byte {
	return nil
}
func (m *StateInfoMessage) GetData() interface{} {
	return m
}


type BlockReqMessage struct {
	HeightByte []byte

	Peer string
}

func (m *BlockReqMessage) GetRaw() []byte {
	return nil
}
func (m *BlockReqMessage) GetData() interface{} {
	return m
}

type TotalQnMessage struct {
	BlockHeaderByte []byte

	Peer string
}

func (m *TotalQnMessage) GetRaw() []byte {
	return m.BlockHeaderByte
}

func (m *TotalQnMessage) GetData() interface{} {
	return m
}

type ChainPieceReqMessage struct {
	HeightByte []byte

	Peer string
}

func (m *ChainPieceReqMessage) GetRaw() []byte {
	return nil
}
func (m *ChainPieceReqMessage) GetData() interface{} {
	return m
}

type ChainPieceMessage struct {
	ChainPieceByte []byte

	Peer string
}

func (m *ChainPieceMessage) GetRaw() []byte {
	return nil
}
func (m *ChainPieceMessage) GetData() interface{} {
	return m
}