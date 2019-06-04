package core

import (
	"github.com/gogo/protobuf/proto"
	"github.com/taschain/taschain/middleware/pb"
	"github.com/taschain/taschain/middleware/types"
)

/*
**  Creator: pxf
**  Date: 2019/3/14 下午2:53
**  Description:
 */

type blockResponseMessage struct {
	Blocks []*types.Block
}

type SyncCandidateInfo struct {
	Candidate       string
	CandidateHeight uint64
	ReqHeight       uint64
}

type syncMessage struct {
	CandidateInfo *SyncCandidateInfo
}

func (msg *syncMessage) GetRaw() []byte {
	panic("implement me")
}

func (msg *syncMessage) GetData() interface{} {
	return msg.CandidateInfo
}

type syncRequest struct {
	ReqHeight uint64
	ReqSize   int32
}

func marshalSyncRequest(r *syncRequest) ([]byte, error) {
	pbr := &tas_middleware_pb.SyncRequest{
		ReqSize:   &r.ReqSize,
		ReqHeight: &r.ReqHeight,
	}
	return proto.Marshal(pbr)
}

func unmarshalSyncRequest(b []byte) (*syncRequest, error) {
	m := new(tas_middleware_pb.SyncRequest)
	e := proto.Unmarshal(b, m)
	if e != nil {
		return nil, e
	}
	return &syncRequest{ReqHeight: *m.ReqHeight, ReqSize: *m.ReqSize}, nil
}
