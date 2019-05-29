package logical

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/taslog"
)

/*
**  Creator: pxf
**  Date: 2018/7/27 下午2:43
**  Description:
 */

const NormalFailed int = -1
const NormalSuccess int = 0

//共识名词定义：
//铸块组：轮到铸当前高度块的组。
//铸块时间窗口：给该组完成当前高度铸块的时间。如无法完成，则当前高度为空块，由上个铸块的哈希的哈希决定铸下一个块的组。
//铸块槽：组内某个时间窗口的出块和验证。一个铸块槽如果能走到最后，则会成功产出该组的一个铸块。一个铸块组会有多个铸块槽。
//铸块槽权重：组内第一个铸块槽权重最高=1，第二个=2，第三个=4...。铸块槽权重<=0为无效。权重在>0的情况下，越小越好（有特权）。
//出块时间窗口：一个铸块槽的完成时间。如当前铸块槽在出块时间窗口内无法组内达成共识（上链，组外广播），则当前槽的King的组内排位后继成员成为下一个铸块槽的King，启动下一个铸块槽。
//出块人（King）：铸块槽内的出块节点，一个铸块槽只有一个出块人。
//见证人（Witness）：铸块槽内的验证节点，一个铸块槽有多个见证人。见证人包括组内除了出块人之外的所有成员。
//出块消息：在一个铸块槽内，出块人广播到组内的出块消息（类似比特币的铸块）。
//验块消息：在一个铸块槽内，见证人广播到组内的验证消息（对出块人出块消息的验证）。出块人的出块消息同时也是他的验块消息。
//合法铸块：就某个铸块槽组内达成一致，完成上链和组外广播。
//最终铸块：权重=1的铸块槽完成合法铸块；或当期高度铸块时间窗口内的组内权重最小的合法铸块。
//插槽替换规则（插槽可以容纳铸块槽，如插槽容量=5，则同时最多能容纳5个铸块槽）：
//1. 每过一个铸块槽时间窗口，如组内无法对上个铸块槽达成共识（完成组签名，向组外宣布铸块完成），则允许新启动一个铸块槽。新槽的King为上一个槽King的组内排位后继。
//2. 一个铸块高度在同一时间内可能会有多个铸块槽同时运行，如插槽未满，则所有满足规则1的出块消息或验块消息都允许新生成一个铸块槽。
//3. 如插槽已满，则时间窗口更早的铸块槽替换时间窗口较晚的铸块槽。
//4. 如某个铸块槽已经完成该高度的铸块（上链，组外广播），则只允许时间窗口更早的铸块槽更新该高度的铸块（上链，组外广播）。
//组内第一个KING的QN值=0。
/*
bls曲线使用情况：
使用：CurveFP382_1曲线，初始化参数枚举值=1.
ID长度（即地址）：48*8=384位。底层同私钥结构。
私钥长度：48*8=384位。底层结构Fr。
公钥长度：96*8=768位。底层结构G2。
签名长度：48*8=384位。底层结构G1。
*/

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
