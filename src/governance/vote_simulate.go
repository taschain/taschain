package governance

import (
	"core"
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

	voters []common.Address
	voteAddress common.Address
	//vote *contract.Vote

	idx int64
)

const VOTE_TEMPLATE_1 = "vote_template_1"

func ToChain() {
	b := chain.CastingBlock()
	ret := chain.AddBlockOnChain(b)
	if ret < 0 {
		fmt.Println("上链失败, 高度=", b.Header.Height)
	}
}


func prepare() {
	idx = 100
	common.InitConf("test.ini")
	core.Clear(core.DefaultBlockChainConfig())

	err := core.InitCore()
	if err != nil {
		fmt.Println("初始化失败", err)
	}
	chain = core.BlockChainImpl

	chain.GetTransactionPool().Clear()

	//初始化治理环境
	global.InitGov(chain)
	gov = global.GetGOV()


	deployAcc := string2Address("3")
	nonce := chain.GetNonce(deployAcc)

	fmt.Println("init balance ", chain.GetBalance(deployAcc))

	//部署信用合约
	creditTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Nonce: nonce,
		Data: common.Hex2Bytes(contract.CREDIT_CODE),
		Hash: string2Hash("creditTx"),
	}
	nonce++
	sendTx(creditTx)

	//
	//部署代码存储合约
	templateTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Nonce: nonce,
		Data: common.Hex2Bytes(contract.TEMPLATE_CODE),
		Hash: string2Hash("templateTx"),
	}
	nonce++
	sendTx(templateTx)

	//部署投票合约地址存储合约
	voteAddrTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Nonce: nonce,
		Data: common.Hex2Bytes(contract.VOTE_ADDR_POOL_CODE),
		Hash: string2Hash("voteAddrPoolTx"),
	}
	nonce++
	sendTx(voteAddrTx)

	//部署参数存储存储合约
	paramStoreTx := &core.Transaction{
		GasPrice: 1,
		GasLimit: 10,
		Source: &deployAcc,
		Target: nil,
		Value: 0,
		Nonce: nonce,
		Data: common.Hex2Bytes(contract.PARAM_STORE_CODE),
		Hash: string2Hash("paramStoreTx"),
	}
	nonce++
	sendTx(paramStoreTx)


	ToChain()
	nonce = chain.GetNonce(deployAcc)

	input, err := gov.CodeContract.GetAbi().Pack("addTemplate",
		string2Address(VOTE_TEMPLATE_1),
		common.Hex2Bytes(contract.VOTE_CODE),
		contract.VOTE_ABI)
	if err != nil {
		fmt.Println("添加模板abi pack失败", err)
		panic(err)
	}

	//添加模板
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

	nonce = chain.GetNonce(deployAcc)
	//转账
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
		nonce++
		voters = append(voters, acc)
	}
	ToChain()

	fmt.Println("deploy acc balance ", chain.GetBalance(deployAcc))
	//showVoterBalance()
}

func showVoterBalance() {
	for _, v := range voters {
		fmt.Println("voters ", common.Bytes2Hex(v.Bytes()), chain.GetBalance(v))
	}
}


func sendTx(tx *core.Transaction) {
	//nonce := chain.GetNonce(*tx.Source)
	//tx.Nonce = nonce
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
		DepositMin: 1,
		TotalDepositMin: 2,
		VoterCntMin: 4,
		ApprovalDepositMin: 2,
		ApprovalVoterCntMin: 2,
		DeadlineBlock: 8,
		StatBlock: 9,
		EffectBlock: 10,
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
	//code, err = GetRealCode(core.GetTopBlock(chain), core.GetStateDB(chain), VOTE_TEMPLATE_1, input)

	launcher := string2Address("2")

	voteAddress = util.ToTASAddress(crypto.CreateAddress(util.ToETHAddress(launcher), 0))
	tx := &core.Transaction{
		Data: input,
		Source: &launcher,
		GasLimit: 10,
		GasPrice: 1,
		ExtraData: append(voteAddress.Bytes(), input...),
		Hash: string2Hash(nextIdx()),
		ExtraDataType: 1,
	}

	//vote = corei.NewVoteInst(callctx, voteAddress)

	sendTx(tx)
	ToChain()	//height 15

	fmt.Println("launcher balance ", chain.GetBalance(launcher))
	fmt.Println("vote contract balance ", chain.GetBalance(voteAddress))
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
	tx.GasLimit = 10
	tx.Target = &voteAddress
	tx.Hash = string2Hash(nextIdx())
	tx.Nonce = chain.GetNonce(*tx.Source)
	sendTx(tx)
}

func sendTransaction2(voter common.Address, method string, args ...interface{})  {
	sendTransaction(&core.Transaction{
		Source: &voter,
	}, method, args...)
}

func addDeposit(voter common.Address, value uint64) {

	sendTransaction(&core.Transaction{
		Source: &voter,
		Value: value,
	}, "addDeposit", value)
}

func doVote(voter common.Address, p bool)  {
	sendTransaction2(voter, "vote", p)
}

func delegate(voter common.Address, delegate common.Address) {
	sendTransaction2(voter, "delegateTo", delegate)
}


func string2Address(s string) common.Address {
	return util.String2Address(s)
}

func string2Hash(s string) common.Hash {
	return util.String2Hash(s)
}
