package tvm

import (
	"encoding/json"
	"math/big"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/vm"
)

// HasLoadPyLibPath HasLoadPyLibPath is the flag that whether load python lib
var HasLoadPyLibPath = false

// ControllerTransactionInterface ControllerTransactionInterface is the interface that match Controller
type ControllerTransactionInterface interface {
	GetGasLimit() uint64
	GetValue() uint64
	GetSource() *common.Address
	GetTarget() *common.Address
	GetData() []byte
	GetHash() common.Hash
}

// Controller VM Controller
type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction ControllerTransactionInterface
	AccountDB   vm.AccountDB
	Reader      vm.ChainReader
	VM          *TVM
	LibPath     string
	VMStack     []*TVM
	GasLeft     uint64
	mm          MinerManager
	gcm         GroupChainManager
}

// MinerManager MinerManager is the interface of the miner manager
type MinerManager interface {
	GetMinerByID(id []byte, ttype byte, accountdb vm.AccountDB) *types.Miner
	GetLatestCancelStakeHeight(from []byte, miner *types.Miner, accountdb vm.AccountDB) uint64
	RefundStake(from []byte, miner *types.Miner, accountdb vm.AccountDB) (uint64, bool)
	CancelStake(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool
	ReduceStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB, height uint64) bool
	AddStake(id []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool
	AddStakeDetail(from []byte, miner *types.Miner, amount uint64, accountdb vm.AccountDB) bool
}

// GroupChainManager GroupChainManager is the interface of the GroupChain manager
type GroupChainManager interface {
	WhetherMemberInActiveGroup(id []byte, currentHeight uint64) bool
}

// NewController New a TVM controller
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
	controller.VMStack = make([]*TVM, 0)
	controller.GasLeft = transaction.GetGasLimit() - gasUsed
	controller.mm = manager
	controller.gcm = chainManager
	return controller
}

// Deploy Deploy a contract instance
func (con *Controller) Deploy(contract *Contract) (int, string) {
	con.VM = NewTVM(con.Transaction.GetSource(), contract, con.LibPath)
	defer func() {
		con.VM.DelTVM()
	}()
	con.VM.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().GetHexString()}
	errorCodeDeploy, errorDeployMsg := con.VM.Deploy(msg)

	if errorCodeDeploy != 0 {
		return errorCodeDeploy, errorDeployMsg
	}
	errorCodeStore, errorStoreMsg := con.VM.storeData()
	if errorCodeStore != 0 {
		return errorCodeStore, errorStoreMsg
	}
	con.GasLeft = uint64(con.VM.Gas())
	return 0, ""
}

func canTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// ExecuteABI Execute the contract with abi
func (con *Controller) ExecuteABI(sender *common.Address, contract *Contract, abiJSON string) (bool, []*types.Log, *types.TransactionError) {
	con.VM = NewTVM(sender, contract, con.LibPath)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	if con.Transaction.GetValue() > 0 {
		amount := new(big.Int).SetUint64(con.Transaction.GetValue())
		if canTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.GetTarget(), amount)
		} else {
			return false, nil, types.TxErrorBalanceNotEnough
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: con.Transaction.GetSource().GetHexString()}
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
	errorCode, errorMsg = con.VM.executABIVMSucceed(abi) //execute
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	errorCode, errorMsg = con.VM.storeData() //store
	if errorCode != 0 {
		return false, nil, types.NewTransactionError(errorCode, errorMsg)
	}
	return true, con.VM.Logs, nil
}

// ExecuteAbiEval Execute the contract with abi and returns result
func (con *Controller) ExecuteAbiEval(sender *common.Address, contract *Contract, abiJSON string) *ExecuteResult {
	con.VM = NewTVM(sender, contract, con.LibPath)
	con.VM.SetGas(int(con.GasLeft))
	defer func() {
		con.VM.DelTVM()
		con.GasLeft = uint64(con.VM.Gas())
	}()
	if con.Transaction.GetValue() > 0 {
		amount := big.NewInt(int64(con.Transaction.GetValue()))
		if canTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.GetTarget(), amount)
		} else {
			return nil
		}
	}
	msg := Msg{Data: con.Transaction.GetData(), Value: con.Transaction.GetValue(), Sender: sender.GetHexString()}
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
	result := con.VM.executeABIKindEval(abi) //execute
	if result.ResultType == 4 /*C.RETURN_TYPE_EXCEPTION*/ {
		return result
	}
	errorCode, _ = con.VM.storeData() //store
	if errorCode != 0 {
		return nil
	}
	return result
}

// GetGasLeft get gas left
func (con *Controller) GetGasLeft() uint64 {
	return con.GasLeft
}
