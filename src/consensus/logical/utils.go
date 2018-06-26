package logical

import (
	"consensus/groupsig"
	"time"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/25 下午4:14
**  Description: 
*/

func GetSecKeyPrefix(sk groupsig.Seckey) string {
	str := sk.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetPubKeyPrefix(pk groupsig.Pubkey) string {
	str := pk.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetIDPrefix(id groupsig.ID) string {
	str := id.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetHashPrefix(h common.Hash) string {
	str := h.Hex()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetSignPrefix(sign groupsig.Signature) string {
	str := sign.GetHexString()
	if len(str) >= 12 {
		link := str[0:6] + "-" + str[len(str)-6:]
		return link
	} else {
		return str[0:]
	}
}

func GetCastExpireTime(base time.Time, deltaHeight uint64) time.Time {
	return base.Add(time.Second * time.Duration(deltaHeight * uint64(MAX_GROUP_BLOCK_TIME)))
}
