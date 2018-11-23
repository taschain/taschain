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
	VrfPKs [][]byte
	Pks 	[][]byte
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