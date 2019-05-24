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

func voteConfigHelp() string {
	return `format like key=value, keys include: 
							TemplateId          string      //合约模板id
							PIndex              int         //投票参数索引
							PValue              interface{} //投票值
							Custom              bool        //'是否自定义投票合约', true时, pIndex pValue无效
							Desc                string      //描述
							DepositMin          uint64      //每个投票人最低缴纳保证金
							TotalDepositMin     uint64      //最低总保证金
							VoterCntMin         uint64      //最低参与投票人
							ApprovalDepositMin  uint64      //通过的最低保证金
							ApprovalVoterCntMin uint64      //通过的最低投票人
							DeadlineBlock       uint64      //投票截止的最高区块高度
							StatBlock           uint64      //唱票区块高度
							EffectBlock         uint64      //生效高度
							DepositGap          int         //缴纳保证金后到可以投票的间隔, 用区块高度差表示`
}
