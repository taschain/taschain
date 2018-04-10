package log

import (
	"github.com/cihub/seelog"
	"fmt"
)

const baseConfigFilePath = "conf/default.xml"

const defaultConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="/home/admin/tas/logs/default.log" maxsize="500000000" maxrolls="10"/>
						</outputs>
						<formats>
							<format id="default" format="%Date/%Time [%Level]  [%File:%Line] %Msg%n" />
						</formats>
					</seelog>`

var logManager = []seelog.LoggerInterface{}

func GetLogger(configFilePath string) seelog.LoggerInterface {

	if configFilePath == "" {
		configFilePath = baseConfigFilePath
	}
	logger, err := seelog.LoggerFromConfigAsFile(configFilePath)

	if err != nil {
		fmt.Print("Get logger error!")
		return nil
	}
	regiter(logger)
	return logger
}

func GetLoggerByConfig(config string) seelog.LoggerInterface {

	if config == `` {
		config = defaultConfig
	}
	logger, err := seelog.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Print("Get logger error!")
		return nil
	}
	regiter(logger)
	return logger
}

func regiter(logger seelog.LoggerInterface) {
	if logger != nil {
		logManager = append(logManager, logger)
	}
}

func Close() {
	for _, logger := range logManager {
		logger.Flush()
		logger.Close()
	}
}
