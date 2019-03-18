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