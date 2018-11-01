package tvm

import (
	"storage/core/vm"
	"middleware/types"
	"fmt"
	"common"
	"math/big"
	"encoding/json"
	"log"
)

type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction types.Transaction
	AccountDB   vm.AccountDB
	Reader      vm.ChainReader
	Vm          *Tvm
	Tasks       []*CallTask
	LibPath     string
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
	controller.Tasks = make([]*CallTask, 0)
	controller.Vm = nil
	controller.LibPath = libPath
	return controller
}

func (con *Controller) Deploy(sender *common.Address, contract *Contract) bool {
	var succeed bool
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.Transaction.GasLimit))
	msg := Msg{Data: []byte{}, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	succeed = con.Vm.Deploy(msg) && con.Vm.StoreData()
	con.Vm.DelTvm()
	if !succeed {
		return false
	}
	succeed = con.ExecuteTask()
	con.Transaction.GasLimit = uint64(con.Vm.Gas())
	return succeed
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func (con *Controller) ExecuteAbi(sender *common.Address, contract *Contract, abiJson string) bool {
	var succeed bool
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.Transaction.GasLimit))

	//先转账
	if con.Transaction.Value > 0 {
		amount := big.NewInt(int64(con.Transaction.Value))
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.Target, amount)
		} else {
			return false
		}
	}

	msg := Msg{Data: con.Transaction.Data, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	succeed = con.Vm.CreateContractInstance(msg) && con.Vm.LoadContractCode(msg)
	if succeed {
		abi := ABI{}
		err := json.Unmarshal([]byte(abiJson), &abi)
		if err != nil {
			log.Println(err)
		}
		//fmt.Println(abi)
		succeed = con.Vm.checkABI(abi) && con.Vm.ExecuteABI(abi) && con.Vm.StoreData()
		if succeed {
			con.Vm.DelTvm()
			con.Transaction.GasLimit = uint64(con.Vm.Gas())
			succeed = con.ExecuteTask()
		}else {
			con.Vm.DelTvm()
		}
	}else {
		con.Vm.DelTvm()
	}
	con.Transaction.GasLimit = uint64(con.Vm.Gas())
	return succeed
}

//func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
//	return db.GetBalance(addr).Cmp(amount) >= 0
//}

func (con *Controller) ExecuteTask() bool{
	succeed := true
	for _, task := range con.Tasks {
		contract := LoadContract(*task.ContractAddr)
		gasLeft := con.Transaction.GasLimit
		con.Vm = NewTvm(task.Sender, contract, con.LibPath)
		con.Vm.SetGas(int(gasLeft))
		msg := Msg{Data: []byte{}, Value: 0, Sender: task.Sender.GetHexString()}
		abiJson := fmt.Sprintf(`{"FuncName": "%s", "Args": %s}`, task.FuncName, task.Params)
		succeed = con.Vm.CreateContractInstance(msg) && con.Vm.LoadContractCode(msg)
		if succeed {
			abi := ABI{}
			json.Unmarshal([]byte(abiJson), &abi)
			succeed = con.Vm.checkABI(abi) && con.Vm.ExecuteABI(abi) && con.Vm.StoreData()
		}
		if !succeed {
			con.Vm.DelTvm()
			break
		}
		con.Transaction.GasLimit = uint64(con.Vm.Gas())
	}
	return succeed
}
