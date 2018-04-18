package core

type Group struct {
	id   []byte
	members   []Member
	pubKey    []byte
	parent    []byte//父亲组 的组ID
	//yayuan   logical.ConsensusGroupInitSummary
	dummy      []byte
	signature  []byte
}
