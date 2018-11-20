package tvm

import (
	"common"
	"encoding/json"
	"fmt"
	"math/big"
	"middleware/types"
	"storage/core/vm"
	tt "storage/core/types"
)

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

func (con *Controller) Deploy(sender *common.Address, contract *Contract) bool {
	var succeed bool
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.GasLeft))
	msg := Msg{Data: []byte{}, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	succeed = con.Vm.Deploy(msg) && con.Vm.StoreData()
	con.Vm.DelTvm()
	if !succeed {
		return false
	}
	con.GasLeft = uint64(con.Vm.Gas())
	return succeed
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func (con *Controller) ExecuteAbi(sender *common.Address, contract *Contract, abiJson string) (bool,*[]*tt.Log) {
	var succeed bool
	con.Vm = NewTvm(sender, contract, con.LibPath)
	con.Vm.SetGas(int(con.GasLeft))

	//先转账
	if con.Transaction.Value > 0 {
		amount := big.NewInt(int64(con.Transaction.Value))
		if CanTransfer(con.AccountDB, *sender, amount) {
			transfer(con.AccountDB, *sender, *con.Transaction.Target, amount)
		} else {
			return false,nil
		}
	}

	msg := Msg{Data: con.Transaction.Data, Value: con.Transaction.Value, Sender: con.Transaction.Source.GetHexString()}
	succeed = con.Vm.CreateContractInstance(msg)
	if succeed {
		abi := ABI{}
		json.Unmarshal([]byte(abiJson), &abi)
		fmt.Println(abi)
		succeed = con.Vm.checkABI(abi) && ExecutedVmSucceed(con.Vm.ExecuteABI(abi, false)) && con.Vm.StoreData()
		if succeed {
			con.Vm.DelTvm()
			con.GasLeft = uint64(con.Vm.Gas())
		} else {
			//todo 告知用户明确失败的原因，如ABI非法
			con.Vm.DelTvm()
		}
	} else {
		con.Vm.DelTvm()
	}
	con.GasLeft = uint64(con.Vm.Gas())
	return succeed,&(con.Vm.Logs)
}

func(con *Controller) GetGasLeft() uint64{
	return con.GasLeft
}
