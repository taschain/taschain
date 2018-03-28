package core

import (
	"time"
	
	"common"
)

type Group struct {
	address   common.Address
	members   []Member
	pubKey    common.Hash256
	gmtCreate time.Time
	parent    common.Address
	signature common.Hash256
	status    int8
}
