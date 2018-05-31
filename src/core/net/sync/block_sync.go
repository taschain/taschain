package sync

import (
	"time"
	"core"
	"sync"
	"common"
	"utility"
	"network/p2p"
	"taslog"
	"pb"
	"github.com/gogo/protobuf/proto"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

const (
	BLOCK_HEIGHT_RECEIVE_INTERVAL = 5 * time.Second

	BLOCK_SYNC_INTERVAL = 10 * time.Second
)

var BlockSyncer blockSyncer

type blockSyncer struct {
	neighborMaxHeight uint64     //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan string
	HeightCh        chan core.EntityHeightMessage
	BlockRequestCh  chan core.EntityRequestMessage
	BlockArrivedCh  chan core.BlockArrivedMessage
}

func InitBlockSyncer() {
	BlockSyncer = blockSyncer{neighborMaxHeight: 0, HeightRequestCh: make(chan string), HeightCh: make(chan core.EntityHeightMessage),
		BlockRequestCh: make(chan core.EntityRequestMessage), BlockArrivedCh: make(chan core.BlockArrivedMessage),}
	go BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.HeightRequestCh:
			//logger.Debugf("BlockSyncer HeightRequestCh get message from:%s", sourceId)
			//收到块高度请求
			if nil == core.BlockChainImpl {
				return
			}
			sendBlockHeight(sourceId, core.BlockChainImpl.Height())
		case h := <-bs.HeightCh:
			//logger.Debugf("BlockSyncer HeightCh get message from:%s,it's height is:%d", h.SourceId, h.Height)
			//收到来自其他节点的块链高度
			bs.maxHeightLock.Lock()
			if h.Height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.Height
				bs.bestNodeId = h.SourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//logger.Debugf("BlockRequestCh get message from:%s\n,current height:%d,current hash:%s", br.SourceId, br.SourceHeight, br.SourceCurrentHash.String())
			//收到块请求
			if nil == core.BlockChainImpl {
				return
			}
			sendBlocks(br.SourceId, core.BlockChainImpl.GetBlockMessage(br.SourceHeight, br.SourceCurrentHash))
		case bm := <-bs.BlockArrivedCh:
			//logger.Debugf("BlockArrivedCh get message from:%s,block length:%v", bm.SourceId, len(bm.BlockEntity.Blocks))
			//收到块信息
			if nil == core.BlockChainImpl {
				return
			}
			e := core.BlockChainImpl.AddBlockMessage(bm.BlockEntity)
			if e != nil {
				logger.Debugf("Block chain add block error:%s", e.Error())
			}
		case <-t.C:
			//logger.Debug("sync time up, start to block sync!")
			bs.syncBlock()
		}
	}
}

func (bs *blockSyncer) syncBlock() {
	go requestBlockChainHeight()
	t := time.NewTimer(BLOCK_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	//logger.Debug("block height request  time up!")
	//获取本地块链高度
	if nil == core.BlockChainImpl {
		return
	}

	localHeight, currentHash := core.BlockChainImpl.Height(), core.BlockChainImpl.QueryTopBlock().Hash
	bs.maxHeightLock.Lock()
	maxHeight := bs.neighborMaxHeight
	bestNodeId := bs.bestNodeId
	bs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		//logger.Debugf("Neighbor max block height: %d,is less than self block height: %d .Don't sync!", maxHeight, localHeight)
		return
	} else {
		//logger.Debugf("Neighbor max block height: %d is greater than self block height: %d.Sync from %s!", maxHeight, localHeight, bestNodeId)
		requestBlockByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链高度
func requestBlockChainHeight() {
	message := p2p.Message{Code: p2p.REQ_BLOCK_CHAIN_HEIGHT_MSG}
	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//返回自身链高度
func sendBlockHeight(targetId string, localHeight uint64) {
	body := utility.UInt64ToByte(localHeight)
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_HEIGHT_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//向某一节点请求Block
func requestBlockByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := core.EntityRequestMessage{SourceHeight: localHeight, SourceCurrentHash: currentHash}
	body, e := marshalEntityRequestMessage(&m)
	if e != nil {
		logger.Errorf("requestBlockByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_BLOCK_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

//本地查询之后将结果返回
func sendBlocks(targetId string, blockEntity *core.BlockMessage) {
	body, e := marshalBlockMessage(blockEntity)
	if e != nil {
		logger.Errorf("sendBlocks marshal BlockEntity error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//----------------------------------------------块同步------------------------------------------------------------------
func marshalEntityRequestMessage(e *core.EntityRequestMessage) ([]byte, error) {
	sourceHeight := e.SourceHeight
	currentHash := e.SourceCurrentHash.Bytes()
	sourceId := []byte(e.SourceId)

	m := tas_pb.EntityRequestMessage{SourceHeight: &sourceHeight, SourceCurrentHash: currentHash, SourceId: sourceId}
	return proto.Marshal(&m)
}

func marshalBlockMessage(e *core.BlockMessage) ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	blocks := make([]*tas_pb.Block, 0)

	if e.Blocks != nil {
		for _, b := range e.Blocks {
			pb := core.BlockToPb(b)
			if pb == nil{
				logger.Errorf("Block is nil while marshalBlockMessage")
			}
			blocks = append(blocks, pb)
		}
	}
	blockSlice := tas_pb.BlockSlice{Blocks: blocks}

	height := e.Height

	hashes := make([][]byte, 0)
	if e.BlockHashes != nil {
		for _, h := range e.BlockHashes {
			hashes = append(hashes, h.Bytes())
		}
	}
	hashSlice := tas_pb.Hashes{Hashes: hashes}

	ratioSlice := tas_pb.Ratios{Ratios: e.BlockRatios}

	message := tas_pb.BlockMessage{Blocks: &blockSlice, Height: &height, Hashes: &hashSlice, Ratios: &ratioSlice}
	return proto.Marshal(&message)
}
