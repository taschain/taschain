package core

import (
	"time"
	"sync"
)

const (
	badPeersCleanInterval = time.Minute * 15
	evilMaxCount          = 3
)

var PeerManager = initPeerManager()

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

func (bpm *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	bpm.lock.RLock()
	defer bpm.lock.RUnlock()
	_, exit := bpm.badPeers[id]
	return exit
}

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
