package core

import (
	"middleware/types"
	"common"
	t "storage/core/types"
	"storage/core"
	"tvm"
	"math/big"
	"storage/core/vm"
	"fmt"
)

type TVMExecutor struct {
	bc     *BlockChain
}

func NewTVMExecutor(bc *BlockChain) *TVMExecutor {
	return &TVMExecutor{
		bc:     bc,
	}
}

func (executor *TVMExecutor) Execute(accountdb *core.AccountDB, block *types.Block, processor VoteProcessor) (common.Hash,[]t.Receipt,error) {
	if 0 == len(block.Transactions) {
		hash := accountdb.IntermediateRoot(false)
		Logger.Infof("TVMExecutor Execute Hash:%s",hash.Hex())
		return hash, nil, nil
	}

	vm := tvm.NewTvm(accountdb)
	receipts := make([]t.Receipt,len(block.Transactions))
	for i,transaction := range block.Transactions{
		receipt := t.Receipt{}
		if transaction.Target == nil{
			receipt.ContractAddress,_ = createContract(accountdb, transaction)
		} else if len(transaction.Data) > 0 {
			snapshot := accountdb.Snapshot()
			script := string(accountdb.GetCode(*transaction.Target))
			if !vm.Execute(script){
				accountdb.RevertToSnapshot(snapshot)
			}
		} else {
			amount := big.NewInt(int64(transaction.Value))
			if CanTransfer(accountdb, *transaction.Source, amount){
				Transfer(accountdb, *transaction.Source, *transaction.Target, amount)
			}
		}
		receipt.TxHash = transaction.Hash
		receipts[i] = receipt
	}

	//if nil != processor {
	//	processor.AfterAllTransactionExecuted(block, statedb, receipts)
	//}

	return accountdb.IntermediateRoot(false), receipts, nil
}

func createContract(accountdb *core.AccountDB, transaction *types.Transaction) (common.Address, error) {
	amount := big.NewInt(int64(transaction.Value))
	if !CanTransfer(accountdb, *transaction.Source, amount){
		return common.Address{}, fmt.Errorf("balance not enough")
	}

	nance := accountdb.GetNonce(*transaction.Source)
	accountdb.SetNonce(*transaction.Source, nance + 1)
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine(transaction.Source[:], common.Uint64ToByte(nance))))

	if accountdb.GetCodeHash(contractAddr) != (common.Hash{}){
		return common.Address{}, fmt.Errorf("contract address conflict")
	}
	accountdb.CreateAccount(contractAddr)
	accountdb.SetCode(contractAddr, transaction.Data)
	accountdb.SetNonce(contractAddr, 1)

	Transfer(accountdb, *transaction.Source, contractAddr, amount)
	return contractAddr, nil
}

func CanTransfer(db vm.AccountDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func Transfer(db vm.AccountDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
