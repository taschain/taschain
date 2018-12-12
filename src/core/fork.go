package core

import (
	"time"
	"utility"
	"network"
	"middleware/notify"
	"middleware/types"
	"math/big"
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
	"common"
	"taslog"
)

const forkTimeOut = 3 * time.Second

type forkHandler struct {
	candidite string
	reqTimer  *time.Timer
	logger    taslog.Logger
}

func initForkHandler() *forkHandler {
	fh := forkHandler{}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceHandler)
	return &fh
}

func (fh *forkHandler) requestChainPieceInfo(targetNode string, height uint64) {
	if targetNode == "" {
		return
	}
	fh.candidite = targetNode
	fh.reqTimer.Reset(forkTimeOut)
	fh.logger.Debugf("Req chain piece info to:%s,local height:%d", targetNode, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkHandler) chainPieceInfoReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.ChainPieceReqMessage)
	if !ok {
		return
	}
	reqHeight := utility.ByteToUInt64(chainPieceReqMessage.HeightByte)
	id := chainPieceReqMessage.Peer

	chainPiece := BlockChainImpl.GetChainPiece(reqHeight)
	fh.sendChainPieceInfo(id, ChainPieceInfo{ChainPiece: chainPiece, TopHeader: BlockChainImpl.QueryTopBlock()})
}

func (fh *forkHandler) sendChainPieceInfo(targetNode string, chainPieceInfo ChainPieceInfo) {
	chainPiece := chainPieceInfo.ChainPiece
	if len(chainPiece) == 0 {
		return
	}
	fh.logger.Debugf("Send chain piece %d-%d to:%s", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, targetNode)
	body, e := marshalChainPieceInfo(chainPieceInfo)
	if e != nil {
		fh.logger.Errorf("Discard marshalChainPiece because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceInfo, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkHandler) chainPieceHandler(msg notify.Message) {
	chainPieceMessage, ok := msg.GetData().(*notify.ChainPieceMessage)
	if !ok {
		return
	}
	chainPieceInfo, err := unMarshalChainPieceInfo(chainPieceMessage.ChainPieceInfoByte)
	if err != nil {
		fh.logger.Errorf("unMarshalChainPiece error:%s", err.Error())
		return
	}
	fh.processChainPieceInfo(chainPieceMessage.Peer, chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader
}

func (fh *forkHandler) processChainPieceInfo(id string, chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) {
	if id != fh.candidite {
		return
	}

	localTopHeader := BlockChainImpl.QueryTopBlock()
	if topHeader.TotalQN < localTopHeader.TotalQN {
		return
	}

	if !fh.verifyChainPiece(chainPiece, topHeader) {
		return
	}
	fh.logger.Debugf("ProcessChainPiece id:%s,chainPiece %d-%d,topHeader height:%d,totalQn:%d,hash:%v",
		id, chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, topHeader.Height, topHeader.TotalQN, topHeader.Hash.Hex())
	commonAncestor, hasCommonAncestor, index := fh.findCommonAncestor(chainPiece, 0, len(chainPiece)-1)
	if hasCommonAncestor {
		fh.logger.Debugf("Got common ancestor! Height:%d,localHeight:%d", commonAncestor.Height, localTopHeader.Height)
		if topHeader.TotalQN > localTopHeader.TotalQN {
			RequestBlock(id, commonAncestor.Height+1)
			return
		}

		if topHeader.TotalQN == chain.latestBlock.TotalQN {
			var remoteNext *types.BlockHeader
			for i := index - 1; i >= 0; i-- {
				if chainPiece[i].ProveValue != nil {
					remoteNext = chainPiece[i]
					break
				}
			}
			if remoteNext == nil {
				return
			}
			if chain.compareValue(commonAncestor, remoteNext) {
				fh.logger.Debugf("Local value is great than coming value!")
				return
			}
			fh.logger.Debugf("Coming value is great than local value!")
			chain.removeFromCommonAncestor(commonAncestor)
			RequestBlock(id, commonAncestor.Height+1)
		}
	} else {
		if index == 0 {
			fh.logger.Debugf("Local chain is same with coming chain piece.")
			if chainPiece[0].Height == chain.latestBlock.Height {
				RequestBlock(id, chainPiece[0].Height+1)
				return
			}
			fh.logger.Debugf("Local height is more than chainPiece[0].Height. Ignore it!")
			return
		} else {
			fh.logger.Debugf("Do not find common ancestor!Request hashes form node:%s,base height:%d", id, chainPiece[len(chainPiece)-1].Height-1, )
			RequestChainPiece(id, chainPiece[len(chainPiece)-1].Height)
		}
	}
}

func (fh *forkHandler) verifyChainPiece(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 {
		return false
	}
	if topHeader.Hash != topHeader.GenHash() {
		fh.logger.Infof("invalid topHeader!Hash:%s", topHeader.Hash.String())
		return false
	}

	for i := 0; i < len(chainPiece)-1; i++ {
		bh := chainPiece[i]
		if bh.Hash != bh.GenHash() {
			fh.logger.Infof("invalid chainPiece element,hash:%s", bh.Hash.String())
			return false
		}
		if bh.PreHash != chainPiece[i+1].Hash {
			fh.logger.Infof("invalid preHash,expect prehash:%s,real hash:%s", bh.PreHash.String(), chainPiece[i+1].Hash.String())
			return false
		}
	}
	return true
}

func (fh *forkHandler) compareValue(commonAncestor *types.BlockHeader, remoteHeader *types.BlockHeader) bool {
	if commonAncestor.Height == chain.latestBlock.Height {
		return false
	}
	var localValue *big.Int
	remoteValue := chain.consensusHelper.VRFProve2Value(remoteHeader.ProveValue)
	fh.logger.Debugf("coming hash:%s,coming value is:%v", remoteHeader.Hash.String(), remoteValue)
	fh.logger.Debugf("compareValue hash:%s height:%d latestheight:%d", commonAncestor.Hash.Hex(), commonAncestor.Height, chain.latestBlock.Height)
	for height := commonAncestor.Height + 1; height <= chain.latestBlock.Height; height++ {
		fh.logger.Debugf("compareValue queryBlockHeaderByHeight height:%d ", height)
		header := chain.queryBlockHeaderByHeight(height, true)
		if header == nil {
			fh.logger.Debugf("compareValue queryBlockHeaderByHeight nil !height:%d ", height)
			continue
		}
		localValue = chain.consensusHelper.VRFProve2Value(header.ProveValue)
		fh.logger.Debugf("local hash:%s,local value is:%v", header.Hash.String(), localValue)
		break
	}
	if localValue == nil {
		time.Sleep(time.Second)
	}
	if localValue.Cmp(remoteValue) >= 0 {
		return true
	}
	return false
}

func (fh *forkHandler) findCommonAncestor(chainPiece []*types.BlockHeader, l int, r int) (*types.BlockHeader, bool, int) {
	if l > r {
		return nil, false, -1
	}

	m := (l + r) / 2
	result := fh.isCommonAncestor(chainPiece, m)
	if result == 0 {
		return chainPiece[m], true, m
	}

	if result == 1 {
		return fh.findCommonAncestor(chainPiece, l, m-1)
	}

	if result == -1 {
		return fh.findCommonAncestor(chainPiece, m+1, r)
	}
	if result == 100 {
		return nil, false, 0
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//100  当前HASH相等，但是到达数组边界，找不到后面一块 无法判断同祖先
//-1  当前HASH不相等
//-100 参数不合法
func (fh *forkHandler) isCommonAncestor(chainPiece []*types.BlockHeader, index int) int {
	if index < 0 || index >= len(chainPiece) {
		return -100
	}
	he := chainPiece[index]

	bh := chain.queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		fh.logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, nil, he.Hash)
		return -1
	}
	fh.logger.Debugf("isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 100
	}
	if index == 0 {
		return -1
	}
	//判断链更后面的一块
	afterHe := chainPiece[index-1]
	afterbh := chain.queryBlockHeaderByHeight(afterHe.Height, true)
	if afterbh == nil {
		fh.logger.Debugf("isCommonAncestor:after block height:%d,local hash:%s,coming hash:%x\n", afterHe.Height, "null", afterHe.Hash)
		if afterHe != nil && bh.Hash == he.Hash {
			return 0
		}
		return -1
	}
	fh.logger.Debugf("isCommonAncestor:after block height:%d,local hash:%x,coming hash:%x\n", afterHe.Height, afterbh.Hash, afterHe.Hash)
	if afterHe.Hash != afterbh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterbh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
}

func (fh *forkHandler) RequestChainPieceBlock(id string, height uint64) {
	fh.logger.Debugf("Req chain piece block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

func (fh *forkHandler) newBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(types.Block)
	if !ok {
		return
	}
	fh.logger.Debugf("Rcv new block hash:%v,height:%d,totalQn:%d,tx len:%d", m.Header.Hash.Hex(), m.Header.Height, m.Header.TotalQN, len(m.Transactions))
	//todo 收齐上链
}

func (fh *forkHandler)unMarshalChainPieceInfo(b []byte) (*ChainPieceInfo, error) {
	message := new(tas_middleware_pb.ChainPieceInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		fh.logger.Errorf("unMarshalChainPieceInfo error:%s", e.Error())
		return nil, e
	}

	chainPiece := make([]*types.BlockHeader, 0)
	for _, header := range message.BlockHeaders {
		h := types.PbToBlockHeader(header)
		chainPiece = append(chainPiece, h)
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	chainPieceInfo := ChainPieceInfo{ChainPiece: chainPiece, TopHeader: topHeader}
	return &chainPieceInfo, nil
}
