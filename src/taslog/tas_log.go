package taslog

import (
	"github.com/cihub/seelog"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/sha3"
)

var logManager = map[string]seelog.LoggerInterface{}

var lock sync.Mutex

func GetLogger(config string) Logger {
	if config == `` {
		config = DefaultConfig
	}
	key := getKey(config)

	lock.Lock()
	r := logManager[key]
	lock.Unlock()

	if r == nil {
		l := newLoggerByConfig(config)
		register(getKey(config), l)
		return &defaultLogger{logger: l}
	} else {
		return &defaultLogger{logger: r}
	}
}

func getKey(s string) string {
	hash := sha3.Sum256([]byte(s))
	return string(hash[:])
}

func newLoggerByConfig(config string) seelog.LoggerInterface {
	var logger seelog.LoggerInterface
	l, err := seelog.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Printf("Get logger error:%s\n", err.Error())
		panic(err)
	} else {
		logger = l
	}
	return logger
}

func GetLoggerByName(name string) Logger {
	key := getKey(name)
	lock.Lock()
	r := logManager[key]
	lock.Unlock()

	if r != nil {
		return &defaultLogger{logger: r}
	} else {
		var config string
		if name == "" {
			config = DefaultConfig
			return GetLogger(config)
		} else {
			fileName := name + ".log"
			config = strings.Replace(DefaultConfig, "default.log", fileName, 1)
			l := newLoggerByConfig(config)
			register(getKey(name), l)
			return &defaultLogger{logger: l}
		}
	}
}

func register(name string, logger seelog.LoggerInterface) {
	lock.Lock()
	defer lock.Unlock()
	if logger != nil {
		logManager[name] = logger
	}
}

func Close() {
	lock.Lock()
	defer lock.Unlock()
	for _, logger := range logManager {
		logger.Flush()
		logger.Close()
	}
}
