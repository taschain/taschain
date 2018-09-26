package types

/*
**  Creator: pxf
**  Date: 2018/9/26 下午6:39
**  Description: 
*/

type GenesisInfo struct {
	Group Group
	VrfPKs map[int][]byte
}

type GenesisGenerator interface {
	Generate() *GenesisInfo
}