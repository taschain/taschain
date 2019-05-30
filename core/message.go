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

package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/pb"
	"github.com/taschain/taschain/middleware/types"
)

/*
**  Creator: pxf
**  Date: 2019/3/14 下午2:53
**  Description:
 */

type MessageBase struct {
}

type BlockResponseMessage struct {
	Blocks []*types.Block
}

type BlockPieceReqMessage struct {
	BeginHash common.Hash
	EndHash   common.Hash
}

type SyncCandidateInfo struct {
	Candidate       string
	CandidateHeight uint64
	ReqHeight       uint64
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

type SyncRequest struct {
	ReqHeight uint64
	ReqSize   int32
}

func MarshalSyncRequest(r *SyncRequest) ([]byte, error) {
	pbr := &tas_middleware_pb.SyncRequest{
		ReqSize:   &r.ReqSize,
		ReqHeight: &r.ReqHeight,
	}
	return proto.Marshal(pbr)
}

func UnmarshalSyncRequest(b []byte) (*SyncRequest, error) {
	m := new(tas_middleware_pb.SyncRequest)
	e := proto.Unmarshal(b, m)
	if e != nil {
		return nil, e
	}
	return &SyncRequest{ReqHeight: *m.ReqHeight, ReqSize: *m.ReqSize}, nil
}
