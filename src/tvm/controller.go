package tvm

import (
	"common"
	"encoding/json"
	"fmt"
	"math/big"
	"middleware/types"
	"storage/vm"
)
var HasLoadPyLibPath = false
type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction types.Transaction
	AccountDB   vm.AccountDB
	Reader      vm.ChainReader
	Vm          *Tvm
	LibPath     string
	VmStack     []*Tvm
	GasLeft     uint64
}

func NewController(accountDB vm.AccountDB,
	chainReader vm.ChainReader,
	header *types.BlockHeader,
	transaction *types.Transaction, libPath string) *Controller {
	if controller == nil {
		controller = &Controller{}
	}
	controller.BlockHeader = header
	controller.Transaction = *transaction
	controller.AccountDB = accountDB
	controller.Reader = chainReader
	controller.Vm = nil
	controller.LibPath = libPath
	controller.VmStack = make([]*Tvm, 0)
	controller.GasLeft = transaction.GasLimit
	return controller
}

func (con *Controller) Deploy(sender *common.Address, contract *Contract) (int,string) {
	con.Vm = NewTvm(sender, contract, con.LibPath)
	defer func() {
		con.Vm.DelTvm()
	}()
	con.Vm.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
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
	if con.Transaction.Value > 0 {
		amount := big.NewInt(int64(con.Transaction.Value))
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.Target, amount)
		} else {
			return false,nil,types.TxErrorBalanceNotEnough
		}
	}
	msg := Msg{Data: con.Transaction.Data, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	errorCode,errorMsg,libLen := con.Vm.CreateContractInstance(msg)
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	abi := ABI{}
	abiJsonError := json.Unmarshal([]byte(abiJson), &abi)
	if abiJsonError!= nil{
		return false,nil,types.TxErrorAbiJson
	}
	fmt.Println(abi)
	errorCode,errorMsg = con.Vm.checkABI(abi)//checkABI
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	con.Vm.SetLibLine(libLen)
	errorCode,errorMsg = ExecutedVmSucceed(con.Vm.ExecuteABI(abi, false,false))//execute
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	errorCode,errorMsg = con.Vm.StoreData()//store
	if errorCode != 0{
		return false,nil,types.NewTransactionError(errorCode,errorMsg)
	}
	return true,con.Vm.Logs,nil
}

func(con *Controller) GetGasLeft() uint64{
	return con.GasLeft
}
