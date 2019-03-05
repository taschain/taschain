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
package core

import (
	"sync"
	"time"
)

const (
	badPeersCleanInterval = time.Minute * 3
	evilMaxCount          = 3
)

var PeerManager *peerManager

type peerManager struct {
	badPeerMeter map[string]uint64
	badPeers     map[string]time.Time
	cleaner      *time.Ticker

	lock sync.RWMutex
}

func initPeerManager() *peerManager {
	badPeerMeter := peerManager{badPeerMeter: make(map[string]uint64), badPeers: make(map[string]time.Time), cleaner: time.NewTicker(badPeersCleanInterval), lock: sync.RWMutex{}}
	go badPeerMeter.loop()
	return &badPeerMeter
}

//"黑名单"节点的记录功能：3次标识正式进入黑名单
func (bpm *peerManager) markEvil(id string) {
	if id == "" {
		return
	}
	bpm.lock.Lock()
	defer bpm.lock.Unlock()
	_, exit := bpm.badPeers[id]
	if exit {
		return
	}

	evilCount, meterExit := bpm.badPeerMeter[id]
	if !meterExit {
		bpm.badPeerMeter[id] = 1
		return
	} else {
		evilCount ++
		if evilCount > evilMaxCount {
			delete(bpm.badPeerMeter, id)
			bpm.badPeers[id] = time.Now()
			Logger.Debugf("[PeerManager]Add bad peer:%s", id)
		} else {
			bpm.badPeerMeter[id] = evilCount
			Logger.Debugf("[PeerManager]EvilCount:%s,%d", id, evilCount)
		}
	}
}

//是否在黑名单节点中
func (bpm *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	bpm.lock.RLock()
	defer bpm.lock.RUnlock()
	_, exit := bpm.badPeers[id]
	return exit
}

//黑名单节点释放功能：超过3分钟的黑名单节点直接洗白
func (bpm *peerManager) loop() {
	for {
		select {
		case <-bpm.cleaner.C:
			bpm.lock.Lock()
			Logger.Debugf("[PeerManager]Bad peers cleaner time up!")
			cleanIds := make([]string, 0, len(bpm.badPeers))
			for id, markTime := range bpm.badPeers {
				if time.Since(markTime) >= badPeersCleanInterval {
					cleanIds = append(cleanIds, id)
				}
			}
			for _, id := range cleanIds {
				delete(bpm.badPeers, id)
				Logger.Debugf("[PeerManager]Clean id:%s", id)
			}
			bpm.lock.Unlock()
		}
	}
}
