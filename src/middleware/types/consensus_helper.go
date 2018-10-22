package types

import "math/big"

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

	//generateo genesis group and member pk info
	GenerateGenesisInfo() *GenesisInfo

	//vrf prove 2 value
	VRFProve2Value(prove *big.Int) *big.Int

	//bonus for proposal a block
	ProposalBonus() *big.Int

	//bonus for packing one bonus transaction
	PackBonus() *big.Int

	//calcaulate the blockheader qn
	//it needs to be equal to the totalQN - preHeader totalQN
	CalculateQN(bh *BlockHeader) uint64

}