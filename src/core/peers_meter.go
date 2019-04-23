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
	"time"
	"github.com/hashicorp/golang-lru"
)

const (
	evilTimeoutMeterMax          = 3
	maxReqBlockCount			= 16
)

var PeerManager *peerManager

type peerMeter struct {
	id string
	timeoutMeter int
	lastHeard time.Time
	reqBlockCount int	//每次请求块的最大数量
}

func (m *peerMeter) isEvil() bool {
    return time.Since(m.lastHeard).Seconds() > 30 || m.timeoutMeter > evilTimeoutMeterMax
}

func (m *peerMeter) increaseTimeout()  {
    m.timeoutMeter++
}
func (m *peerMeter) decreaseTimeout()  {
	m.timeoutMeter--
	if m.timeoutMeter < 0 {
		m.timeoutMeter = 0
	}
}

func (m *peerMeter) updateReqCnt(increase bool)  {
	if !increase {
		m.reqBlockCount -= 4
		if m.reqBlockCount <= 0 {
			m.reqBlockCount = 1
		}
	} else {
		m.reqBlockCount += 1
		if m.reqBlockCount > maxReqBlockCount {
			m.reqBlockCount = maxReqBlockCount
		}
	}
}

func (m *peerMeter) updateLastHeard()  {
	m.lastHeard = time.Now()
}

type peerManager struct {
	//peerMeters map[string]*peerMeter
	peerMeters *lru.Cache
}

func initPeerManager() {
	cache, err := lru.New(100)
	if err != nil {
		panic(err)
	}
	badPeerMeter := peerManager{
		peerMeters: cache,
	}
	//go badPeerMeter.loop()
	PeerManager = &badPeerMeter
}

func (bpm *peerManager) getOrAddPeer(id string) *peerMeter {
	v, exit := bpm.peerMeters.Get(id)
	if !exit {
		v = &peerMeter{
			id:id,
			reqBlockCount: maxReqBlockCount,
		}
		if exit, _ = bpm.peerMeters.ContainsOrAdd(id, v); exit {
			v, _ = bpm.peerMeters.Get(id)
		}
	}
	return v.(*peerMeter)
}

func (bpm *peerManager) getPeerReqBlockCount(id string) int {
	pm := bpm.getOrAddPeer(id)
	if pm == nil {
		return maxReqBlockCount
	}
	return pm.reqBlockCount
}

func (bpm *peerManager) heardFromPeer(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.updateLastHeard()
	pm.decreaseTimeout()
}

func (bpm *peerManager) timeoutPeer(id string) {
	if id == "" {
		return
	}
	pm := bpm.getOrAddPeer(id)
	pm.increaseTimeout()
}

func (bpm *peerManager) isEvil(id string) bool {
	if id == "" {
		return false
	}
	pm := bpm.getOrAddPeer(id)
	return pm.isEvil()
}
func (bpm *peerManager) updateReqBlockCnt(id string, increase bool) {
	pm := bpm.getOrAddPeer(id)
	if pm == nil {
		return
	}
	pm.updateReqCnt(increase)
}
