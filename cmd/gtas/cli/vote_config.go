//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cli

import (
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"reflect"
	"strconv"
	"strings"
)

type VoteConfig struct {
	TemplateName        string `json:"template_name"`          //合约模板名称
	PIndex              uint32 `json:"p_index"`                //投票参数索引
	PValue              string `json:"p_value"`                //投票值
	Custom              bool   `json:"custom"`                 //'是否自定义投票合约', true时, pIndex pValue无效
	Desc                string `json:"desc"`                   //描述
	DepositMin          uint64 `json:"deposit_min"`            //每个投票人最低缴纳保证金
	TotalDepositMin     uint64 `json:"total_deposit_min"`      //最低总保证金
	VoterCntMin         uint64 `json:"voter_cnt_min"`          //最低参与投票人
	ApprovalDepositMin  uint64 `json:"approval_deposit_min"`   //通过的最低保证金
	ApprovalVoterCntMin uint64 `json:"approval_voter_cnt_min"` //通过的最低投票人
	DeadlineBlock       uint64 `json:"deadline_block"`         //投票截止的最高区块高度
	StatBlock           uint64 `json:"stat_block"`             //唱票区块高度
	EffectBlock         uint64 `json:"effect_block"`           //生效高度
	DepositGap          uint64 `json:"deposit_gap"`            //缴纳保证金后到可以投票的间隔, 用区块高度差表示
}

func (v VoteConfig) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

//func (v VoteConfig) ToGlobal() *global.VoteConfig {
//	return &global.VoteConfig{
//		TemplateName:        v.TemplateName,
//		PIndex:              v.PIndex,
//		PValue:              v.PValue,
//		Custom:              v.Custom,
//		Desc:                v.Desc,
//		DepositMin:          v.DepositMin,
//		TotalDepositMin:     v.TotalDepositMin,
//		VoterCntMin:         v.VoterCntMin,
//		ApprovalDepositMin:  v.ApprovalDepositMin,
//		ApprovalVoterCntMin: v.ApprovalVoterCntMin,
//		DeadlineBlock:       v.DeadlineBlock,
//		StatBlock:           v.StatBlock,
//		EffectBlock:         v.EffectBlock,
//		DepositGap:          v.DepositGap,
//	}
//}

// VoteConfigList 解析vote命令中的传入的config参数。
type VoteConfigKvs [][2]string

// Set value from command params
func (vcl *VoteConfigKvs) Set(value string) error {
	kv := strings.Split(value, "=")
	if len(kv) != 2 {
		return fmt.Errorf("'%s' is not like 'key=value', please confirm your input", value)
	}
	*vcl = append(*vcl, [2]string{kv[0], kv[1]})
	return nil
}

// String
func (vcl *VoteConfigKvs) String() string {
	return ""
}

// IsCumulative 表示选项可叠加
func (vcl *VoteConfigKvs) IsCumulative() bool {
	return true
}

// ToVoteConfig vote命令行参数转换为core.VoteConfig
func (vcl *VoteConfigKvs) ToVoteConfig() (*VoteConfig, error) {
	voteConfig := &VoteConfig{}
	refVc := reflect.ValueOf(voteConfig).Elem()
	for _, kv := range *vcl {
		fd := refVc.FieldByName(kv[0])
		if !fd.CanSet() {
			// 传入未定义参数，提示用户
			// TODO 提示
		} else {
			switch fd.Kind() {
			case reflect.Interface:
				fd.Set(reflect.ValueOf(kv[1]))
			case reflect.Int:
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return nil, err
				}
				fd.Set(reflect.ValueOf(v))
			case reflect.Uint64:
				// TODO 数字过大时转换有问题
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return nil, err
				}
				if v < 0 {
					return nil, fmt.Errorf("'%d' is not a uint64 param", v)
				}
				fd.Set(reflect.ValueOf(uint64(v)))
			case reflect.Uint32:
				// TODO 数字过大时转换有问题
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return nil, err
				}
				if v < 0 {
					return nil, fmt.Errorf("'%d' is not a uint32 param", v)
				}
				fd.Set(reflect.ValueOf(uint32(v)))
			case reflect.String:
				fd.Set(reflect.ValueOf(kv[1]))
			case reflect.Bool:
				v, err := strconv.ParseBool(kv[1])
				if err != nil {
					return nil, err
				}
				fd.Set(reflect.ValueOf(v))
			}
		}

	}
	return voteConfig, nil
}

// VoteConfigParams 解析返回[][2]string, 0:key, 1:value
func VoteConfigParams(s kingpin.Settings) (target *VoteConfigKvs) {
	target = (*VoteConfigKvs)(new([][2]string))
	s.SetValue((*VoteConfigKvs)(target))
	return target
}
