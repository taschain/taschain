package sync

import (
	"time"
	"core"
	"sync"
	"common"
	"utility"
	"network/p2p"
	"taslog"
	"fmt"
	"pb"
	"github.com/gogo/protobuf/proto"
)

var logger = taslog.GetLogger(taslog.P2PConfig)

const (
	BLOCK_HEIGHT_RECEIVE_INTERVAL = 30 * time.Second

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
			//收到块高度请求

			//todo 获取本地块链高度
			//type getLocalBlockChainHeightFn func() (uint64, common.Hash, error)
			//height,_,e := bs.getHeight()

			//height, e := uint64(10), errors.New("")
			//if e != nil {
			//	logger.Errorf("Get block height rquest from %s error:%s\n", sourceId, e.Error())
			//	return
			//}

			//for test
			height := uint64(100)

			sendBlockHeight(sourceId, height)
		case h := <-bs.HeightCh:
			//收到来自其他节点的块链高度
			bs.maxHeightLock.Lock()
			if h.Height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.Height
				bs.bestNodeId = h.SourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//收到块请求

			//todo 根据高度获取对应的block
			//type queryBlocksByHeightFn func(localHeight uint64, currentHash common.Hash) (*core.BlockMessage, error)
			//blockEntity, e := bs.queryBlock(br.Bre.SourceHeight, br.Bre.SourceCurrentHash)

			//blockEntity, e := new(core.BlockMessage), errors.New("")
			//if e != nil {
			//	logger.Errorf("query block request from %s error:%s\n", br.SourceId, e.Error())
			//	return
			//}

			//for test
			blockEntity := &core.BlockMessage{Blocks: []*core.Block{mockBlock()}}
			sendBlocks(br.SourceId, blockEntity)
		case bm := <-bs.BlockArrivedCh:
			//收到块信息

			//todo block上链
			//type addBlocksToChainFn func(blockEntity *core.BlockMessage, targetId string)error
			//bs.addBlocks(&bm.BlockEntity, bm.SourceId)
			fmt.Printf(bm.SourceId)
		case <-t.C:
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
	//todo 获取本地块链高度
	//localHeight, currentHash, e := bs.getLocalHeight()

	//localHeight, currentHash, e := uint64(0), common.BytesToHash([]byte{}), errors.New("")
	//if e != nil {
	//	logger.Errorf("Self get block height error:%s\n", e.Error())
	//	return
	//}
	localHeight, currentHash := uint64(0), common.BytesToHash([]byte{})
	bs.maxHeightLock.Lock()
	maxHeight := bs.neighborMaxHeight
	bestNodeId := bs.bestNodeId
	bs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		logger.Infof("Neightbor max block height %d is less than self block height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		logger.Infof("Neightbor max block height %d is greater than self block height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
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
			p2p.Server.SendMessage(message, string(id))
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
