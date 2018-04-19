package taslog

import (
	"github.com/cihub/seelog"
	"fmt"
	"strings"
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
	var logger seelog.LoggerInterface
	l, err := seelog.LoggerFromConfigAsFile(configFilePath)

	if err != nil {
		fmt.Printf("Get logger error! use defalut log!\n ")
		logger = GetLoggerByConfig(defaultConfig)
	} else {
		logger = l
	}
	register(logger)
	return logger
}

func GetLoggerByConfig(config string) seelog.LoggerInterface {

	if config == `` {
		config = defaultConfig
	}
	var logger seelog.LoggerInterface
	l, err := seelog.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Printf("Get logger error!use defalut log!\n")
		logger = GetLoggerByConfig(defaultConfig)
	} else {
		logger = l
	}
	register(logger)
	return logger
}


func GetLoggerByName(name string) seelog.LoggerInterface {
	var config string
	if name == "" {
		config = defaultConfig
	}else {
		fileName := name+".log"
		config = strings.Replace(defaultConfig,"default.log",fileName,1)
	}
	var logger seelog.LoggerInterface
	l, err := seelog.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Printf("Get logger error! use defalut log!\n ")
		logger = GetLoggerByConfig(defaultConfig)
	} else {
		logger = l
	}
	register(logger)
	return logger
}


func register(logger seelog.LoggerInterface) {
	if logger != nil {
		logManager = append(logManager, logger)
	}
}

var P2pLogger seelog.LoggerInterface

func init() {
	P2pLogger = GetLogger("conf/p2p.xml")

}
func Close() {
	for _, logger := range logManager {
		logger.Flush()
		logger.Close()
	}
}
