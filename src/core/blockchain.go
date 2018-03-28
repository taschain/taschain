package core

import (
	"time"
	"common"
)

type BlockChain struct {
	blocks []*Block
}

func (chain *BlockChain) MakeBlockHeader(parent common.Hash256,
	hash common.Hash256,
	castor common.Address,
	castorWeight uint32,
	group common.Address,
	consensusVersion uint32,
	gmtCreate time.Time,
	transactions []common.Hash256,
	txTree common.Hash256,
	receiptTree common.Hash256,
	stateTree common.Hash256,
	nonce int32,
	extraData []int8) *Block {

	return nil
}

func (chain *BlockChain) MakeSignedBlockHeader(header *BlockHeader,
	verifier []common.Address,
	pubKeys []common.Hash256,
	signatures []common.Hash256) *SignedBlockHeader {
	return nil
}

func (chain *BlockChain) VerifyBlockHeader(header *SignedBlockHeader) int8 {
	return 0
}

func (chain *BlockChain) AddBlock(block Block) int8 {
	return 0
}
