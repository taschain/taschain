package core

import (
	"bytes"
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"middleware/types"
	"storage/core/vm"
	"strconv"
	"sync"
	"taslog"
)


var BonusLogger = taslog.GetLogger(taslog.BonusStatConfig)
//var BonusLogger = taslog.GetLoggerByName("BonusStat")
var CastBlockLogger = taslog.GetLogger(taslog.CastBlockStatConfig)
//var CastBlockLogger = taslog.GetLoggerByName("CastBlockStat")
var VerifyGroupLogger = taslog.GetLogger(taslog.VerifyGroupStatConfig)
//var VerifyGroupLogger = taslog.GetLoggerByName("VerifyGroupStat")

type BonusManager struct {
	lock sync.RWMutex
}

func newBonusManager() *BonusManager {
	manager := &BonusManager{}
	return manager
}

func (bm *BonusManager) Contain(blockHash []byte, accountdb vm.AccountDB) bool {
	value := accountdb.GetData(common.BonusStorageAddress,string(blockHash))
	if value != nil{
		return true
	}
	return false
}

func (bm *BonusManager) Put(blockHash []byte, transactionHash []byte, accountdb vm.AccountDB)  {
	accountdb.SetData(common.BonusStorageAddress, string(blockHash), transactionHash)
}

func (bm *BonusManager) WhetherBonusTransaction(transaction *types.Transaction) bool{
	return transaction.Type == types.TransactionTypeBonus
}

func (bm *BonusManager) GetBonusTransactionByBlockHash(blockHash []byte) *types.Transaction{
	transactionHash := BlockChainImpl.(*FullBlockChain).latestStateDB.GetData(common.BonusStorageAddress, string(blockHash))
	if transactionHash == nil{
		return nil
	}
	transaction,_ := BlockChainImpl.(*FullBlockChain).transactionPool.GetTransaction(common.BytesToHash(transactionHash))
	return transaction
}

func (bm *BonusManager) GenerateBonus(targetIds []int32, blockHash common.Hash, groupId []byte, totalValue uint64) (*types.Bonus,*types.Transaction) {
	group := GroupChainImpl.getGroupById(groupId)
	buffer := &bytes.Buffer{}
	buffer.Write(groupId)
	Logger.Debugf("GenerateBonus Group:%s",common.BytesToAddress(groupId).GetHexString())
	for i:=0;i<len(targetIds);i++{
		index := targetIds[i]
		buffer.Write(group.Members[index].Id)
		Logger.Debugf("GenerateBonus Index:%d Member:%s",index,common.BytesToAddress(group.Members[index].Id).GetHexString())
	}
	transaction := &types.Transaction{}
	transaction.Data = blockHash.Bytes()
	transaction.ExtraData = buffer.Bytes()
	transaction.Hash = transaction.GenHash()
	transaction.Value = totalValue / uint64(len(targetIds))
	transaction.Type = types.TransactionTypeBonus
	return &types.Bonus{TxHash:transaction.Hash,TargetIds:targetIds,BlockHash:blockHash,GroupId:groupId,TotalValue:totalValue},transaction
}

func(bm *BonusManager) StatBonusByBlockHeight(blockHeight uint64){
	var i uint64
	for i = 1; i <= blockHeight; i++{
		blockHeader := BlockChainImpl.QueryBlockByHeight(blockHeight)

		// 打印铸块信息
		casterId := blockHeader.Castor
		groupId := blockHeader.GroupId

		group := GroupChainImpl.GetGroupById(groupId)
		for _, member := range group.Members{
			memberId := member.Id
			if miner := MinerManagerImpl.GetMinerById(memberId, types.MinerTypeLight, nil);miner != nil{
				minerStake := miner.Stake
				VerifyGroupLogger.Infof(fmt.Sprintf("%v|%v|%v|%v", i, groupsig.DeserializeId(groupId).GetHexString(), groupsig.DeserializeId(memberId).GetHexString(), minerStake))
			}
		}

		if miner := MinerManagerImpl.GetMinerById(casterId,types.MinerTypeHeavy, nil); miner != nil{
			minerStake := miner.Stake
			totalStake := MinerManagerImpl.GetTotalStakeByHeight(blockHeight)
			CastBlockLogger.Infof(fmt.Sprintf("%v|%v|%v|%v|%v", i, groupsig.DeserializeId(groupId).GetHexString(), groupsig.DeserializeId(casterId).GetHexString(), minerStake, totalStake))
		}

		// 获取验证分红的交易信息
		bonusTx := BlockChainImpl.GetBonusManager().GetBonusTransactionByBlockHash(blockHeader.Hash.Bytes())

		// 从交易信息中解析出targetId列表
		reader := bytes.NewReader(bonusTx.ExtraData)
		groupIdExtra := make([]byte,common.GroupIdLength)
		addr := make([]byte,common.AddressLength)

		// 分配给每一个验证节点的分红交易
		value := big.NewInt(int64(bonusTx.Value))

		if n,_ := reader.Read(groupIdExtra);n != common.GroupIdLength{
			panic("TVMExecutor Read GroupId Fail")
		}

		for n,_ := reader.Read(addr);n > 0;n,_ = reader.Read(addr){
			address := common.BytesToAddress(addr)
			balance := BlockChainImpl.GetBalance(address)

			// 打印日志
			BonusLogger.Infof(genMinerBonusLogInfo(i, blockHeader.Hash, bonusTx.Hash, groupId, casterId, address, balance, value))
		}
	}
}

func genMinerBonusLogInfo(blockHeight uint64, bh common.Hash, th common.Hash, groupId []byte, casterId []byte, address common.Address, balance *big.Int, bonus *big.Int) string{
	buffer := &bytes.Buffer{}
	buffer.WriteString(strconv.Itoa(int(blockHeight)))
	buffer.WriteString("|")

	buffer.WriteString(bh.String())
	buffer.WriteString("|")

	buffer.WriteString(th.String())
	buffer.WriteString("|")

	buffer.WriteString(groupsig.DeserializeId(groupId).GetHexString())
	buffer.WriteString("|")

	buffer.WriteString(groupsig.DeserializeId(casterId).GetHexString())
	buffer.WriteString("|")

	buffer.WriteString(address.GetHexString())
	buffer.WriteString("|")

	buffer.WriteString(balance.String())
	buffer.WriteString("|")

	buffer.WriteString(bonus.String())

	return buffer.String()
}