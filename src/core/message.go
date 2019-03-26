package core

import (
	"middleware/types"
	"common"
)

/*
**  Creator: pxf
**  Date: 2019/3/14 下午2:53
**  Description: 
*/

type MessageBase struct {

}

type BlockResponseMessage struct {
	Blocks       []*types.Block
}



type BlockPieceReqMessage struct {
	BeginHash common.Hash
	EndHash common.Hash
}

type SyncCandidateInfo struct {
	Candidate string
	CandidateHeight uint64
	ReqHeight uint64
}

type SyncMessage struct {
	CandidateInfo *SyncCandidateInfo
}

func (msg *SyncMessage) GetRaw() []byte {
	panic("implement me")
}

func (msg *SyncMessage) GetData() interface{} {
	return msg.CandidateInfo
}


type transactionRequestMessage struct {
	TransactionHashes []common.Hash
	CurrentBlockHash  common.Hash
	//BlockHeight       uint64
	//BlockPv           *big.Int
}
