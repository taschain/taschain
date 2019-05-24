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

package cli

import (
	"github.com/taschain/taschain/common"
	chandler "github.com/taschain/taschain/consensus/net/handler"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/core/net/handler"
	"github.com/taschain/taschain/core/net/sync"
	"github.com/taschain/taschain/network"
	"github.com/taschain/taschain/network/p2p"
	"log"

	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/logical"
	"github.com/taschain/taschain/middleware/types"
	"testing"
	"time"
)

func TestSeedGroupSync(t *testing.T) {
	mockSeed()

	log.Printf("seed group height:%d", core.GroupChainImpl.Height())
	mockGroup()
	log.Printf("seed group height:%d", core.GroupChainImpl.Height())
	sync.InitGroupSyncer()

	time.Sleep(1 * time.Minute)
}

func TestClientGroupSync(t *testing.T) {
	mockClient(t)

	log.Printf("client group height:%d", core.GroupChainImpl.Height())
	if core.GroupChainImpl.Height() != 0 {
		t.Fatal("client group height not excepted!")
	}
	sync.InitGroupSyncer()
	time.Sleep(time.Second * 20)
	log.Printf("client group height:%d", core.GroupChainImpl.Height())
	if core.GroupChainImpl.Height() != 1 {
		t.Fatal("client group height not excepted after sync!")
	}

	time.Sleep(30 * time.Minute)
}

func mockClient(t *testing.T) {
	common.GlobalConf = common.NewConfINIManager("tas_client_test.ini")
	err := core.InitCore()
	if err != nil {
		log.Printf("Init core error:%s", err.Error())
		return
	}

	p2p.SetChainHandler(new(handler.ChainHandler))
	p2p.SetConsensusHandler(new(chandler.ConsensusHandler))

	err1 := network.InitNetwork(&common.GlobalConf)
	if err1 != nil {
		log.Printf("Init network error:%s", err1.Error())
		return
	}
}

func mockSeed() {
	common.InitConf("tas_seed_test.ini")

	// 椭圆曲线初始化
	groupsig.Init(1)
	err := core.InitCore()
	if err != nil {
		log.Printf("Init core error:%s", err.Error())
		return
	}

	p2p.SetChainHandler(new(handler.ChainHandler))
	p2p.SetConsensusHandler(new(chandler.ConsensusHandler))

	err1 := network.InitNetwork(&common.GlobalConf)
	if err1 != nil {
		log.Printf("Init network error:%s", err1.Error())
		return
	}
}

func mockGroup() {
	miners := LoadPubKeyInfo("pubkeys1")
	gn := "gtas1"
	var gis logical.ConsensusGroupInitSummary
	//gis.ParentID = p.GetMinerID()

	var parentID groupsig.ID
	parentID.Deserialize([]byte("genesis group dummy"))
	gis.ParentID = parentID
	gis.DummyID = *groupsig.NewIDFromString(gn)
	gis.Authority = 777
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Now()
	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	gis.Members = uint64(3)
	gis.Extends = "Dummy"
	var grm logical.ConsensusGroupRawMessage
	grm.MEMS = make([]logical.PubKeyInfo, 3)
	copy(grm.MEMS[:], miners[:])
	grm.GI = gis

	members := make([]types.Member, 0)
	for _, miner := range miners {
		member := types.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
		members = append(members, member)
	}
	//此时组ID 跟组公钥是没有的
	group := types.Group{Members: members, Dummy: gis.DummyID.Serialize(), Parent: []byte("genesis group dummy")}
	err := core.GroupChainImpl.AddGroup(&group, nil, nil)
	if err != nil {
		log.Printf("Add dummy group error:%s\n", err.Error())
	}
}
