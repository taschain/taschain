package logical

import (
	"fmt"
	"time"
	"middleware/types"
	"consensus/model"
	"consensus/groupsig"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/8 上午9:52
**  Description: 
*/
const TIMESTAMP_LAYOUT = "2006-01-02/15:04:05.000"


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

func logKeyword(mtype string, key string, sender string, format string, params ... interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,%v,#%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, key, sender, s)
}

func logStart(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	key := fmt.Sprintf("%v-%v", height, qn)
	logKeyword(mtype + "-begin", key, sender, format, params...)
}

func logEnd(mtype string, height uint64, qn uint64, sender string) {
	key := fmt.Sprintf("%v-%v", height, qn)
	logKeyword(mtype + "-end", key, sender, "%v", "")
}


func logHalfway(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	key := fmt.Sprintf("%v-%v", height, qn)
	logKeyword(mtype + "-half", key, sender, format, params...)
}

func GetCastExpireTime(base time.Time, deltaHeight uint64) time.Time {
	return base.Add(time.Second * time.Duration(deltaHeight * uint64(model.Param.MaxGroupCastTime)))
}

func ConvertStaticGroup2CoreGroup(sgi *StaticGroupInfo, isDummy bool) *types.Group {
	members := make([]types.Member, 0)
	for _, miner := range sgi.Members {
		member := types.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
		members = append(members, member)
	}

	if isDummy {
		return &types.Group{
			Dummy: sgi.GIS.DummyID.Serialize(),
			Members: members,
			Signature: sgi.Signature.Serialize(),
			Parent: sgi.ParentId.Serialize(),
			BeginHeight: sgi.BeginHeight,
			DismissHeight: sgi.DismissHeight,
			Authority: sgi.Authority,
			Name: sgi.Name,
			Extends: sgi.Extends,
		}
	} else {
		return &types.Group{
			Id: sgi.GroupID.Serialize(),
			Members: members,
			PubKey: sgi.GroupPK.Serialize(),
			Signature: sgi.Signature.Serialize(),
			Parent: sgi.ParentId.Serialize(),
			BeginHeight: sgi.BeginHeight,
			DismissHeight: sgi.DismissHeight,
			Authority: sgi.Authority,
			Name: sgi.Name,
			Extends: sgi.Extends,
		}
	}
}