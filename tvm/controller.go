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

package tvm

import (
	"encoding/json"
	"math/big"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/vm"
)

var HasLoadPyLibPath = false

type ControllerTransactionInterface interface {
	GetGasLimit() uint64
	GetValue() uint64
	GetSource() *common.Address
	GetTarget() *common.Address
	GetData() []byte
	GetHash() common.Hash
}

type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction ControllerTransactionInterface
	AccountDB   vm.AccountDB
	Reader      vm.ChainReader
	VM          *Tvm
	LibPath     string
	VMStack     []*Tvm
	GasLeft     uint64
	mm          MinerManager
	gcm         GroupChainManager
}

type MinerManager interface {
	GetMinerByID(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner
	GetLatestCancelStakeHeight(from []byte, miner *types.Miner, accountdb vm.AccountDB) uint64
	RefundStake(from []byte, miner *types.Miner, accountdb vm.AccountDB) (uint64, bool)
	CancelStake(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool
	ReduceStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool
	AddStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool
	AddStakeDetail(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool
}

type GroupChainManager interface {
	WhetherMemberInActiveGroup(id []byte, currentHeight uint64) bool
}

func NewController(accountDB vm.AccountDB,
	chainReader vm.ChainReader,
	header *types.BlockHeader,
	transaction ControllerTransactionInterface,
	gasUsed uint64,
	libPath string,
	manager MinerManager, chainManager GroupChainManager) *Controller {
	if controller == nil {
		controller = &Controller{}
	}
	controller.BlockHeader = header
	controller.Transaction = transaction
	controller.AccountDB = accountDB
	controller.Reader = chainReader
	controller.VM = nil
	controller.LibPath = libPath
	controller.VMStack = make([]*Tvm, 0)
	controller.GasLeft = transaction.GetGasLimit() - gasUsed
	controller.mm = manager
	controller.gcm = chainManager
	return controller
}

func (con *Controller) Deploy(contract *Contract) (int, string) {
	con.VM = NewTvm(con.Transaction.GetSource(), contract, con.LibPath)
	defer func() {
		con.VM.DelTvm()
	}()
	con.VM.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().Hex()}
	errorCodeDeploy, errorDeployMsg := con.VM.Deploy(msg)

	if errorCodeDeploy != 0 {
		return errorCodeDeploy, errorDeployMsg
	}
	errorCodeStore, errorStoreMsg := con.VM.StoreData()
	if errorCodeStore != 0 {
		return errorCodeStore, errorStoreMsg
	}
	con.GasLeft = uint64(con.VM.Gas())
	return 0, ""
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func (con *Controller) ExecuteAbi(sender *common.Address, contract *Contract, abiJSON string) (bool, []*types.Log, *types.TransactionError) {
	con.VM = NewTvm(sender, contract, con.LibPath)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTvm()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	//先转账
	if con.Transaction.GetValue() > 0 {
		amount := new(big.Int).SetUint64(con.Transaction.GetValue())
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.GetTarget(), amount)
		} else {
			return false, nil, types.TxErrorBalanceNotEnough
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().Hex()}
	errorCode, errorMsg, libLen := con.VM.CreateContractInstance(msg)
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		return false, nil, types.TxErrorABIJSON
	}
	errorCode, errorMsg = con.VM.checkABI(abi) //checkABI
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	con.VM.SetLibLine(libLen)
	errorCode, errorMsg = con.VM.ExecutedAbiVMSucceed(abi) //execute
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	errorCode, errorMsg = con.VM.StoreData() //store
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	return true, con.VM.Logs, nil
}

func (con *Controller) ExecuteAbiEval(sender *common.Address, contract *Contract, abiJSON string) *ExecuteResult {
	con.VM = NewTvm(sender, contract, con.LibPath)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTvm()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	//先转账
	if con.Transaction.GetValue() > 0 {
		amount := big.NewInt(int64(con.Transaction.GetValue()))
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.GetTarget(), amount)
		} else {
			return nil
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: sender.Hex()}
	errorCode, _, libLen := con.VM.CreateContractInstance(msg)
	if errorCode != 0 {
		return nil
	}
	abi := ABI{}
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		return nil
	}
	errorCode, _ = con.VM.checkABI(abi) //checkABI
	if errorCode != 0 {
		return nil
	}
	con.VM.SetLibLine(libLen)
	result := con.VM.ExecuteABIKindEval(abi) //execute
	if result.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		return result
	}
	errorCode, _ = con.VM.StoreData() //store
	if errorCode != 0 {
		return nil
	}
	return result
}

func (con *Controller) GetGasLeft() uint64 {
	return con.GasLeft
}
