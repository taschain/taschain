package governance

/*
**  Creator: pxf
**  Date: 2018/4/25 上午9:45
**  Description: 
*/
import (
	"testing"
	"common"

	"core"
	"governance/global"
)

func TestBlockChain_AddBlock(t *testing.T) {

	common.InitConf("test.ini")
	core.Clear()
	core.InitCore()
	chain := core.BlockChainImpl
	chain.GetTransactionPool().Clear()
	//chain.Clear()

	global.InitGov(chain)

	// 查询创始块
	blockHeader := chain.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	txpool := chain.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))

	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))

	castor:=new([]byte)
	groupid:=new([]byte)

	// 铸块1
	block := chain.CastingBlock(1, 12, 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	if 0 != chain.AddBlockOnChain(block) {
		t.Fatalf("fail to add block")
	}

	//最新块是块1
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}


	//交易3
	txpool.Add(genTestTx("jdai3", 1, "1", "2", 1, 10))

	// 铸块2
	block2 := chain.CastingBlock(2, 123, 0, *castor, *groupid)
	block2.Header.QueueNumber = 2
	if 0 != chain.AddBlockOnChain(block2) {
		t.Fatalf("fail to add empty block")
	}


	//最新块是块2
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block2.Header.Hash || block.Header.Hash != block2.Header.PreHash {
		t.Fatalf("add block2 failed")
	}
	blockHeader = chain.QueryBlockByHash(block2.Header.Hash)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHash, hash: %x ", block2.Header.Hash)
	}

	blockHeader = chain.QueryBlockByHeight(2)
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHeight, height: %d ", 2)
	}

	// 铸块3 空块
	block3 := chain.CastingBlock(3, 125, 0, *castor, *groupid)
	if 0 != chain.AddBlockOnChain(block3) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	// 铸块4 空块
	// 模拟分叉
	block4 := chain.CastingBlockAfter(block.Header, 2, 1256, 0, *castor, *groupid)

	if 0 != chain.AddBlockOnChain(block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块4
	blockHeader = chain.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block4 failed")
	}
	blockHeader = chain.QueryBlockByHeight(3)
	if nil != blockHeader {
		t.Fatalf("failed to remove uncle blocks")
	}



}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *core.Transaction {

	sourcebyte := common.BytesToAddress(generateHash(source))
	targetbyte := common.BytesToAddress(generateHash(target))

	return &core.Transaction{
		GasPrice: price,
		Hash:     common.BytesToHash(generateHash(hash)),
		Source:   &sourcebyte,
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    value,
	}
}

func generateHash(hash string) []byte {
	bytes3 := []byte(hash)
	return core.Sha256(bytes3)
}