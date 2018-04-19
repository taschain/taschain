package core

type Group struct {
	Id        []byte
	Members   []Member
	PubKey    []byte
	Parent    []byte //父亲组 的组ID
	Dummy     []byte
	Signature []byte
}
