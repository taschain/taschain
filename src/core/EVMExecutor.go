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
)

type EVMExecutor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain
	cfg    vm.Config
}

// NewStateProcessor initialises a new StateProcessor.
func NewEVMExecutor(bc *BlockChain) *EVMExecutor {
	testnetChainConfig := &params.ChainConfig{
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

	return &EVMExecutor{
		config: testnetChainConfig,
		bc:     bc,
	}
}

func (executor *EVMExecutor) Execute(statedb *state.StateDB, block *Block) (types.Receipts, *common.Hash, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header
		gp       = new(core.GasPool).AddGas(math.MaxUint64)
	)

	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions {
		statedb.Prepare(common.BytesToHash(tx.Hash.Bytes()), common.BytesToHash(header.Hash.Bytes()), i)
		receipt, _, err := executor.execute(statedb, gp, header, tx, usedGas, executor.cfg, executor.config)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)

	}

	// Accumulate any block and uncle rewards and commit the final state root
	//accumulateRewards(chain.Config(), state, header, uncles)
	hash := statedb.IntermediateRoot(true)

	return receipts, &hash, *usedGas, nil

}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func (executor *EVMExecutor) execute(statedb *state.StateDB, gp *core.GasPool, header *BlockHeader, tx *Transaction, usedGas *uint64, cfg vm.Config, config *params.ChainConfig) (*types.Receipt, uint64, error) {

	// Create a new context to be used in the EVM environment
	context := NewEVMContext(tx, header, executor.bc)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)
	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := NewSession(statedb, tx, gp).run(vmenv)
	if err != nil {
		return nil, 0, err
	}
	// Update the state with pending changes
	root := statedb.IntermediateRoot(true).Bytes()

	*usedGas += gas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = common.BytesToHash(tx.Hash.Bytes())
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if tx.Target == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce)
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(common.BytesToHash(tx.Hash.Bytes()))
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return receipt, gas, err
}

// NewEVMContext creates a new context for use in the EVM.
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

// CanTransfer checks wether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *BlockHeader, chain *BlockChain) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		header := chain.QueryBlockByHeight(n)
		if nil != header {
			return common.BytesToHash(header.Hash.Bytes())
		}
		return common.Hash{}
	}
}
