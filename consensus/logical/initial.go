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

package logical

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/taslog"
)

const NormalFailed int = -1
const NormalSuccess int = 0

const ConsensusConfSection = "consensus"

var consensusLogger taslog.Logger
var stdLogger taslog.Logger
var groupLogger taslog.Logger
var consensusConfManager common.SectionConfManager

func InitConsensus() {
	cc := common.GlobalConf.GetSectionManager(ConsensusConfSection)
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	stdLogger = taslog.GetLoggerByIndex(taslog.StdConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	groupLogger = taslog.GetLoggerByIndex(taslog.GroupLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusConfManager = cc
	model.InitParam(cc)
	return
}
