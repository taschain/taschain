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
	Vm          *Tvm
	LibPath     string
	VmStack     []*Tvm
	GasLeft     uint64
	mm          MinerManager
	gcm 		GroupChainManager
}

type MinerManager interface {
	GetMinerById(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner
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
	controller.Vm = nil
	controller.LibPath = libPath
	controller.VmStack = make([]*Tvm, 0)
	controller.GasLeft = transaction.GetGasLimit() - gasUsed
	controller.mm = manager
	controller.gcm = chainManager
	return controller
}

func (con *Controller) Deploy(contract *Contract) (int,string) {
	con.Vm = NewTvm(con.Transaction.GetSource(), contract, con.LibPath)
	defer func() {
		con.Vm.DelTvm()
	}()
	con.Vm.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().GetHexString()}
	errorCodeDeploy,errorDeployMsg:= con.Vm.Deploy(msg)

	if errorCodeDeploy != 0 {
		return errorCodeDeploy,errorDeployMsg
	}
	errorCodeStore,errorStoreMsg := con.Vm.StoreData()
	if errorCodeStore != 0 {
		return errorCodeStore,errorStoreMsg
	}
	con.GasLeft = uint64(con.Vm.Gas())
	return 0,""
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func (con *Controller) ExecuteAbi(sender *common.Address, contract *Contract, abiJson string) (bool,[]*types.Log,*types.TransactionError) {
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.GasLeft))
	defer func() {
		con.Vm.DelTvm()
		con.GasLeft = uint64(con.Vm.Gas())
	}()
	//先转账
	if con.Transaction.GetValue() > 0 {
		amount := new(big.Int).SetUint64(con.Transaction.GetValue())
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.GetTarget(), amount)
		} else {
			return false,nil,types.TxErrorBalanceNotEnough
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().GetHexString()}
	errorCode,errorMsg,libLen := con.Vm.CreateContractInstance(msg)
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	abi := ABI{}
	abiJsonError := json.Unmarshal([]byte(abiJson), &abi)
	if abiJsonError!= nil{
		return false,nil,types.TxErrorAbiJson
	}
	errorCode,errorMsg = con.Vm.checkABI(abi)//checkABI
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	con.Vm.SetLibLine(libLen)
	errorCode,errorMsg = con.Vm.ExecutedAbiVmSucceed(abi)//execute
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	errorCode,errorMsg = con.Vm.StoreData()//store
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	return true,con.Vm.Logs,nil
}

func (con *Controller) ExecuteAbiEval(sender *common.Address, contract *Contract, abiJson string) *ExecuteResult {
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.GasLeft))
	defer func() {
		con.Vm.DelTvm()
		con.GasLeft = uint64(con.Vm.Gas())
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
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: sender.GetHexString()}
	errorCode,_,libLen := con.Vm.CreateContractInstance(msg)
	if errorCode != 0{
		return nil
	}
	abi := ABI{}
	abiJsonError := json.Unmarshal([]byte(abiJson), &abi)
	if abiJsonError!= nil{
		return nil
	}
	errorCode,_ = con.Vm.checkABI(abi)//checkABI
	if errorCode != 0{
		return nil
	}
	con.Vm.SetLibLine(libLen)
	result := con.Vm.ExecuteABIKindEval(abi)//execute
	if result.ResultType == 4/*C.RETURN_TYPE_EXCEPTION*/ {
		return result
	}
	errorCode,_ = con.Vm.StoreData()//store
	if errorCode != 0{
		return nil
	}
	return result
}

func(con *Controller) GetGasLeft() uint64{
	return con.GasLeft
}
