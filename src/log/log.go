package log

import (
	log "github.com/cihub/seelog"
	"fmt"
)

func InitLog(fileName string) {
	logger, e := log.LoggerFromConfigAsFile(fileName)
	if e != nil {
		fmt.Println(e)
	}
	log.ReplaceLogger(logger)
}

func InitLogByConfig(config string) {
	logger, e := log.LoggerFromConfigAsBytes([]byte(config))
	if e != nil {
		fmt.Println(e)
	}
	log.ReplaceLogger(logger)
}
