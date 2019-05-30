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

package mediator

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware"
	"log"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2018/10/24 下午5:31
**  Description:
 */
const CONF_PATH_PREFIX = `/Users/pxf/workspace/tas_develop/tas/deploy/daily`

func TestCore(t *testing.T) {
	middleware.InitMiddleware()
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")

	// block初始化
	err := core.InitCore(false, NewConsensusHelper(groupsig.ID{}))
	if err != nil {
		panic(err)
	}

	b := core.BlockChainImpl.QueryBlockByHeight(0)
	log.Println(b.Hash.ShortS(), b.GenHash().ShortS())

	b = core.BlockChainImpl.QueryBlockByHeight(0)
	log.Println(b.Hash.ShortS(), b.GenHash().ShortS())
}

func TestNewSelfMinerDO(t *testing.T) {
	i := 0
	for i < 5 {
		d := model.NewSelfMinerDO("123")
		t.Log(d.PK.GetHexString(), string(d.VrfPK), d.ID.GetHexString())
		i++
	}
}
