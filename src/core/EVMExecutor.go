package core

import (
	"vm/common"
	"vm/core/state"
	"vm/core/types"
	"vm/core/vm"
	"vm/crypto"
	"vm/params"
	"vm/core"
	"math/big"
	"vm/common/math"
	"fmt"
	c "common"
)

type EVMExecutor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain
	cfg    vm.Config
}

var TestnetChainConfig = &params.ChainConfig{
	ChainId:        big.NewInt(3),
	HomesteadBlock: big.NewInt(0),
	DAOForkBlock:   nil,
	DAOForkSupport: true,
	EIP150Block:    big.NewInt(0),
	EIP150Hash:     common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"),
	EIP155Block:    big.NewInt(10),
	EIP158Block:    big.NewInt(10),
	ByzantiumBlock: big.NewInt(1700000),

	Ethash: new(params.EthashConfig),
}

func NewEVMExecutor(bc *BlockChain) *EVMExecutor {
	return &EVMExecutor{
		config: TestnetChainConfig,
		cfg:    vm.Config{},
		bc:     bc,
	}
}

func (executor *EVMExecutor) Execute(statedb *state.StateDB, block *Block, processor VoteProcessor) (types.Receipts, []*Transaction, []c.Hash, *common.Hash, uint64) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header
		gp       = new(core.GasPool).AddGas(math.MaxUint64)
	)
	executedTxs := make([]*Transaction, 0)
	errTxs := make([]c.Hash, 0)

	if 0 == len(block.Transactions) {
		hash := statedb.IntermediateRoot(true)
		return receipts, executedTxs, errTxs, &hash, *usedGas
	}

	for _, tx := range block.Transactions {
		var realData []byte
		if nil != processor {
			realData, _ = processor.BeforeExecuteTransaction(block, statedb, tx)
		}

		statedb.Prepare(common.BytesToHash(tx.Hash.Bytes()), common.BytesToHash(header.Hash.Bytes()), 0)
		receipt, _, err := executor.execute(statedb, gp, header, tx, usedGas, executor.cfg, executor.config, realData)
		if err != nil {
			fmt.Printf("[block] fail to execute tx, error: %s, tx: %+v\n", err, tx)
			errTxs = append(errTxs, tx.Hash)
		} else {
			receipts = append(receipts, receipt)
			executedTxs = append(executedTxs, tx)
		}

	}

	if nil != processor {
		processor.AfterAllTransactionExecuted(block, statedb, receipts)
	}

	//accumulateRewards(chain.Config(), state, header, uncles)
	hash := statedb.IntermediateRoot(true)

	return receipts, executedTxs, errTxs, &hash, *usedGas

}

func (executor *EVMExecutor) execute(statedb *state.StateDB, gp *core.GasPool, header *BlockHeader, tx *Transaction, usedGas *uint64, cfg vm.Config, config *params.ChainConfig, realData []byte) (*types.Receipt, uint64, error) {

	context := NewEVMContext(tx, header, executor.bc)
	vmenv := vm.NewEVM(context, statedb, config, cfg)

	_, gas, failed, err := NewSession(statedb, tx, gp, realData).Run(vmenv)
	if err != nil {
		return nil, 0, err
	}

	//statedb.IntermediateRoot(true).Bytes()

	*usedGas += gas

	receipt := types.NewReceipt(nil, failed, *usedGas)
	receipt.TxHash = common.BytesToHash(tx.Hash.Bytes())
	receipt.GasUsed = gas

	if tx.Target == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce)
	}

	receipt.Logs = statedb.GetLogs(common.BytesToHash(tx.Hash.Bytes()))
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, gas, err
}

func NewEVMContext(tx *Transaction, header *BlockHeader, chain *BlockChain) vm.Context {
	return vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(header, chain),
		Origin:      common.BytesToAddress(tx.Source.Bytes()),
		//Coinbase:    nil,
		BlockNumber: new(big.Int).SetUint64(header.Height),
		Time:        new(big.Int).SetInt64(header.CurTime.Unix()),
		Difficulty:  new(big.Int).SetUint64(0),
		GasLimit:    math.MaxUint64,
		GasPrice:    new(big.Int).SetInt64(1),
	}
}

func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

func GetHashFn(ref *BlockHeader, chain *BlockChain) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		header := chain.QueryBlockByHeight(n)
		if nil != header {
			return common.BytesToHash(header.Hash.Bytes())
		}
		return common.Hash{}
	}
}
