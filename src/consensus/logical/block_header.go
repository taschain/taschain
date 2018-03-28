package logical
   
import (
	"common"
	"consensus/groupsig"
	"time"
)

//区块头结构
type BlockHeader struct {
	//Trans_Hash_List：交易集哈希列表
	//Trans_Root_Hash：默克尔树根哈希
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块铸块时间
	BlockHeight uint64      //铸块高度
	QueueNumber uint32      //轮转序号
	CurTime     time.Time   //当前铸块时间
	Castor      groupsig.ID //铸块人(ID同时决定了铸块人的权重)
	NOnce       uint64      //盐
}

//数据签名结构
type SignData struct {
	DataHash   common.Hash        //哈希值
	DataSign   groupsig.Signature //签名
	SignMember groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	return groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
}