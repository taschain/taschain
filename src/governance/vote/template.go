package vote

import (
	"common"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午8:06
**  Description: 
*/

type ABI string	//todo ABI define

type VoteTemplate struct {
	code []byte
	hash common.Hash
	abi ABI
	gmtPublish time.Time
	author common.Address
	valid bool
}

func DeployVoteTemplate(template *VoteTemplate) (*TemplateID) {
	//TODO:
	return nil
}


func GetVoteTemplate(id TemplateID) (*VoteTemplate, error) {
	return nil, nil
}