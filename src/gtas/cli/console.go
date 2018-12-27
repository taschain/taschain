package cli

import (
	"os"
	"fmt"
	"strings"
	"regexp"
	"flag"
	"encoding/json"
	"github.com/howeyc/gopass"
	"github.com/peterh/liner"
)

/*
**  Creator: pxf
**  Date: 2018/12/20 下午1:06
**  Description: 
*/

type baseCmd struct {
	name string
	help string
	fs 	*flag.FlagSet
}

func genbaseCmd(n string, h string) *baseCmd {
	return &baseCmd{
		name:n,
		help: h,
		fs: flag.NewFlagSet(n, flag.ContinueOnError),
	}
}

type newAccountCmd struct {
	baseCmd
	password string
	miner 	bool
}
func genNewAccountCmd() *newAccountCmd {
	c := &newAccountCmd{
		baseCmd: *genbaseCmd("newaccount", "create account"),
	}
	c.fs.StringVar(&c.password, "password", "", "password for the account")
	c.fs.BoolVar(&c.miner, "miner", false, "create the account for miner if set")
	return c
}

func (c *newAccountCmd) parse(args []string) bool {
	err := c.fs.Parse(args)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.password) == "" {
		fmt.Println("please input the password")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type unlockCmd struct {
	baseCmd
	addr string
	//password string
}

func genUnlockCmd() *unlockCmd {
	c := &unlockCmd{
		baseCmd: *genbaseCmd("unlock", "unlock the account"),
	}
	//c.fs.StringVar(&c.password, "password", "", "password for the account")
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *unlockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type balanceCmd struct {
	baseCmd
	addr string
}

func genBalanceCmd() *balanceCmd {
	c := &balanceCmd{
		baseCmd: *genbaseCmd("balance", "get the balance of the current unlocked account"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the account address")
	return c
}

func (c *balanceCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type minerInfoCmd struct {
	baseCmd
	addr string
}

func genMinerInfoCmd() *minerInfoCmd {
	c := &minerInfoCmd{
		baseCmd: *genbaseCmd("minerinfo", "get the info of the miner"),
	}
	c.fs.StringVar(&c.addr, "addr", "", "the miner address")
	return c
}

func (c *minerInfoCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.addr) == "" {
		fmt.Println("please input the address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type connectCmd struct {
	baseCmd
	host string
	port int
}

func genConnectCmd() *connectCmd {
	c := &connectCmd{
		baseCmd: *genbaseCmd("connect", "connect to one tas node"),
	}
	c.fs.StringVar(&c.host, "host", "", "the node ip")
	c.fs.IntVar(&c.port, "port", 8101, "the node port, default is 8101")
	return c
}

func (c *connectCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.host) == "" {
		fmt.Println("please input the host")
		c.fs.PrintDefaults()
		return false
	}
	if c.port == 0 {
		fmt.Println("please input the port")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type txCmd struct {
	baseCmd
	hash string
}

func genTxCmd() *txCmd {
	c := &txCmd{
		baseCmd: *genbaseCmd("tx", "get transaction detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex transaction hash")
	return c
}

func (c *txCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.hash) == "" {
		fmt.Println("please input the transaction hash")
		c.fs.PrintDefaults()
		return false
	}
	return true
}
type blockCmd struct {
	baseCmd
	hash string
	height uint64
}

func genBlockCmd() *blockCmd {
	c := &blockCmd{
		baseCmd: *genbaseCmd("block", "get block detail"),
	}
	c.fs.StringVar(&c.hash, "hash", "", "the hex block hash")
	c.fs.Uint64Var(&c.height, "height", 0, "the block height")
	return c
}

func (c *blockCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

type sendTxCmd struct {
	baseCmd
	to string
	value uint64
	data string
	txType int
	gas uint64
	gasPrice uint64
}

func genSendTxCmd() *sendTxCmd {
	c := &sendTxCmd{
		baseCmd: *genbaseCmd("sendtx", "send a transaction to the tas system"),
	}
	c.fs.StringVar(&c.to, "to", "", "the transaction receiver address")
	c.fs.Uint64Var(&c.value, "value", 0, "transfer value")
	c.fs.StringVar(&c.data, "data", "", "transaction data")
	c.fs.IntVar(&c.txType, "type", 0, "transaction type: 0=general tx, 1=contract create, 2=contract call, 3=bonus, 4=miner apply,5=miner abort, 6=miner refund")
	c.fs.Uint64Var(&c.gas, "gas", 0, "gas limit")
	c.fs.Uint64Var(&c.gasPrice, "gasprice", 0, "gas price")
	return c
}

func (c *sendTxCmd) toTxRaw() *txRawData {
	return &txRawData{
		Target:   c.to,
		Value:    c.value,
		TxType:   c.txType,
		Data:     c.data,
		Gas:      c.gas,
		Gasprice: c.gasPrice,
	}
}

func (c *sendTxCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	if strings.TrimSpace(c.to) == "" {
		fmt.Println("please input the target address")
		c.fs.PrintDefaults()
		return false
	}
	return true
}

type minerApplyCmd struct {
	baseCmd
	stake uint64
	mtype int
	gas  uint64
	gasprice uint64
}

func genMinerApplyCmd() *minerApplyCmd {
	c := &minerApplyCmd{
		baseCmd: *genbaseCmd("minerapply", "apply to be a miner"),
	}
	c.fs.Uint64Var(&c.stake, "stake", 10, "freeze stake, default 10")
	c.fs.IntVar(&c.mtype, "type", 0, "apply miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.Uint64Var(&c.gas, "gas", 10000, "gaslimit for the transaction, default 10000")
	c.fs.Uint64Var(&c.gasprice, "gasprice", 100, "gasprice for the transaction, default 100")
	return c
}

func (c *minerApplyCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

type minerAbortCmd struct {
	baseCmd
	mtype int
	gas  uint64
	gasprice uint64
}

func genMinerAbortCmd() *minerAbortCmd {
	c := &minerAbortCmd{
		baseCmd: *genbaseCmd("minerabort", "abort a miner identifier"),
	}
	c.fs.IntVar(&c.mtype, "type", 0, "abort miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.Uint64Var(&c.gas, "gas", 10000, "gaslimit for the transaction, default 10000")
	c.fs.Uint64Var(&c.gasprice, "gasprice", 100, "gasprice for the transaction, default 100")
	return c
}

func (c *minerAbortCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

type minerRefundCmd struct {
	baseCmd
	mtype int
	gas  uint64
	gasprice uint64
}

func genMinerRefundCmd() *minerRefundCmd {
	c := &minerRefundCmd{
		baseCmd: *genbaseCmd("minerrefund", "apply to refund the miner freeze stake"),
	}
	c.fs.IntVar(&c.mtype, "type", 0, "refund miner type: 0=verify node, 1=proposal node, default 0")
	c.fs.Uint64Var(&c.gas, "gas", 10000, "gaslimit for the transaction, default 10000")
	c.fs.Uint64Var(&c.gasprice, "gasprice", 100, "gasprice for the transaction, default 100")
	return c
}

func (c *minerRefundCmd) parse(args []string) bool {
	if err := c.fs.Parse(args); err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

var cmdNewAccount = genNewAccountCmd()
var cmdExit = genbaseCmd("exit", "quit  gtas")
var cmdHelp = genbaseCmd("help", "show help info")
var cmdAccountList = genbaseCmd("accountlist", "list the account of the keystore")
var cmdUnlock = genUnlockCmd()
var cmdBalance = genBalanceCmd()
var cmdAccountInfo = genbaseCmd("accountinfo", "get the info of the current unlocked account")
var cmdDelAccount = genbaseCmd("delaccount", "delete the info of the current unlocked account")
var cmdMinerInfo = genMinerInfoCmd()
var cmdConnect = genConnectCmd()
var cmdBlockHeight = genbaseCmd("blockheight", "the current block height")
var cmdGroupHeight = genbaseCmd("groupheight", "the current group height")
var cmdTx = genTxCmd()
var cmdBlock = genBlockCmd()
var cmdSendTx = genSendTxCmd()
var cmdMinerApply = genMinerApplyCmd()
var cmdMinerAbort = genMinerAbortCmd()
var cmdMinerRefund = genMinerRefundCmd()

var list = make([]*baseCmd, 0)

func init() {
	list = append(list, cmdHelp)
	list = append(list, &cmdNewAccount.baseCmd)
	list = append(list, cmdAccountList)
	list = append(list, &cmdUnlock.baseCmd)
	list = append(list, &cmdBalance.baseCmd)
	list = append(list, cmdAccountInfo)
	list = append(list, cmdDelAccount)
	list = append(list, &cmdMinerInfo.baseCmd)
	list = append(list, &cmdConnect.baseCmd)
	list = append(list, cmdBlockHeight)
	list = append(list, cmdGroupHeight)
	list = append(list, &cmdTx.baseCmd)
	list = append(list, &cmdBlock.baseCmd)
	list = append(list, &cmdSendTx.baseCmd)
	list = append(list, &cmdMinerApply.baseCmd)
	list = append(list, &cmdMinerAbort.baseCmd)
	list = append(list, &cmdMinerRefund.baseCmd)
	list = append(list, cmdExit)
}

func Usage() {
	fmt.Println("Usage:")
	for _, cmd := range list {
		fmt.Println(" " + cmd.name + ":\t" + cmd.help)
		cmd.fs.PrintDefaults()
		fmt.Print("\n")
	}
}


func ConsoleInit(host string, port int, show bool) error {

	chainop := InitRemoteChainOp(host, port, show, AccountOp)
	if chainop.base != "" {

	}

	loop(AccountOp, chainop)

	return nil
}

func handleCmd(handle func() *Result) {
	ret := handle()
	if !ret.IsSuccess() {
		fmt.Println(ret.Message)
	} else {
		bs, err := json.MarshalIndent(ret, "", "\t")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(bs))
		}
	}
}

func unlockLoop(cmd *unlockCmd, acm accountOp) {
	c := 0

	for c < 3 {
		c++

		bs, err := gopass.GetPasswdPrompt("please input password: ", true, os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		ret := acm.UnLock(cmd.addr, string(bs))
		if ret.IsSuccess() {
			fmt.Printf("unlock will last %v secs:%v\n", accountUnLockTime.Seconds(), cmd.addr)
			break
		} else {
			fmt.Fprintln(os.Stderr, ret.Message)
		}
	}
}

func loop(acm accountOp, chainOp chainOp) {

	re, _ := regexp.Compile("\\s{2，}")


	//reader := bufio.NewReader(os.Stdin)

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	items := make([]string, len(list))
	for idx, cmd := range list {
		items[idx] = cmd.name
	}

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range items {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})


	for {
		ep := chainOp.Endpoint()
		if ep == ":0" {
			ep = "not connected"
		}
		//input, err := reader.ReadString('\n')
		input, err := line.Prompt(fmt.Sprintf("gtas:%v > ", ep))
		if err != nil {
			if err == liner.ErrPromptAborted {
				line.Close()
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, err)
		}
		input = strings.TrimSpace(input)
		input = re.ReplaceAllString(input, " ")
		inputArr := strings.Split(input, " ")

		line.AppendHistory(input)

		if len(inputArr) == 0 {
			continue
		}
		cmdStr := inputArr[0]

		switch cmdStr {
		case "":
			break
		case cmdNewAccount.name:
			cmd := genNewAccountCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return acm.NewAccount(cmd.password, cmd.miner)
				})
			}
			//fmt.Printf("pass %v, miner %v\n", cmd.password, cmd.miner)
		case cmdExit.name, "quit":
			fmt.Printf("thank you, bye\n")
			line.Close()
			os.Exit(0)
		case cmdHelp.name:
			Usage()
		case cmdAccountList.name:
			handleCmd(func() *Result {
				return acm.AccountList()
			})
		case cmdUnlock.name:
			cmd := genUnlockCmd()
			if cmd.parse(inputArr[1:]) {
				unlockLoop(cmd, acm)
			}
		case cmdAccountInfo.name:
			handleCmd(func() *Result {
				return acm.AccountInfo()
			})
		case cmdDelAccount.name:
			handleCmd(func() *Result {
				return acm.DeleteAccount()
			})
		case cmdConnect.name:
			cmd := genConnectCmd()
			if cmd.parse(inputArr[1:]) {
				chainOp.Connect(cmd.host, cmd.port)
			}

		case cmdBalance.name:
			cmd := genBalanceCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.Balance(cmd.addr)
				})
			}
		case cmdMinerInfo.name:
			cmd := genMinerInfoCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.MinerInfo(cmd.addr)
				})
			}
		case cmdBlockHeight.name:
			handleCmd(func() *Result {
				return chainOp.BlockHeight()
			})
		case cmdGroupHeight.name:
			handleCmd(func() *Result {
				return chainOp.GroupHeight()
			})
		case cmdTx.name:
			cmd := genTxCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.TxInfo(cmd.hash)
				})
			}
		case cmdBlock.name:
			cmd := genBlockCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					if cmd.hash != "" {
						return chainOp.BlockByHash(cmd.hash)
					} else {
						return chainOp.BlockByHeight(cmd.height)
					}
				})
			}
		case cmdSendTx.name:
			cmd := genSendTxCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.SendRaw(cmd.toTxRaw())
				})
			}

		case cmdMinerApply.name:
			cmd := genMinerApplyCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.ApplyMiner(cmd.mtype, cmd.stake, cmd.gas, cmd.gasprice)
				})
			}
		case cmdMinerAbort.name:
			cmd := genMinerAbortCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.AbortMiner(cmd.mtype, cmd.gas, cmd.gasprice)
				})
			}
		case cmdMinerRefund.name:
			cmd := genMinerRefundCmd()
			if cmd.parse(inputArr[1:]) {
				handleCmd(func() *Result {
					return chainOp.RefundMiner(cmd.mtype, cmd.gas, cmd.gasprice)
				})
			}
		default:
			fmt.Printf("not supported command %v\n", cmdStr)
			Usage()
		}
	}
}
