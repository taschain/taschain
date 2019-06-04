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
var slowLogger taslog.Logger

func InitConsensus() {
	cc := common.GlobalConf.GetSectionManager(ConsensusConfSection)
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	stdLogger = taslog.GetLoggerByIndex(taslog.StdConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	groupLogger = taslog.GetLoggerByIndex(taslog.GroupLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	slowLogger = taslog.GetLoggerByIndex(taslog.SlowLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusConfManager = cc
	model.SlowLog = slowLogger
	model.InitParam(cc)
	return
}
