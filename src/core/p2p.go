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

import "middleware/types"

var (
	BlockChainConnectorImpl *BlockChainConnector
	GroupChainConnectorImpl *GroupChainConnector
)

type BlockChainConnector struct {
	chain BlockChain
}

type GroupChainConnector struct {
	chain *GroupChain
}

func InitCore(light bool, helper types.ConsensusHelper) error {
	light = false

	if nil == BlockChainImpl {
		var err error
		err = initBlockChain(helper)
		if nil != err {
			return err
		}
	}
	BlockChainConnectorImpl = &BlockChainConnector{
		chain: BlockChainImpl,
	}

	if nil == GroupChainImpl {
		err := initGroupChain(helper.GenerateGenesisInfo(), helper)
		if nil != err {
			return err
		}
	}
	GroupChainConnectorImpl = &GroupChainConnector{
		chain: GroupChainImpl,
	}

	PeerManager = initPeerManager()
	return nil
}