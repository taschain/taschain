package global

import (
	"testing"
	"common/abi"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2018/4/19 下午3:35
**  Description: 
*/



func TestVoteConfig_abi(t *testing.T) {

	_abi, err := abi.JSON(strings.NewReader(CONFIG_ABI))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &VoteConfig{
		TemplateName: "test_template",
		PIndex: 2,
		PValue: "103",
		Custom: false,
		Desc: "描述",
		DepositMin: 10,
		TotalDepositMin: 20,
		VoterCntMin: 4,
		ApprovalDepositMin: 14,
		ApprovalVoterCntMin: 4,
		DeadlineBlock: 20000,
		StatBlock: 20003,
		EffectBlock: 2303223,
		DepositGap: 1332,
	}
	var (
		ret []byte
		parseCfg = new(VoteConfig)
	)

	ret, err = _abi.Pack("",
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


	err = _abi.Unpack(parseCfg, "", ret)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(parseCfg)
}

func TestHelper(t *testing.T) {
	cfg := &VoteConfig{
		TemplateName: "test_template",
		PIndex: 2,
		PValue: "103",
		Custom: false,
		Desc: "描述",
		DepositMin: 10,
		TotalDepositMin: 20,
		VoterCntMin: 4,
		ApprovalDepositMin: 14,
		ApprovalVoterCntMin: 4,
		DeadlineBlock: 20000,
		StatBlock: 20003,
		EffectBlock: 2303223,
		DepositGap: 1332,
	}

	ret, err := abiEncode(cfg)
	t.Log(len(ret), err)

	parse, _ := abiDecode(ret)
	t.Log(parse)
}