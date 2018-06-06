package sync

import (
	"time"
	"core"
	"sync"
	"utility"
	"network/p2p"
	"pb"
	"github.com/gogo/protobuf/proto"
	"log"
)


const (
	BLOCK_TOTAL_QN_RECEIVE_INTERVAL = 5 * time.Second

	BLOCK_SYNC_INTERVAL = 10 * time.Second
)

var BlockSyncer blockSyncer

type blockSyncer struct {
	neighborMaxTotalQN uint64     //邻居节点的最大高度
	bestNodeId         string     //最佳同步节点
	maxTotalQNLock     sync.Mutex //同步锁

	TotalQNRequestCh chan string
	TotalQNCh        chan core.EntityTotalQNMessage
	BlockRequestCh   chan core.EntityRequestMessage
	BlockArrivedCh   chan core.BlockArrivedMessage
}

func InitBlockSyncer() {
	BlockSyncer = blockSyncer{neighborMaxTotalQN: 0, TotalQNRequestCh: make(chan string), TotalQNCh: make(chan core.EntityTotalQNMessage),
		BlockRequestCh: make(chan core.EntityRequestMessage), BlockArrivedCh: make(chan core.BlockArrivedMessage),}
	go BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.TotalQNRequestCh:
			//logger.Debugf("[BlockSyncer] TotalQNRequestCh get message from:%s\n", sourceId)
			//收到块高度请求
			if nil == core.BlockChainImpl {
				return
			}
			sendBlockTotalQN(sourceId, core.BlockChainImpl.TotalQN())
		case h := <-bs.TotalQNCh:
			//logger.Debugf("[BlockSyncer] TotalQNCh get message from:%s,it's totalQN is:%d\n", h.SourceId, h.TotalQN)
			//收到来自其他节点的块链高度
			bs.maxTotalQNLock.Lock()
			if h.TotalQN > bs.neighborMaxTotalQN {
				bs.neighborMaxTotalQN = h.TotalQN
				bs.bestNodeId = h.SourceId
			}
			bs.maxTotalQNLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//logger.Debugf("[BlockSyncer] BlockRequestCh get message from:%s,request height:%d,request current hash:%s\n", br.SourceId, br.SourceHeight, br.SourceCurrentHash.String())
			//收到块请求
			if nil == core.BlockChainImpl {
				return
			}
			sendBlocks(br.SourceId, core.BlockChainImpl.GetBlockMessage(br.SourceHeight, br.SourceCurrentHash))
		case bm := <-bs.BlockArrivedCh:
			//logger.Debugf("[BlockSyncer] BlockArrivedCh get message from:%s\n", bm.SourceId)
			//收到块信息
			if nil == core.BlockChainImpl {
				return
			}
			blocks := bm.BlockEntity.Blocks
			if blocks != nil && len(blocks) != 0 {
				log.Printf("[BlockSyncer] BlockArrivedCh get block,length:%d\n", len(blocks))
				for i := 0; i < len(blocks); i++ {
					block := blocks[i]
					code := core.BlockChainImpl.AddBlockOnChain(block)
					if code < 0 {
						panic("fail to add block to block chain")
					}
					//todo 如果将来改为发送多次 此处需要修改
					if i == len(blocks)-1 {
						core.BlockChainImpl.SetAdujsting(false)
					}
				}
			} else {
				blockHashes := bm.BlockEntity.BlockHashes
				log.Printf("[BlockSyncer] BlockArrivedCh get block hashes,length:%d,lowest height:%d\n", len(blockHashes), blockHashes[len(blockHashes)-1].Height)
				if blockHashes == nil {
					return
				}
				blockHash, hasCommonAncestor := core.FindCommonAncestor(blockHashes, 0, len(blockHashes)-1)
				if hasCommonAncestor {
					log.Printf("[BlockSyncer]Got common ancestor! Height:%d\n", blockHash.Height)
					core.RequestBlockByHeight(bm.SourceId, blockHash.Height, blockHash.Hash)
				} else {
					cbhr := core.ChainBlockHashesReq{Height: blockHashes[len(blockHashes)-1].Height, Length: uint64(len(blockHashes) * 10)}
					log.Printf("[BlockSyncer]Do not find common ancestor!Request hashes form node:%s,base height:%d,length:%d\n", bm.SourceId, cbhr.Height, cbhr.Length)
					core.RequestBlockChainHashes(bm.SourceId, cbhr)
				}
			}
		case <-t.C:
			//logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			bs.syncBlock()
		}
	}
}

func (bs *blockSyncer) syncBlock() {
	go requestBlockChainTotalQN()
	t := time.NewTimer(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)

	<-t.C
	//获取本地块链高度
	if nil == core.BlockChainImpl {
		return
	}

	localTotalQN, localHeight, currentHash := core.BlockChainImpl.TotalQN(), core.BlockChainImpl.Height(), core.BlockChainImpl.QueryTopBlock().Hash
	bs.maxTotalQNLock.Lock()
	maxTotalQN := bs.neighborMaxTotalQN
	bestNodeId := bs.bestNodeId
	bs.maxTotalQNLock.Unlock()
	if maxTotalQN <= localTotalQN {
		//logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!\n", maxTotalQN, localTotalQN)
		return
	} else {
		//logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d is greater than self chain's totalQN: %d.\nSync from %s!", maxTotalQN, localTotalQN, bestNodeId)
		if core.BlockChainImpl.IsAdujsting() {
			log.Printf("[BlockSyncer]Local chain is adujsting, don't sync")
			return
		}
		core.RequestBlockByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链的QN值
func requestBlockChainTotalQN() {
	message := p2p.Message{Code: p2p.REQ_BLOCK_CHAIN_TOTAL_QN_MSG}
	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//返回自身链QN值
func sendBlockTotalQN(targetId string, localTotalQN uint64) {
	body := utility.UInt64ToByte(localTotalQN)
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_TOTAL_QN_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
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
func marshalBlockMessage(e *core.BlockMessage) ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	blocks := make([]*tas_pb.Block, 0)

	if e.Blocks != nil {
		for _, b := range e.Blocks {
			pb := core.BlockToPb(b)
			if pb == nil {
				logger.Errorf("Block is nil while marshalBlockMessage")
			}
			blocks = append(blocks, pb)
		}
	}
	blockSlice := tas_pb.BlockSlice{Blocks: blocks}

	cbh := make([]*tas_pb.ChainBlockHash, 0)

	if e.BlockHashes != nil {
		for _, b := range e.BlockHashes {
			pb := core.ChainBlockHashToPb(b)
			if pb == nil {
				logger.Errorf("ChainBlockHash is nil while marshalBlockMessage")
			}
			cbh = append(cbh, pb)
		}
	}
	cbhs := tas_pb.ChainBlockHashSlice{ChainBlockHashes: cbh}

	message := tas_pb.BlockMessage{Blocks: &blockSlice, ChainBlockHash: &cbhs}
	return proto.Marshal(&message)
}
