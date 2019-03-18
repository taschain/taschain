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
	"common"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"taslog"
	"math/big"
	"os"
	"storage/account"
	"storage/tasdb"
	"time"
	"middleware/notify"
	"github.com/pkg/errors"
	"middleware/ticker"
	"sync"
)

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC = "chain"

	addBlockMark = "addBlockMark"

	removeBlockMark = "removeBlockMark"
)

var (
	ErrBlockExist = errors.New("block exist")
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

var consensusLogger taslog.Logger

type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	bonus string

	tx string
	//heavy string
	//light string
	//check string
}

type FullBlockChain struct {
	isLightMiner bool
	blocks       *tasdb.PrefixedDatabase
	blockHeight  *tasdb.PrefixedDatabase
	//checkdb tasdb.Database
	statedb    *tasdb.PrefixedDatabase
	batch 		tasdb.Batch

	stateCache account.AccountDatabase

	transactionPool TransactionPool

	//已上链的最新块
	latestBlock   *types.BlockHeader
	latestStateDB *account.AccountDB

	topBlocks *lru.Cache

	// 读写锁
	rwLock sync.RWMutex
	//互斥锁
	mu 		sync.Mutex

	// 是否可以工作
	init bool

	executor *TVMExecutor

	futureBlocks   *lru.Cache
	verifiedBlocks *lru.Cache

	verifiedBodyCache *lru.Cache

	isAdujsting bool

	consensusHelper types.ConsensusHelper

	bonusManager *BonusManager

	forkProcessor *forkProcessor
	config       *BlockChainConfig
	castedBlock  *lru.Cache

	ticker 		*ticker.GlobalTicker	//全局定时器
}


type castingBlock struct {
	state    *account.AccountDB
	receipts types.Receipts
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := &BlockChainConfig{
		block: "bha",

		blockHeight: "bh",

		state: "st",

		bonus: "bnus",

		tx: "tx",

		//light: "light",
		//
		//heavy: "heavy",
		//
		//check: "check",
	}

	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &BlockChainConfig{
		block: common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),

		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),

		bonus: common.GlobalConf.GetString(CONFIG_SEC, "bonus", defaultConfig.bonus),

		//heavy: common.GlobalConf.GetString(CONFIG_SEC, "heavy", defaultConfig.heavy),

		//light: common.GlobalConf.GetString(CONFIG_SEC, "light", defaultConfig.light),

		//check: common.GlobalConf.GetString(CONFIG_SEC, "check", defaultConfig.check),
	}

}

func initBlockChain(helper types.ConsensusHelper) error {
	Logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	chain := &FullBlockChain{
		config: getBlockChainConfig(),
		latestBlock:     nil,
		init:            true,
		isAdujsting:     false,
		isLightMiner:    false,
		consensusHelper: helper,
		ticker: 		ticker.NewGlobalTicker("chain"),
	}

	notify.BUS.Subscribe(notify.BlockAddSucc, chain.onBlockAddSuccess)

	var err error
	chain.futureBlocks, err = lru.New(10)
	if err != nil {
		return err
	}
	chain.verifiedBlocks, err = lru.New(10)
	if err != nil {
		return err
	}
	chain.topBlocks, _ = lru.New(20)
	if err != nil {
		return err
	}
	chain.castedBlock, err = lru.New(10)
	if err != nil {
		return err
	}

	chain.verifiedBodyCache, _ = lru.New(50)

	chain.blocks, err = tasdb.NewPrefixDatabase(chain.config.block)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.blockHeight, err = tasdb.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.statedb, err = tasdb.NewPrefixDatabase(chain.config.state)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = NewTransactionPool(chain.batch)

	chain.bonusManager = newBonusManager()
	chain.stateCache = account.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager()
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

	chain.forkProcessor = initForkProcessor()
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
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("tas")),
		CurTime:      time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hash, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0), //important!!
		Nonce:        ChainDataVersion,
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

	ok, err := chain.commitBlock(block, stateDB, nil)
	if !ok {
		panic("insert genesis block fail, err=%v" + err.Error())
	}

	Logger.Debugf("GenesisBlock %+v", block.Header)
}

//清除链所有数据
func (chain *FullBlockChain) Clear() error {
	chain.mu.Lock()
	defer chain.mu.Unlock()

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

	var err error

	chain.blocks.Close()
	chain.blockHeight.Close()
	chain.statedb.Close()

	os.RemoveAll(tasdb.DEFAULT_FILE)

	chain.statedb, err = tasdb.NewPrefixDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.stateCache = account.NewDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	chain.insertGenesisBlock()
	chain.init = true
	chain.transactionPool.Clear()
	return err
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
	if version != ChainDataVersion {
		return false
	}
	return true
}

func (chain *FullBlockChain) compareChainWeight(bh2 *types.BlockHeader) int {
	bh1 := chain.getLatestBlock()
	return chain.compareBlockWeight(bh1, bh2)
}

func (chain *FullBlockChain) compareBlockWeight(bh1 *types.BlockHeader, bh2 *types.BlockHeader) int {
	if bh1.TotalQN > bh2.TotalQN {
		return 1
	} else if bh1.TotalQN == bh2.TotalQN {
		v1 := chain.consensusHelper.VRFProve2Value(bh1.ProveValue)
		v2 := chain.consensusHelper.VRFProve2Value(bh2.ProveValue)
		ret := v1.Cmp(v2)
		if ret == 0 && bh1.Hash != bh2.Hash {
			panic("different block hash same prove value")
		}
		return ret
	} else {
		return -1
	}
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


func (chain *FullBlockChain) ResetTop(bh *types.BlockHeader)  {
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