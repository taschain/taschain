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
