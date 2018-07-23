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
