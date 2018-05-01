package governance

import (
	"testing"
	"governance/contract"
	"core"
	"fmt"
	"governance/util"
	"common"
	"governance/global"
)

/*
**  Creator: pxf
**  Date: 2018/4/20 下午4:06
**  Description: 
*/


func TestRemove(t *testing.T) {
	core.Clear(core.DefaultBlockChainConfig())
}
func TestPrepare(t *testing.T) {
	prepare()
}

func TestForrange(t *testing.T) {
	ss := []string{}
	ss = append(ss, "1")
	ss = append(ss, "2")
	ss = append(ss, "3")
	ss = append(ss, "4")
	ss = append(ss, "5")
	for _, s := range ss {
		t.Log(s, &s)
	}
}

func showVoterInfo()  {
	callctx := contract.ChainTopCallContext()
	vote := gov.NewVoteInst(callctx, voteAddress)
	vs, err := vote.VoterAddrs()
	if err != nil {
		fmt.Println("获取地址", err)
	}
	for _, voter := range vs {
		voteInfo, err := vote.VoterInfo(util.ToTASAddress(voter))
		if err != nil {
			fmt.Println("获取投票信息", err)
		}
		fmt.Println(common.ToHex(voter.Bytes()), voteInfo)
	}

}

func showCreditInfo() {
	callctx := contract.ChainTopCallContext()
	credit := gov.NewTasCreditInst(callctx)
	for _, voter := range voters {
		ci, err := credit.CreditInfo(voter)
		if err != nil {
			fmt.Println("creditInfo error", err)
		}
		fmt.Println(common.ToHex(voter.Bytes()), ci)
	}
}

func showParams() {
	callctx := contract.ChainTopCallContext()
	ps := gov.NewParamStoreInst(callctx)
	pw := global.NewParamWrapper()

	fmt.Println("params", pw.GetGasPriceMin(ps), pw.GetBlockFixAward(ps), pw.GetVoterCountMin(ps))

	meta, err := ps.GetCurrentMeta(2)
	fmt.Println(meta, err)

	meta, err = ps.GetFutureMeta(2, 0)
	fmt.Println(meta, err)
}
func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return core.Sha256(bytes3)
}
func TestToChain(t *testing.T) {
	common.InitConf("test.ini")
	core.Clear(core.DefaultBlockChainConfig())

	err := core.InitCore()
	if err != nil {
		fmt.Println("初始化失败", err)
	}
	chain = core.BlockChainImpl
	chain.GetTransactionPool().Clear()

	global.InitGov(chain)
	deployAcc := string2Address("122")
	creditTx := &core.Transaction{
		GasPrice: 1,
		Source: &deployAcc,
		Target: &deployAcc,
		Value: 0,
		//Data: common.Hex2Bytes(contract.CREDIT_CODE),
		Hash: string2Hash("creditTx1"),
	}

	sourcebyte := common.BytesToAddress(genHash("2343"))
	targetbyte := common.BytesToAddress(genHash("234"))

	creditTx = &core.Transaction{
		GasPrice: 1,
		Hash:     common.BytesToHash(genHash("jde")),
		Source:   &sourcebyte,
		Target:   &targetbyte,
		Nonce:    0,
		Value:    0,
	}

	sendTx(creditTx)

	ToChain()
	ToChain()
}

func TestVote(t *testing.T) {
	prepare()

	deployVote()

	for _, voter := range voters {
		addDeposit(voter, 2)
	}

	ToChain()//height 5
	//showVoterBalance()
	fmt.Println("vote contract balance ", chain.GetBalance(voteAddress))

	showVoterInfo()

	ToChain()//heigt 16

	////代理
	//delegate(voters[0], voters[9])
	//delegate(voters[1], voters[9])
	//
	////投票
	for i, voter := range voters {
		if i < 2 {
			//continue
		}
		if i >= 2 {
			doVote(voter, true)
		} else {
			doVote(voter, false)
		}
	}
	doVote(common.Address{}, true)

	ToChain()//heigt 17

	fmt.Println("=====after vote====")
	showVoterInfo()

	ToChain() //height 8

	ToChain() //height 9 唱票

	ToChain() //height 10 生效

	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()
	ToChain()

	showVoterBalance()
	fmt.Println("vote contract balance ", chain.GetBalance(voteAddress))

	showCreditInfo()

	showParams()

	fmt.Println("height", chain.Height())
}