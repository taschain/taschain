package core

import (
	"common"
	"time"
)

type BlockHeader struct {
	parent               common.Hash
	hash                 common.Hash
	castor               common.Address
	castorWeight         uint32
	group                common.Address
	consensusVersion     uint32
	gmtCreate            time.Time
	transactions         []common.Hash
	maxTransactionLength uint16
	txTree               common.Hash
	receiptTree          common.Hash
	stateTree            common.Hash
	nonce                int32
	extraData            []int8
	maxExtraDataLength   uint16

}

type SignedBlockHeader struct {
	//签名
	signature common.Hash
	//区块头
	header *BlockHeader
}

type Block struct {
	header       *SignedBlockHeader
	transactions []*Transaction
}
