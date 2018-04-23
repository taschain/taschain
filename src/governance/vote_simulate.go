package governance

import (
	"core"
	"vm/core/vm"
	"governance/global"
	"governance/contract"
	"common"
	"fmt"
	"governance/util"
	"vm/crypto"
	"strconv"
)

/*
**  Creator: pxf
**  Date: 2018/4/20 下午2:41
**  Description: 
*/

var (
	chain *core.BlockChain
	//block *core.Block
	//state vm.StateDB
	gov *global.GOV
	//template *contract.TemplateCode
	//callctx *contract.CallContext

	launcher common.Address
	voters []common.Address
	voteAddress common.Address
	//vote *contract.Vote

	idx int64
)

const VOTE_TEMPLATE_1 = "vote_template_1"

func ToChain() {
	chain.AddBlockOnChain(chain.CastingBlock())
}

func prepare() {
	idx = 100
	common.InitConf("test.ini")
	core.Clear(core.DefaultBlockChainConfig())

	chain = core.InitBlockChain()

	//初始化治理环境
	global.InitGov(chain)
	gov = global.GetGOV()

	nonce := uint64(0)

	deployAcc := string2Address("3")

	fmt.Println("init balance ", chain.GetBalance(deployAcc))

	creditTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Data: common.Hex2Bytes(contract.CREDIT_CODE),
		Hash: string2Hash("creditTx"),
	}
	sendTx(creditTx)

	ToChain()

	//
	//部署合约2
	//_, _, _ = contract.SimulateDeployContract(callctx, global.DEPLOY_ACCOUNT, contract.TEMPLATE_ABI, contract.TEMPLATE_CODE)
	templateTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Data: common.Hex2Bytes(contract.TEMPLATE_CODE),
		Hash: string2Hash("templateTx"),
	}
	sendTx(templateTx)
	ToChain()

	input, err := gov.CodeContract.GetAbi().Pack("addTemplate",
		string2Address("vote_template"),
		common.Hex2Bytes(contract.VOTE_CODE),
		contract.VOTE_ABI)
	if err != nil {
		fmt.Println("添加模板abi pack失败", err)
		panic(err)
	}

	codeAddr := gov.CodeContract.GetAddress()
	addTemplateTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: &codeAddr,
		Nonce: nonce,
		Value: 0,
		Data: input,
		Hash: string2Hash("addTemplateTx"),
	}
	sendTx(addTemplateTx)
	ToChain()

	for i := 0; i < 10; i ++ {
		acc := string2Address("voters_" + strconv.FormatInt(int64(i), 10))
		transferTx := &core.Transaction{
			GasPrice: 1,
			GasLimit: 10,
			Source: &deployAcc,
			Target: &acc,
			Nonce: nonce,
			Value: 10,
			Hash: string2Hash("hash_" + strconv.FormatInt(int64(i), 10)),
		}
		sendTx(transferTx)
		voters = append(voters, acc)
		ToChain()
	}

	ret := chain.AddBlockOnChain(chain.CastingBlock())
	if ret < 0 {
		panic("上链失败 1")
	}

	fmt.Println("deploy acc balance ", chain.GetBalance(deployAcc))
	for _, v := range voters {
		fmt.Println("voters ", common.Bytes2Hex(v.Bytes()), chain.GetBalance(v))
	}
}

func sendTx(tx *core.Transaction) {
	if nonce := chain.GetNonce(*tx.Source); nonce == 0 {
		tx.Nonce = 0
	} else {
		tx.Nonce = nonce + 1
	}
	chain.GetTransactionPool().Add(tx)
}

func deployVote() {
	//配置项
	cfg := &global.VoteConfig{
		TemplateName: VOTE_TEMPLATE_1,
		PIndex: 2,
		PValue: "103",
		Custom: false,
		Desc: "描述",
		DepositMin: 10,
		TotalDepositMin: 20,
		VoterCntMin: 4,
		ApprovalDepositMin: 20,
		ApprovalVoterCntMin: 4,
		DeadlineBlock: 4,
		StatBlock: 5,
		EffectBlock: 6,
		DepositGap: 1,
	}

	//配置项编码
	input, err := cfg.AbiEncode()
	if err != nil {
		fmt.Println("部署投票失败", err)
		return
	}

	//获取真正的执行代码
	//var code []byte
	//code, err = GetRealCode(block, state, VOTE_TEMPLATE_1, input)

	voteAddress = util.ToTASAddress(crypto.CreateAddress(util.ToETHAddress(launcher), 0))
	tx := &core.Transaction{
		Data: input,
		Source: &launcher,
		GasLimit: 100000,
		GasPrice: 1,
		ExtraData: voteAddress.Bytes(),
		Hash: string2Hash(nextIdx()),
	}

	idx += 1
	//vote = gov.NewVoteInst(callctx, voteAddress)

	sendTx(tx)

}

func nextIdx() string {
	idx += 100
	return strconv.FormatInt(idx, 10)
}
func sendTransaction(tx *core.Transaction, method string, args ...interface{}) {
	input, err := gov.VoteContract.GetAbi().Pack(method, args...)
	if err != nil {
		fmt.Println(method, err, tx)
		return
	}
	tx.Data  = input
	tx.GasPrice = 1
	tx.GasLimit = 100000
	tx.Target = &voteAddress
	tx.Hash = string2Hash(nextIdx())
	sendTx(tx)
}

func sendTransaction2(voter *common.Address, method string, args ...interface{})  {
	sendTransaction(&core.Transaction{
		Source: voter,
	}, method, args...)
}

func addDeposit(voter *common.Address, value uint64) {

	sendTransaction(&core.Transaction{
		Source: voter,
		Value: value,
	}, "addDeposit", value)
}

func doVote(voter *common.Address, p bool)  {
	sendTransaction2(voter, "vote", p)
}

func delegate(voter *common.Address, delegate common.Address) {
	sendTransaction2(voter, "delegateTo", delegate)
}


func hashBytes(hash string) []byte {
	bytes3 := []byte(hash)
	return core.Sha256(bytes3)
}

func string2Address(s string) common.Address {
	return common.BytesToAddress(hashBytes(s))
}

func string2Hash(s string) common.Hash {
	return common.BytesToHash(hashBytes(s))
}

func newStateDB() vm.StateDB {
	return core.NewStateDB(chain.QueryTopBlock().StateTree, chain)
}