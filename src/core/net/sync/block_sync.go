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
	BLOCK_HEIGHT_RECEIVE_INTERVAL = 60 * time.Second

	BLOCK_SYNC_INTERVAL = 3 * time.Minute
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
	BlockSyncer = blockSyncer{HeightRequestCh: make(chan string), HeightCh: make(chan core.EntityHeightMessage),
		BlockRequestCh: make(chan core.EntityRequestMessage), BlockArrivedCh: make(chan core.BlockArrivedMessage),}
	BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.HeightRequestCh:
			logger.Infof("HeightRequestCh get message from:%s\n",sourceId)
			//收到块高度请求
			if nil == core.BlockChainImpl {
				return
			}
			//todo for test
			//sendBlockHeight(sourceId, 111)
			sendBlockHeight(sourceId, core.BlockChainImpl.Height())
		case h := <-bs.HeightCh:
			logger.Infof("HeightCh get message from:%s\n",h.SourceId)
			//收到来自其他节点的块链高度
			bs.maxHeightLock.Lock()
			if h.Height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.Height
				bs.bestNodeId = h.SourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			logger.Infof("BlockRequestCh get message from:%s\n,current height:%s,current hash:%s",br.SourceId,br.SourceHeight,br.SourceCurrentHash)
			//收到块请求
			if nil == core.BlockChainImpl {
				return
			}
			sendBlocks(br.SourceId, core.BlockChainImpl.GetBlockMessage(br.SourceHeight, br.SourceCurrentHash))
		case bm := <-bs.BlockArrivedCh:
			logger.Infof("BlockArrivedCh get message from:%s,hash:%v\n",bm.SourceId,bm.BlockEntity.BlockHashes)
			//收到块信息
			if nil == core.BlockChainImpl {
				return
			}
			core.BlockChainImpl.AddBlockMessage(bm.BlockEntity)
		case <-t.C:
			logger.Info("sync time up start to sync!\n")
			bs.syncBlock()
		}
	}
}

func (bs *blockSyncer) syncBlock() {
	bs.maxHeightLock.Lock()
	bs.neighborMaxHeight = 0
	bs.bestNodeId = ""
	bs.maxHeightLock.Unlock()

	go requestBlockChainHeight()
	t := time.NewTimer(BLOCK_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	logger.Info("block height request  time up!\n")
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
		logger.Infof("Neighbor max block height %d is less than self block height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		logger.Infof("Neighbor max block height %d is greater than self block height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
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
		logger.Errorf("requestBlockByHeight marshal EntityRequestMessage error:%s\n", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_BLOCK_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

//本地查询之后将结果返回
func sendBlocks(targetId string, blockEntity *core.BlockMessage) {
	body, e := marshalBlockMessage(blockEntity)
	if e != nil {
		logger.Errorf("sendBlocks marshal BlockEntity error:%s\n", e.Error())
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
	blocks := make([]*tas_pb.Block, 0)

	if e.Blocks != nil {
		for _, b := range e.Blocks {
			blocks = append(blocks, core.BlockToPb(b))
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

func mockBlock() *core.Block {
	txpool := core.BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		logger.Error("fail to get txpool")
	}

	// 交易1
	txpool.Add(genTestTx("tx1", 123, "111", "abc", 0, 1))

	//交易2
	txpool.Add(genTestTx("tx1", 456, "222", "ddd", 0, 1))

	// 铸块1
	block := core.BlockChainImpl.CastingBlock()
	if nil == block {
		logger.Error("fail to cast new block")
	}
	return block
}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *core.Transaction {

	sourcebyte := common.BytesToAddress(core.Sha256([]byte(source)))
	targetbyte := common.BytesToAddress(core.Sha256([]byte(target)))

	//byte: 84,104,105,115,32,105,115,32,97,32,116,114,97,110,115,97,99,116,105,111,110
	data := []byte("This is a transaction")
	return &core.Transaction{
		Data:     data,
		Value:    value,
		Nonce:    nonce,
		Source:   &sourcebyte,
		Target:   &targetbyte,
		GasPrice: price,
		GasLimit: 3,
		Hash:     common.BytesToHash(core.Sha256([]byte(hash))),
	}
}
