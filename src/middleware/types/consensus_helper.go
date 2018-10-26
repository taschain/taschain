package types

import (
	"math/big"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/9/26 下午6:39
**  Description: 
*/

type GenesisInfo struct {
	Group Group
	VrfPKs map[int][]byte
}

type ConsensusHelper interface {

	//generate genesis group and member pk info
	GenerateGenesisInfo() *GenesisInfo

	//vrf prove to value
	VRFProve2Value(prove *big.Int) *big.Int

	//bonus for proposal a block
	ProposalBonus() *big.Int

	//bonus for packing one bonus transaction
	PackBonus() *big.Int

	//calcaulate the blockheader's qn
	//it needs to be equal to the blockheader's totalQN - preHeader's totalQN
	CalculateQN(bh *BlockHeader) uint64

	//generate verify hash of the block for current node
	VerifyHash(b *Block) common.Hash

	//check the prove root hash for weight node when add block on chain
	CheckProveRoot(bh *BlockHeader) (bool, error)

	//check the new block
	//mainly verify the cast legality, group signature
	VerifyNewBlock(bh *BlockHeader, preBH *BlockHeader) (bool, error)

	//verify the blockheader: mainly verify the group signature
	VerifyBlockHeader(bh *BlockHeader) (bool, error)
}