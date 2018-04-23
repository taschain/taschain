package global

import (
	"common/abi"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2018/4/19 下午5:20
**  Description: 
*/

const CONFIG_ABI = `[{"inputs":[{"name":"TemplateName","type":"string"},{"name":"PIndex","type":"uint32"},{"name":"PValue","type":"string"},{"name":"Custom","type":"bool"},{"name":"Desc","type":"string"},{"name":"DepositMin","type":"uint64"},{"name":"TotalDepositMin","type":"uint64"},{"name":"VoterCntMin","type":"uint64"},{"name":"ApprovalDepositMin","type":"uint64"},{"name":"ApprovalVoterCntMin","type":"uint64"},{"name":"DeadlineBlock","type":"uint64"},{"name":"StatBlock","type":"uint64"},{"name":"EffectBlock","type":"uint64"},{"name":"DepositGap","type":"uint64"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`

var helperAbi abi.ABI

func init() {
	helperAbi, _ = abi.JSON(strings.NewReader(CONFIG_ABI))
}

func abiEncode(cfg *VoteConfig) ([]byte, error) {
	ret, err := helperAbi.Pack("",
		cfg.TemplateName,
		cfg.PIndex,
		cfg.PValue,
		cfg.Custom,
		cfg.Desc,
		cfg.DepositMin,
		cfg.TotalDepositMin,
		cfg.VoterCntMin,
		cfg.ApprovalDepositMin,
		cfg.ApprovalVoterCntMin,
		cfg.DeadlineBlock,
		cfg.StatBlock,
		cfg.EffectBlock,
		cfg.DepositGap)

	if err != nil {
		return nil, err
	}
	return ret, nil
}

func abiDecode(data []byte) (*VoteConfig, error) {
	ret := new(VoteConfig)
	err := helperAbi.Unpack(ret, "", data)
	return ret, err
}