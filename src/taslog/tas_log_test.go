package taslog

import (
	"testing"
	"time"
)

// seelog wiki:https://github.com/cihub/seelog/wiki
var config = `<seelog minlevel="debug">
		<outputs formatid="testConfig">
			<rollingfile type="size" filename="./logs/test_config.log" maxsize="100000" maxrolls="3"/>
		</outputs>
		<formats>
			<format id="testConfig" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
		</formats>
	</seelog>`


var testConfig = `<seelog minlevel="debug">
		<outputs formatid="testConfig">
			<rollingfile type="size" filename="./logs/test.log" maxsize="100000" maxrolls="3"/>
		</outputs>
		<formats>
			<format id="testConfig" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
		</formats>
	</seelog>`

func TestGetLogger(t *testing.T) {
	logger := GetLogger(config)
	for i := 0; i < 3; i++ {
		logger.Debug("TestGetLogger debug output", i)
		logger.Info("TestGetLogger info output", i)
		logger.Warn("TestGetLogger warn output", i)
		logger.Error("TestGetLogger error output", i)
	}

	l := GetLogger(config)
	l.Debug("TestGetLogger debug output")
	Close()
}



func TestMultiLogger(t *testing.T) {
	logger1 := GetLogger(config)
	logger2 := GetLogger(testConfig)
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
	time.Sleep(1*time.Second)
	Close()
}

//test.log
func logFunc(logger Logger, i int) {
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


func TestGetLoggerByName(t *testing.T) {
	name := "littleBear"
	logger := GetLoggerByName(name)
	logger.Debug("TestGetLoggerByName logFunc debug output")
	logger.Info("TestGetLoggerByName logFunc info output")
	logger.Warn("TestGetLoggerByName logFunc Warn output")
	logger.Error("TestGetLoggerByName logFunc error output")

	//l := GetLoggerByName(name)
	//l.Debug("TestGetLoggerByName logFunc debug output")
	Close()
}



