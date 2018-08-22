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

package logical

import (
	"encoding/json"
	"consensus/groupsig"
	"log"
)

/*
**  Creator: pxf
**  Date: 2018/6/12 下午6:12
**  Description: 
*/

var STORE_PREFIX = "consensus_store"

func (p *Processor) saveJoinedGroup(jg *JoinedGroup) {
	buf, err := json.Marshal(jg)
	if err != nil {
		panic("Processor::Save Marshal failed ." + err.Error())
	}
	p.storage.Put(jg.GroupID.Serialize(), buf)
}


func (p *Processor) loadJoinedGroup(gid *groupsig.ID) *JoinedGroup {
	ret, err := p.storage.Get(gid.Serialize())
	if err != nil {
		log.Printf("loadJoinedGroup fail, err=%v\n", err.Error())
		return nil
	}
	if ret == nil {
		return nil
	}

	var jg = new(JoinedGroup)
	err = json.Unmarshal(ret, jg)
	if err != nil {
		log.Printf("loadJoinedGroup unmarshal fail, err=%v\n", err.Error())
		return nil
	}
	return jg
}

func (p *Processor) prepareMiner()  {
    rets := p.GroupChain.GetAllGroupID()
	if len(rets) == 0 {
		return
	}
	log.Printf("prepareMiner get groups from groupchain, len=%v\n", len(rets))
	for _, gidBytes := range rets {
		coreGroup := p.GroupChain.GetGroupById(gidBytes)
		if coreGroup == nil {
			panic("buildGlobalGroups getGroupById failed! gid=" + string(gidBytes))
		}
		log.Printf("coreGroup %+v, gid=%v\n", coreGroup, gidBytes)
		if coreGroup.Id == nil || len(coreGroup.Id) == 0 {
			continue
		}
		sgi := StaticGroupInfo{
			GroupID: *groupsig.DeserializeId(coreGroup.Id),
			GroupPK: *groupsig.DeserializePubkeyBytes(coreGroup.PubKey),
			BeginHeight: coreGroup.BeginHeight,
			Members: make([]PubKeyInfo, 0),
			MapCache: make(map[string]int),
		}
		for _, mem := range coreGroup.Members {
			pkInfo := &PubKeyInfo{ID: *groupsig.DeserializeId(mem.Id), PK: *groupsig.DeserializePubkeyBytes(mem.PubKey)}
			sgi.addMember(pkInfo)
		}
		if !p.gg.AddGroup(sgi) {
			continue
		}
		if sgi.MemExist(p.GetMinerID()) {
			gid := &sgi.GroupID
			jg := p.loadJoinedGroup(gid)
			if jg == nil {
				panic("cannot find joinedgroup infos! gid=" + GetIDPrefix(*gid))
			}
			p.addInnerGroup(jg, false)
			p.prepareForCast(*gid)
		}
	}
}

func (p *Processor) Ready() bool {
    return p.ready
}

func (p *Processor) getGroupSecret(gid groupsig.ID) *GroupSecret {
	if jg, ok := p.belongGroups[gid.GetHexString()]; ok {
		return &jg.GroupSec
	} else {
		return nil
	}
}