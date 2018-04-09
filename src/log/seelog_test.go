package log

import (
	"testing"
	log "github.com/cihub/seelog"
)

//wiki: https://github.com/cihub/seelog/wiki
func TestInitLog(t *testing.T) {
	defer log.Flush()
	InitLog("seelog_test.xml")
	log.Warn("test for init log by file ")
}

func TestInitLogByConfig(t *testing.T) {
	defer log.Flush()

	config := `
<seelog minlevel="warn">
    <outputs formatid="test">
        <rollingfile type="size" filename="/home/admin/tas/logs/test.log" maxsize="100000" maxrolls="5"/>
    </outputs>
    <formats>
        <format id="test" format="%Date/%Time [%LEV] %Msg%n" />
    </formats>
</seelog>
`
	InitLogByConfig(config)
	log.Error("test for init log by config ")
}
