package mediator

import (
	"middleware"
	"common"
	"consensus/groupsig"
	"testing"
	"core"
	"log"
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