package taslog

import (
	"testing"
	"github.com/cihub/seelog"
)

// seelog wiki:https://github.com/cihub/seelog/wiki
var config = `<seelog minlevel="debug">
		<outputs formatid="testConfig">
			<rollingfile type="size" filename="/home/admin/tas/logs/test_config.log" maxsize="100000" maxrolls="3"/>
		</outputs>
		<formats>
			<format id="testConfig" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
		</formats>
	</seelog>`

func TestGetLogger(t *testing.T) {
	logger := GetLogger("conf/tas_log_test.xml")
	for i := 0; i < 3; i++ {
		logger.Debug("TestGetLogger debug output", i)
		logger.Info("TestGetLogger info output", i)
		logger.Warn("TestGetLogger warn output", i)
		logger.Error("TestGetLogger error output", i)
	}
	Close()
}

func TestGetLoggerByConfig(t *testing.T) {
	logger := GetLoggerByConfig(config)

	for i := 0; i < 3; i++ {
		logger.Debug("TestGetLoggerByConfigDefault debug output", i)
		logger.Info("TestGetLoggerByConfigDefault info output", i)
		logger.Warn("TestGetLoggerByConfigDefault warn output", i)
		logger.Error("TestGetLoggerByConfigDefault error output", i)
	}
	Close()
}

func TestMultiLogger(t *testing.T) {
	logger1 := GetLoggerByConfig(config)
	logger2 := GetLogger("conf/tas_log_test.xml")
	logger3 := GetLogger("")

	go func() {
		//default.log
		for i := 0; i < 3; i++ {
			logger3.Debug("TestMultiLogger go routine debug output")
			logger3.Info("TestMultiLogger go routine info output")
			logger3.Warn("TestMultiLogger go routine warn output")
			logger3.Error("TestMultiLogger go routine error output")
		}
	}()
	//test_config.log
	for i := 0; i < 3; i++ {
		logger1.Debug("TestMultiLogger test main debug output")
		logger1.Info("TestMultiLogger test main info output")
		logFunc(logger2, i)
		logger1.Warn("TestMultiLogger test main warn output")
		logger1.Error("TestMultiLogger test main error output")
	}
}

//test.log
func logFunc(logger seelog.LoggerInterface, i int) {
	logger.Debug("TestMultiLogger logFunc debug output", i)
	logger.Info("TestMultiLogger logFunc info output", i)
	logger.Warn("TestMultiLogger logFunc Warn output", i)
	logger.Error("TestMultiLogger logFunc error output", i)
}

func TestGetLoggerDefault(t *testing.T) {
	logger := GetLogger("")

	for i := 0; i < 3; i++ {
		logger.Debug("TestGetLoggerDefault debug output", i)
		logger.Info("TestGetLoggerDefault info output", i)
		logger.Warn("TestGetLoggerDefault warn output", i)
		logger.Error("TestGetLoggerDefault error output", i)
	}
	Close()
}

func TestGetLoggerByConfigDefault(t *testing.T) {
	logger := GetLoggerByConfig(``)

	for i := 0; i < 3; i++ {
		logger.Debugf("TestGetLoggerByConfigDefault debug output,%d", i)
		logger.Infof("TestGetLoggerByConfigDefault info output,%d", i)
		logger.Warnf("TestGetLoggerByConfigDefault warn output,%d", i)
		logger.Errorf("TestGetLoggerByConfigDefault error output,%d", i)
	}
	Close()
}

func TestInitTasLog(t *testing.T) {
	P2pLogger.Debug("debug test by p2p logger")
	P2pLogger.Flush()
}
