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
	"errors"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/ticker"
	time2 "github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/storage/tasdb"
	"github.com/taschain/taschain/taslog"
	"os"
	"sync"
	"time"
)

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC    = "chain"
	wantedTxsSize = txCountPerBlock
)

var (
	ErrBlockExist      = errors.New("block exist")
	ErrPreNotExist     = errors.New("pre block not exist")
	ErrLocalMoreWeight = errors.New("local more weight")
	ErrCommitBlockFail = errors.New("commit block fail")
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

var consensusLogger taslog.Logger

type BlockChainConfig struct {
	dbfile string
	block  string

	blockHeight string

	state string

	bonus string

	tx      string
	receipt string
	//heavy string
	//light string
	//check string
}

type FullBlockChain struct {
	blocks      *tasdb.PrefixedDatabase
	blockHeight *tasdb.PrefixedDatabase
	txdb        *tasdb.PrefixedDatabase
	//checkdb tasdb.Database
	statedb *tasdb.PrefixedDatabase
	batch   tasdb.Batch

	stateCache account.AccountDatabase

	transactionPool TransactionPool

	//已上链的最新块
	latestBlock   *types.BlockHeader
	latestStateDB *account.AccountDB

	topBlocks *lru.Cache

	// 读写锁
	rwLock sync.RWMutex
	//互斥锁
	mu      sync.Mutex
	batchMu sync.Mutex //批量上链锁

	// 是否可以工作
	init bool

	executor *TVMExecutor

	futureBlocks   *lru.Cache
	verifiedBlocks *lru.Cache

	//verifiedBodyCache *lru.Cache

	isAdujsting bool

	consensusHelper types.ConsensusHelper

	bonusManager *BonusManager

	forkProcessor *forkProcessor
	config        *BlockChainConfig
	//castedBlock  *lru.Cache

	ticker *ticker.GlobalTicker //全局定时器
	ts     time2.TimeService
}

func getBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		dbfile: common.GlobalConf.GetString(CONFIG_SEC, "db_blocks", "d_b") + common.GlobalConf.GetString("instance", "index", ""),
		block:  "bh",

		blockHeight: "hi",

		state: "st",

		bonus: "nu",

		tx:      "tx",
		receipt: "rc",
	}
}

func initBlockChain(helper types.ConsensusHelper) error {
	instance := common.GlobalConf.GetString("instance", "index", "")
	Logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, instance)
	chain := &FullBlockChain{
		config:          getBlockChainConfig(),
		latestBlock:     nil,
		init:            true,
		isAdujsting:     false,
		consensusHelper: helper,
		ticker:          ticker.NewGlobalTicker("chain"),
		ts:              time2.TSInstance,
		futureBlocks:    common.MustNewLRUCache(10),
		verifiedBlocks:  common.MustNewLRUCache(10),
		topBlocks:       common.MustNewLRUCache(20),
	}

	types.DefaultPVFunc = helper.VRFProve2Value

	chain.initMessageHandler()

	ds, err := tasdb.NewDataSource(chain.config.dbfile)
	if err != nil {
		Logger.Errorf("new datasource error:%v", err)
		return err
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.txdb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.statedb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	receiptdb, err := ds.NewPrefixDatabase(chain.config.receipt)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.bonusManager = newBonusManager()
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = NewTransactionPool(chain, receiptdb)

	chain.stateCache = account.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain.ticker)
	// 恢复链状态 height,latestBlock

	chain.latestBlock = chain.loadCurrentBlock()
	if nil != chain.latestBlock {
		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(10)
		Logger.Debugf("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		state, err := account.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic("initBlockChain NewAccountDB fail:" + err.Error())
		}
	} else {
		chain.insertGenesisBlock()
	}

	chain.forkProcessor = initForkProcessor(chain)

	BlockChainImpl = chain
	return nil
}

func (chain *FullBlockChain) buildCache(size int) {
	hash := chain.latestBlock.Hash
	for size > 0 {
		b := chain.queryBlockByHash(hash)
		if b != nil {
			chain.addTopBlock(b)
			size--
			hash = b.Header.PreHash
		} else {
			break
		}
	}
}

// 创始块
func (chain *FullBlockChain) insertGenesisBlock() {
	stateDB, err := account.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil != err {
		panic("Init block chain error:" + err.Error())
	}

	block := new(types.Block)
	block.Header = &types.BlockHeader{
		Height:     0,
		ExtraData:  common.Sha256([]byte("tas")),
		CurTime:    time2.TimeToTimeStamp(time.Date(2019, 4, 25, 0, 0, 0, 0, time.UTC)),
		ProveValue: []byte{},
		Elapsed:    0,
		TotalQN:    0,
		//Transactions: make([]common.Hash, 0), //important!!
		Nonce: common.ChainDataVersion,
	}

	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	genesisInfo := chain.consensusHelper.GenerateGenesisInfo()
	setupGenesisStateDB(stateDB, genesisInfo)

	//stage := stateDB.IntermediateRoot(false)
	//Logger.Debugf("GenesisBlock Stage1 Root:%s", stage.Hex())
	miners := make([]*types.Miner, 0)
	for i, member := range genesisInfo.Group.Members {
		miner := &types.Miner{Id: member, PublicKey: genesisInfo.Pks[i], VrfPublicKey: genesisInfo.VrfPKs[i], Stake: common.TAS2RA(100)}
		miners = append(miners, miner)
	}
	MinerManagerImpl.addGenesesMiner(miners, stateDB)
	//stage = stateDB.IntermediateRoot(false)
	//Logger.Debugf("GenesisBlock Stage2 Root:%s", stage.Hex())
	stateDB.SetNonce(common.BonusStorageAddress, 1)
	stateDB.SetNonce(common.HeavyDBAddress, 1)
	stateDB.SetNonce(common.LightDBAddress, 1)

	root, _ := stateDB.Commit(true)
	//Logger.Debugf("GenesisBlock final Root:%s", root.Hex())
	//triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()

	ok, err := chain.commitBlock(block, &executePostState{state: stateDB})
	if !ok {
		panic("insert genesis block fail, err=%v" + err.Error())
	}

	Logger.Debugf("GenesisBlock %+v", block.Header)
}

//清除链所有数据
func (chain *FullBlockChain) Clear() error {
	//gchain.mu.Lock()
	//defer gchain.mu.Unlock()
	//
	//gchain.init = false
	//gchain.latestBlock = nil
	//gchain.topBlocks, _ = lru.New(1000)
	//
	//var err error
	//
	//gchain.blocks.Close()
	//gchain.blockHeight.Close()
	//gchain.statedb.Close()
	//
	//os.RemoveAll(tasdb.DEFAULT_FILE)
	//
	//gchain.statedb, err = ds.NewPrefixDatabase(gchain.config.state)
	//if err != nil {
	//	return err
	//}
	//
	//gchain.stateCache = account.NewDatabase(gchain.statedb)
	//gchain.executor = NewTVMExecutor(gchain)
	//
	//gchain.insertGenesisBlock()
	//gchain.init = true
	//gchain.transactionPool.Clear()
	return nil
}

func Clear() {
	path := tasdb.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", tasdb.DEFAULT_FILE)
	}
	os.RemoveAll(path)
}

func (chain *FullBlockChain) versionValidate() bool {
	genesisHeader := chain.queryBlockHeaderByHeight(uint64(0))
	if genesisHeader == nil {
		return false
	}
	version := genesisHeader.Nonce
	if version != common.ChainDataVersion {
		return false
	}
	return true
}

func (chain *FullBlockChain) compareChainWeight(bh2 *types.BlockHeader) int {
	bh1 := chain.getLatestBlock()
	return chain.compareBlockWeight(bh1, bh2)
}

func (chain *FullBlockChain) compareBlockWeight(bh1 *types.BlockHeader, bh2 *types.BlockHeader) int {
	bw1 := types.NewBlockWeight(bh1)
	bw2 := types.NewBlockWeight(bh2)
	return bw1.Cmp(bw2)
}

func (chain *FullBlockChain) Close() {
	chain.blocks.Close()
	chain.blockHeight.Close()
	chain.statedb.Close()
}

func (chain *FullBlockChain) AddBonusTrasanction(transaction *types.Transaction) {
	chain.GetTransactionPool().AddTransaction(transaction)
}

func (chain *FullBlockChain) GetBonusManager() *BonusManager {
	return chain.bonusManager
}

func (chain *FullBlockChain) GetConsensusHelper() types.ConsensusHelper {
	return chain.consensusHelper
}

func (chain *FullBlockChain) ResetTop(bh *types.BlockHeader) {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	chain.resetTop(bh)
}

func (chain *FullBlockChain) Remove(block *types.Block) bool {
	chain.mu.Lock()
	defer chain.mu.Unlock()
	pre := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if pre == nil {
		return chain.removeOrphan(block) == nil
	} else {
		return chain.resetTop(pre) == nil
	}
}

func (chain *FullBlockChain) getLatestBlock() *types.BlockHeader {
	result := chain.latestBlock
	return result
}

func (chain *FullBlockChain) Version() int {
	return common.ChainDataVersion
}
