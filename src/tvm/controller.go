package tvm

import (
	"storage/core/vm"
	"middleware/types"
	"fmt"
	"common"
	"math/big"
	"core"
)

type Controller struct {
	BlockHeader *types.BlockHeader
	Transaction *types.Transaction
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
	controller.Transaction = transaction
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
	snapshot := con.AccountDB.Snapshot()
	succeed = con.Vm.Deploy(msg) && con.Vm.StoreData()
	if !succeed {
		con.AccountDB.RevertToSnapshot(snapshot)
	}
	con.Vm.DelTvm()
	con.ExecuteTask()
	return succeed
}

func (con *Controller) ExecuteAbi(sender *common.Address, contract *Contract, abi string) bool {
	var succeed bool
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(1000000)

	//先转账
	if con.Transaction.Value > 0 {
		amount := big.NewInt(int64(con.Transaction.Value))
		if core.CanTransfer(con.AccountDB, *sender, amount) {
			core.Transfer(con.AccountDB, *sender, *con.Transaction.Target, amount)
		} else {
			return false
		}
	}

	msg := Msg{Data: con.Transaction.Data, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	succeed = con.Vm.LoadContractCode()
	if succeed {
		con.Vm.SetGas(int(con.Transaction.GasLimit))
		succeed = con.Vm.ExecuteABIJson(msg, abi) && con.Vm.StoreData()
	}

	con.Vm.DelTvm()
	con.ExecuteTask()
	return succeed
}

//func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
//	return db.GetBalance(addr).Cmp(amount) >= 0
//}

func (con *Controller) ExecuteTask() {
	var succeed bool
	for _, task := range con.Tasks {
		contract := LoadContract(*task.ContractAddr)
		gasLeft := con.Vm.Gas()
		con.Vm = NewTvm(task.Sender, contract, con.LibPath)
		con.Vm.SetGas(1000000)
		snapshot := con.AccountDB.Snapshot()
		msg := Msg{Data: []byte{}, Value: 0, Sender: task.Sender.GetHexString()}
		abi := fmt.Sprintf(`{"FuncName": "%s", "Args": %s}`, task.FuncName, task.Params)
		succeed = con.Vm.LoadContractCode()
		if succeed {
			con.Vm.SetGas(gasLeft)
			succeed = con.Vm.LoadContractCode() && con.Vm.ExecuteABIJson(msg, abi) && con.Vm.StoreData()
		}
		if !succeed {
			if con.Vm.Gas() == 0 {
				con.Vm.DelTvm()
				return
			}
			con.AccountDB.RevertToSnapshot(snapshot)
			con.Vm.DelTvm()
			continue
		}
	}
}
