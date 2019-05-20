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
	"bytes"
	"fmt"
	"time"
	"math/big"

	"common"
	"tvm"
	"middleware/types"
	"storage/account"
	"storage/vm"
)

const TransactionGasCost uint64 = 1000
const CodeBytePrice = 0.3814697265625
const MaxCastBlockTime = time.Second * 3

type TVMExecutor struct {
	bc BlockChain
}

func NewTVMExecutor(bc BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc: bc,
	}
}

func (executor *TVMExecutor) Execute(accountdb *account.AccountDB, bh *types.BlockHeader, txs []*types.Transaction, pack bool) (state common.Hash, evits []common.Hash, executed []*types.Transaction, recps []*types.Receipt, err error) {
	beginTime := time.Now()
	receipts := make([]*types.Receipt, 0)
	transactions := make([]*types.Transaction, 0)
	evictedTxs := make([]common.Hash, 0)
	castor := common.BytesToAddress(bh.Castor)

	//errs := make([]*types.TransactionError, len(block.Transactions))

	for _, transaction := range txs {
		if pack && time.Since(beginTime).Seconds() > float64(MaxCastBlockTime) {
			Logger.Infof("Cast block execute tx time out!Tx hash:%s ", transaction.Hash.String())
			break
		}
		//Logger.Debugf("TVMExecutor Execute %v,type:%d", transaction.Hash, transaction.Type)
		var success = false
		var contractAddress common.Address
		var logs []*types.Log
		//var err *types.TransactionError
		var cumulativeGasUsed uint64

		if !executor.validateNonce(accountdb, transaction) {
			evictedTxs = append(evictedTxs, transaction.Hash)
			continue
		}

		switch transaction.Type {
		case types.TransactionTypeTransfer:
			success, _, cumulativeGasUsed = executor.executeTransferTx(accountdb, transaction, castor)
		case types.TransactionTypeContractCreate:
			success, _, cumulativeGasUsed, contractAddress = executor.executeContractCreateTx(accountdb, transaction, castor, bh)
		case types.TransactionTypeContractCall:
			success, _, cumulativeGasUsed, logs = executor.executeContractCallTx(accountdb, transaction, castor, bh)
		case types.TransactionTypeBonus:
			success = executor.executeBonusTx(accountdb, transaction, castor)
			if !success {
				evictedTxs = append(evictedTxs, transaction.Hash)
				//failed bonus tx should not be included in block
				continue
			}
		case types.TransactionTypeMinerApply:
			success = executor.executeMinerApplyTx(accountdb, transaction, bh.Height, castor)
		case types.TransactionTypeMinerAbort:
			success = executor.executeMinerAbortTx(accountdb, transaction, bh.Height, castor)
		case types.TransactionTypeMinerRefund:
			success = executor.executeMinerRefundTx(accountdb, transaction, bh.Height, castor)
		}

		idx := len(transactions)
		transactions = append(transactions, transaction)
		receipt := types.NewReceipt(nil, !success, cumulativeGasUsed)
		receipt.Logs = logs
		receipt.TxHash = transaction.Hash
		receipt.ContractAddress = contractAddress
		receipt.TxIndex = uint16(idx)
		receipt.Height = bh.Height
		receipts = append(receipts, receipt)
		//errs[i] = err
		if transaction.Source != nil {
			accountdb.SetNonce(*transaction.Source, transaction.Nonce)
		}

	}
	accountdb.AddBalance(castor, executor.bc.GetConsensusHelper().ProposalBonus())

	state = accountdb.IntermediateRoot(true)
	return state, evictedTxs, transactions, receipts, nil
}

func (executor *TVMExecutor) validateNonce(accountdb *account.AccountDB, transaction *types.Transaction) bool {
	if transaction.Type == types.TransactionTypeBonus || IsTestTransaction(transaction) {
		return true
	}
	nonce := accountdb.GetNonce(*transaction.Source)

	//hardcode for performance test, alway return true
	var performanceTest = true;
	if performanceTest {
		return true
	}

	if transaction.Nonce != nonce+1 {
		Logger.Infof("Tx nonce error! Hash:%s,Source:%s,expect nonce:%d,real nonce:%d ", transaction.Hash.String(), transaction.Source.GetHexString(), nonce+1, transaction.Nonce)
		return false
	}
	return true
}

func (executor *TVMExecutor) executeTransferTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (success bool, err *types.TransactionError, cumulativeGasUsed uint64) {
	success = false

	amount := new(big.Int).SetUint64(transaction.Value)
	intriGas,err := intrinsicGas(transaction)
	if err != nil{
		return
	}
	gasFee := new(big.Int).SetUint64(transaction.GasPrice * intriGas)
	if canTransfer(accountdb, *transaction.Source, amount, gasFee) {
		transfer(accountdb, *transaction.Source, *transaction.Target, amount)
		accountdb.SubBalance(*transaction.Source, gasFee)
		accountdb.AddBalance(castor, gasFee)
		cumulativeGasUsed = gasFee.Uint64()
		success = true
	} else {
		err = types.TxErrorBalanceNotEnough
	}
	//Logger.Debugf("TVMExecutor Execute Transfer Source:%s Target:%s Value:%d Height:%d Type:%s,Gas:%d,Success:%t", transaction.Source.GetHexString(), transaction.Target.GetHexString(), transaction.Value, height, mark,cumulativeGasUsed,success)
	return success, err, cumulativeGasUsed
}

func (executor *TVMExecutor) executeContractCreateTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, contractAddress common.Address) {
	success = false
	intriGas,err := intrinsicGas(transaction)
	if err != nil{
		return
	}
	gasLimit := transaction.GasLimit
	gasLimitFee := new(big.Int).SetUint64(transaction.GasLimit * transaction.GasPrice)

	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), gasLimitFee) {
		accountdb.SubBalance(*transaction.Source, gasLimitFee)
		controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, intriGas, common.GlobalConf.GetString("tvm", "pylib", "lib"))
		snapshot := controller.AccountDB.Snapshot()
		contractAddress, err = createContract(accountdb, transaction)
		if err != nil {
			Logger.Debugf("ContractCreate tx %s execute error:%s ", transaction.Hash.String(), err.Message)
			controller.AccountDB.RevertToSnapshot(snapshot)
		} else {
			contract := tvm.LoadContract(contractAddress)
			errorCode, errorMsg := controller.Deploy(transaction.Source, contract)
			if errorCode != 0 {
				err = types.NewTransactionError(errorCode, errorMsg)
				controller.AccountDB.RevertToSnapshot(snapshot)
			} else {
				success = true
				Logger.Debugf("Contract create success! Tx hash:%s, contract addr:%s", transaction.Hash.String(), contractAddress.String())
			}
		}
		gasLeft := controller.GetGasLeft()
		returnFee := new(big.Int).SetUint64(gasLeft * transaction.GasPrice)
		accountdb.AddBalance(*transaction.Source, returnFee)
		accountdb.AddBalance(castor, new(big.Int).Sub(gasLimitFee, returnFee))

		cumulativeGasUsed = gasLimit - gasLeft
	} else {
		success = false
		err = types.TxErrorBalanceNotEnough
		Logger.Infof("ContractCreate balance not enough! transaction %s source %s  ", transaction.Hash.String(), transaction.Source.String())
	}
	//Logger.Debugf("TVMExecutor Execute ContractCreate Transaction %s,success:%t", transaction.Hash.Hex(),success)
	return success, err, cumulativeGasUsed, contractAddress
}

func (executor *TVMExecutor) executeContractCallTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address, bh *types.BlockHeader) (success bool, err *types.TransactionError, cumulativeGasUsed uint64, logs []*types.Log) {
	success = false
	transferAmount := new(big.Int).SetUint64(transaction.Value)
	intriGas, err := intrinsicGas(transaction)
	if err != nil{
		return
	}
	gasLimit := transaction.GasLimit
	gasLimitFee := new(big.Int).SetUint64(transaction.GasLimit * transaction.GasPrice)

	if canTransfer(accountdb, *transaction.Source, transferAmount, gasLimitFee) {
		accountdb.SubBalance(*transaction.Source, gasLimitFee)
		controller := tvm.NewController(accountdb, BlockChainImpl, bh, transaction, intriGas, common.GlobalConf.GetString("tvm", "pylib", "lib"))
		contract := tvm.LoadContract(*transaction.Target)
		if contract.Code == "" {
			err = types.NewTransactionError(types.TxErrorCode_NO_CODE, fmt.Sprintf(types.NO_CODE_ERROR_MSG, *transaction.Target))

		} else {
			snapshot := controller.AccountDB.Snapshot()
			success, logs, err = controller.ExecuteAbi(transaction.Source, contract, string(transaction.Data))
			if !success {
				controller.AccountDB.RevertToSnapshot(snapshot)
			} else {
				success = true
				transfer(accountdb,*transaction.Source,*contract.ContractAddress,transferAmount)
			}
		}
		gasLeft := controller.GetGasLeft()
		returnFee := new(big.Int).SetUint64(gasLeft * transaction.GasPrice)
		accountdb.AddBalance(*transaction.Source, returnFee)
		accountdb.AddBalance(castor, new(big.Int).Sub(gasLimitFee, returnFee))

		cumulativeGasUsed = gasLimit - gasLeft
	} else {
		err = types.TxErrorBalanceNotEnough
	}
	Logger.Debugf("TVMExecutor Execute ContractCall Transaction %s,success:%t", transaction.Hash.Hex(), success)
	return success, err, cumulativeGasUsed, logs
}

func (executor *TVMExecutor) executeBonusTx(accountdb *account.AccountDB, transaction *types.Transaction, castor common.Address) (success bool) {
	success = false
	if executor.bc.GetBonusManager().contain(transaction.Data, accountdb) == false {
		reader := bytes.NewReader(transaction.ExtraData)
		groupId := make([]byte, common.GroupIdLength)
		addr := make([]byte, common.AddressLength)
		value := new(big.Int).SetUint64(transaction.Value)
		if n, _ := reader.Read(groupId); n != common.GroupIdLength {
			Logger.Errorf("TVMExecutor Read GroupId Fail")
			return success
		}
		for n, _ := reader.Read(addr); n > 0; n, _ = reader.Read(addr) {
			if n != common.AddressLength {
				Logger.Errorf("TVMExecutor Bonus Addr Size:%d Invalid", n)
				break
			}
			address := common.BytesToAddress(addr)
			accountdb.AddBalance(address, value)
		}
		executor.bc.GetBonusManager().put(transaction.Data, transaction.Hash[:], accountdb)
		accountdb.AddBalance(castor, executor.bc.GetConsensusHelper().PackBonus())
		success = true
	}
	//Logger.Debugf("TVMExecutor Execute Bonus Transaction:%s Group:%s,Success:%t", common.BytesToHash(transaction.Data).Hex(), common.BytesToHash(groupId).ShortS(),success)
	return success
}

func (executor *TVMExecutor) executeMinerApplyTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address) (success bool) {
	Logger.Debugf("Execute miner apply tx:%s,source: %v\n", transaction.Hash.String(), transaction.Source.GetHexString())
	success = false
	if transaction.Data == nil {
		Logger.Debugf("TVMExecutor Execute MinerApply Fail(Tx data is nil) Source:%s Height:%d", transaction.Source.GetHexString(), height)
		return success
	}

	intriGas, err := intrinsicGas(transaction)
	if err != nil{
		return
	}
	txExecuteFee := new(big.Int).SetUint64(transaction.GasPrice * intriGas)


	var miner = MinerManagerImpl.Transaction2Miner(transaction)
	mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], miner.Type, accountdb)
	if mexist != nil {
		Logger.Debugf("TVMExecutor Execute MinerApply Fail(Already Exist) Source %s", transaction.Source.GetHexString())
		return success
	}
	amount := new(big.Int).SetUint64(miner.Stake)

	if canTransfer(accountdb, *transaction.Source, amount, txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)

		miner.ApplyHeight = height
		if MinerManagerImpl.addMiner(transaction.Source[:], miner, accountdb) > 0 {
			accountdb.SubBalance(*transaction.Source, amount)
			Logger.Debugf("TVMExecutor Execute MinerApply Success Source:%s Height:%d", transaction.Source.GetHexString(), height)
		}
		success = true
	} else {
		Logger.Debugf("TVMExecutor Execute MinerApply Fail(Balance Not Enough) Source:%s Height:%d", transaction.Source.GetHexString(), height)
	}
	return success
}

func (executor *TVMExecutor) executeMinerAbortTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address) (success bool) {
	success = false

	intriGas, err := intrinsicGas(transaction)
	if err != nil{
		return
	}
	txExecuteFee := new(big.Int).SetUint64(transaction.GasPrice * intriGas)
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)
		if transaction.Data != nil {
			success = MinerManagerImpl.abortMiner(transaction.Source[:], transaction.Data[0], height, accountdb)
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerAbort Fail(Balance Not Enough) Source:%s Height:%d ", transaction.Source.GetHexString(), height)
	}
	Logger.Debugf("TVMExecutor Execute MinerAbort Tx %s,Source:%s, Success:%t", transaction.Hash.String(), transaction.Source.GetHexString(), success)
	return success
}

func (executor *TVMExecutor) executeMinerRefundTx(accountdb *account.AccountDB, transaction *types.Transaction, height uint64, castor common.Address) (success bool) {
	success = false
	intriGas, err := intrinsicGas(transaction)
	if err != nil{
		return
	}

	txExecuteFee := new(big.Int).SetUint64(transaction.GasPrice * intriGas)
	if canTransfer(accountdb, *transaction.Source, new(big.Int).SetUint64(0), txExecuteFee) {
		accountdb.SubBalance(*transaction.Source, txExecuteFee)
		accountdb.AddBalance(castor, txExecuteFee)
	} else {
		Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Balance Not Enough) Hash:%s,Source:%s", transaction.Hash.String(), transaction.Source.GetHexString())
		return success
	}

	mexist := MinerManagerImpl.GetMinerById(transaction.Source[:], transaction.Data[0], accountdb)
	if mexist != nil && mexist.Status == types.MinerStatusAbort {
		if mexist.Type == types.MinerTypeHeavy {
			if height > mexist.AbortHeight+10 {
				MinerManagerImpl.removeMiner(transaction.Source[:], mexist.Type, accountdb)
				amount := new(big.Int).SetUint64(mexist.Stake)
				accountdb.AddBalance(*transaction.Source, amount)
				Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Success %s", transaction.Source.GetHexString())
				success = true
			} else {
				Logger.Debugf("TVMExecutor Execute MinerRefund Heavy Fail(Refund height less than abortHeight+10) Hash%s", transaction.Source.GetHexString())
			}
		} else {
			if !GroupChainImpl.WhetherMemberInActiveGroup(transaction.Source[:], height) {
				MinerManagerImpl.removeMiner(transaction.Source[:], mexist.Type, accountdb)
				amount := new(big.Int).SetUint64(mexist.Stake)
				accountdb.AddBalance(*transaction.Source, amount)
				Logger.Debugf("TVMExecutor Execute MinerRefund Light Success %s,Type:%s", transaction.Source.GetHexString())
				success = true
			} else {
				Logger.Debugf("TVMExecutor Execute MinerRefund Light Fail(Still In Active Group) %s", transaction.Source.GetHexString())
			}
		}
	} else {
		Logger.Debugf("TVMExecutor Execute MinerRefund Fail(Not Exist Or Not Abort) %s", transaction.Source.GetHexString())
	}
	return success
}

func createContract(accountdb *account.AccountDB, transaction *types.Transaction) (common.Address, *types.TransactionError) {
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(transaction.Nonce))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}) {
		return common.Address{}, types.NewTransactionError(types.TxErrorCode_ContractAddressConflict, "contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)
	return contractAddr, nil
}


/**
交易固定消耗gas
*/
func intrinsicGas( transaction *types.Transaction) (gas uint64, err *types.TransactionError) {
	gas = uint64(float32(len(transaction.Data)+len(transaction.ExtraData)) * CodeBytePrice)
	gas = TransactionGasCost + gas
	if transaction.GasLimit < gas {
		return 0, types.TxErrorDeployGasNotEnough
	}
	return gas,nil
}

func canTransfer(db vm.AccountDB, addr common.Address, amount *big.Int, gasFee *big.Int) bool {
	totalAmount := new(big.Int).Add(amount, gasFee)
	return db.GetBalance(addr).Cmp(totalAmount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		//escape if amount is zero
		return
	}
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
