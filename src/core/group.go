package core

import (

	"consensus/logical"
)

type Group struct {
	id   []byte
	members   []Member
	pubKey    []byte
	parent    []byte//父亲组 的组ID
	yayuan logical.ConsensusGroupInitSummary
	signature  []byte
}
