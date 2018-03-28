package vote

import "common"

/*
**  Creator: pxf
**  Date: 2018/3/27 下午8:06
**  Description: 
*/

type Code []byte

func DeployVoteTemplate(code Code) (*TemplateID) {
	//TODO:
	return nil
}

func UpdateVoteTemplate(id TemplateID, code Code) error {
	return nil
}

func GetVoteTemplate(id TemplateID) (Code, common.Hash, error) {
	return nil, common.Hash{}, nil
}