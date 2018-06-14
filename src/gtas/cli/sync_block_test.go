package cli

import (
	"testing"
	"log"
	"core"
	"core/net/sync"
	"time"
)

//---------------------------------------测试同步一个块------------------------------------------------------------------
func TestSeedBlockSync(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	sync.InitBlockSyncer()

	time.Sleep(1 * time.Minute)
	printBlockInfo()
}

func TestClientBlockSync(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 1 {
		t.Fatal("client block chain not excepted after sync!")
	}
	printBlockInfo()
	time.Sleep(30 * time.Second)
}

//---------------------------------------测试同步三个块------------------------------------------------------------------
func TestSBS1(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	sync.InitBlockSyncer()

	time.Sleep(30 * time.Second)
	mockBlock(3, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())

	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS1(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 2 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 3 {
		t.Fatal("client block chain not excepted after sync2!")
	}
	printBlockInfo()
	time.Sleep(20 * time.Second)
}

//---------------------------------------测试组同步分叉，高度3和1，1开始分叉------------------------------------------------------------------
func TestSBS2(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	mockBlock(3, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	sync.InitBlockSyncer()
	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS2(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	mockBlock(1, 1)
	printBlockInfo()
	sync.InitBlockSyncer()
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	if core.BlockChainImpl.Height() != 3 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(20 * time.Second)
}

//---------------------------------------测试组同步分叉，高度10和10，1开始分叉------------------------------------------------------------------
func TestSBS3(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	sync.InitBlockSyncer()
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS3(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()

	//等待块同步
	mockBlock(2, 1)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	printBlockInfo()
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	if core.BlockChainImpl.Height() != 10 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(20 * time.Second)
}

//---------------------------------------测试组同步分叉，高度12和11，11开始分叉------------------------------------------------------------------
func TestSBS4(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	sync.InitBlockSyncer()
	time.Sleep(time.Second * 30)

	mockBlock(11, 4)
	mockBlock(12, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS4(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()

	for {
		if core.BlockChainImpl.Height() == 10 {
			mockBlock(11, 1)
			break
		}
	}
	printBlockInfo()
	if core.BlockChainImpl.Height() != 11 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	if core.BlockChainImpl.Height() != 12 {
		t.Fatal("client block chain not excepted after sync2!")
	}
	time.Sleep(20 * time.Second)
}

//---------------------------------------测试组同步分叉，高度12和11，6开始分叉------------------------------------------------------------------
func TestSBS5(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	sync.InitBlockSyncer()
	time.Sleep(time.Second * 30)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	mockBlock(11, 4)
	mockBlock(12, 4)
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS5(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()

	for {
		if core.BlockChainImpl.Height() == 5 {
			mockBlock(6, 3)
			mockBlock(7, 4)
			mockBlock(8, 4)
			mockBlock(9, 4)
			mockBlock(10, 4)
			mockBlock(11, 4)
			break
		}
	}
	printBlockInfo()
	if core.BlockChainImpl.Height() != 11 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	if core.BlockChainImpl.Height() != 12 {
		t.Fatal("client block chain not excepted after sync2!")
	}
	time.Sleep(20 * time.Second)
}

//---------------------------------------测试组同步分叉，高度12和12，1开始分叉,第二次获取BlockHash才能找到分叉点------------------------------------------------------------------
func TestSBS6(t *testing.T) {
	mockSeed()

	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	mockBlock(1, 4)
	mockBlock(2, 4)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	mockBlock(11, 4)
	mockBlock(12, 4)
	sync.InitBlockSyncer()
	log.Printf("seed block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	time.Sleep(1 * time.Minute)
}

func TestCBS6(t *testing.T) {
	mockClient(t)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	if core.BlockChainImpl.Height() != 0 {
		t.Fatal("client block chain not excepted!")
	}
	sync.InitBlockSyncer()

	//等待块同步
	mockBlock(1, 1)
	mockBlock(2, 4)
	mockBlock(3, 4)
	mockBlock(4, 4)
	mockBlock(5, 4)
	mockBlock(6, 4)
	mockBlock(7, 4)
	mockBlock(8, 4)
	mockBlock(9, 4)
	mockBlock(10, 4)
	mockBlock(11, 4)
	mockBlock(12, 4)
	printBlockInfo()
	time.Sleep(time.Second * 30)

	log.Printf("client block chain height:%d", core.BlockChainImpl.Height())
	printBlockInfo()
	if core.BlockChainImpl.Height() != 12 {
		t.Fatal("client block chain not excepted after sync1!")
	}
	time.Sleep(20 * time.Second)
}

func mockBlock(height uint64, qn uint64) {
	txpool := core.BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		log.Fatal("fail to get txpool")
	}
	// 交易1
	txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))
	//交易2
	txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))
	block := core.BlockChainImpl.CastingBlock(height, 0, qn, []byte("castor"), []byte("groupId"))
	if nil == block {
		log.Fatal("fail to cast new block")
	}
	core.BlockChainImpl.AddBlockOnChain(block)
}

func printBlockInfo() {
	var i uint64
	for i = 0; i <= core.BlockChainImpl.Height(); i++ {
		bh := core.BlockChainImpl.QueryBlockByHeight(i)
		if bh == nil {
			log.Printf("Block height:%d is nil", i)
		} else {
			log.Printf("Block height:%d,qn is:%d,hash is: %x", bh.Height, bh.QueueNumber, bh.Hash)
		}
	}
}
