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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware"
	"github.com/taschain/taschain/middleware/types"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2019/3/19 下午2:46
**  Description:
 */

func TestFullBlockChain_HasBlock(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/tas/deploy/daily/tas1.ini")
	types.InitMiddleware()
	middleware.InitMiddleware()
	initBlockChain(nil)

	hasBLock := BlockChainImpl.HasBlock(common.HexToHash("0x7f57774109cad543d9acfbcfa3630b30ca652d2310470341b78c62ee7463633b"))
	t.Log(hasBLock)
}

func TestFullBlockChain_QueryBlockFloor(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/test9/tas9.ini")
	middleware.InitMiddleware()
	initBlockChain(nil)

	chain := BlockChainImpl.(*FullBlockChain)

	fmt.Println("=====")
	bh := chain.queryBlockHeaderByHeight(0)
	fmt.Println(bh, bh.Hash.String())
	//top := gchain.latestBlock
	//t.Log(top.Height, top.Hash.String())
	//
	//for h := uint64(4460); h <= 4480; h++ {
	//	bh := gchain.queryBlockHeaderByHeightFloor(h)
	//	t.Log(bh.Height, bh.Hash.String())
	//}

	bh = chain.queryBlockHeaderByHeightFloor(0)
	fmt.Println(bh)
}
