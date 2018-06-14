package types

import (
	"common"
	"encoding/json"
	"time"
)

type Transaction struct {
	Data   []byte
	Value  uint64
	Nonce  uint64
	Source *common.Address
	Target *common.Address

	GasLimit uint64
	GasPrice uint64
	Hash     common.Hash

	ExtraData     []byte
	ExtraDataType int32
}

func (tx *Transaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}

	blockByte, _ := json.Marshal(tx)
	return common.BytesToHash(common.Sha256(blockByte))
}

type Transactions []*Transaction

func (c Transactions) Len() int {
	return len(c)
}
func (c Transactions) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c Transactions) Less(i, j int) bool {
	return c[i].Nonce < c[j].Nonce
}


//区块头结构
type BlockHeader struct {
	Hash         common.Hash   // 本块的hash，to do : 是对哪些数据的哈希
	Height       uint64        // 本块的高度
	PreHash      common.Hash   //上一块哈希
	PreTime      time.Time     //上一块铸块时间
	QueueNumber  uint64        //轮转序号
	TotalQN      uint64        //整条链的QN
	CurTime      time.Time     //当前铸块时间
	Castor       []byte        //出块人ID
	GroupId      []byte        //组ID，groupsig.ID的二进制表示
	Signature    []byte        // 组签名
	Nonce        uint64        //盐
	Transactions []common.Hash // 交易集哈希列表
	TxTree       common.Hash   // 交易默克尔树根hash
	ReceiptTree  common.Hash
	StateTree    common.Hash
	EvictedTxs   []common.Hash
	ExtraData    []byte
}

func (bh *BlockHeader) GenHash() common.Hash {
	sign := bh.Signature
	hash := bh.Hash

	bh.Signature = []byte{}
	bh.Hash = common.Hash{}
	blockByte, _ := json.Marshal(bh)
	result := common.BytesToHash(common.Sha256(blockByte))
	bh.Signature = sign
	bh.Hash = hash
	return result
}

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}


type Member struct {
	Id     []byte
	PubKey []byte
}
type Group struct {
	Id        []byte
	Members   []Member
	PubKey    []byte
	Parent    []byte //父亲组 的组ID
	Dummy     []byte
	Signature []byte
	BeginHeight uint64                 //组开始参与铸块的高度
}

const genesis  = `{"Id":null,"Members":[{"Id":"MnlyUmYydHZUelMxdDFVRWJzNThSWTM3cjVGTDlDbnoxZHFEeFRUckE3ZlBtUQAA","PubKey":"137/lYJlU7eAAMLc2P3g7lIUYL+sxnmTqLz7t1v1iWeB9cNkVHErFcda4LHC6wkH/v2XO+bDmVXcBwSU4Cv2YpOiYcb29lTT7mVzKcWQKBo3oInwR3KnYC6BUdOZPiOC"},{"Id":"NWlQbkNVUmR6VTNZczNSV2pZb01KZlhKU2pXazJ0TkxCa0o4ZzZLYWhjN1RtUQAA","PubKey":"vKvF2jTHXd8A0zDlk6/dvmsIWlXrOo/EREDdel3fztqnwfDB9PqbF7sjjtWbWJAgN7qvHe8//XV6LTkWS/n0ljd5jd/pwL3g8tFed221/5bBwOXFUo8Nk3wNPENzaCSD"},{"Id":"OU03dllYTE5HY0ZtRFRiM3A1cXJYZ0xuQXZ4Y3I1ZExOM3Q3dEJRbU5HQmNtUQAA","PubKey":"j/zj1tTCwNcOoJmc7Y4ngjeRoKmFBV1woJ+eHarveStJuGf8wFBeY/gCPLYjmcUMUXKdO1eNmfe2C9mzldGt34CZlGhZJ7sJvVhZZceZ8lrYEL0jMEojfDZkk65NgtEb"},{"Id":"d2s0RUNvb0RDMlA0UFRaaVI1OTJOVFRyeUZCem5UcEY3NnNyUng0eEE1b05tUQAA","PubKey":"5/HH7Kub2FRBXVuz6qLn1rT0sBfijzEbuFMZKXCANSBZLDMe5KVjph6AhwFRZZwgusqDux6vDU7wGvpNon5Cw6Z+Pcbj5vC0L+fseGUd3dxXo2Ya9KtpB4WX3FC+otMg"},{"Id":"TTFtR3E2emUyV2UyNVgzS1B3VTNrakFGRTlqMXFwN25GcDZkZ0toNlBVa1FtUQAA","PubKey":"owYJK49AD1uPS4FmMu0ZJT168gmgKcAJPQ11+/fMoM9sdNm8GUylBe8GvlU5TrgIiVFAmJuPDusMrJkxHUUbB18qfAtgPBNhYbK/6/4efym7vpFw+EW1XT84ELCj50MS"}],"PubKey":"1Mp/guNJtXd4o4jVG0tJhrkSthHrJ3RNWV6ksP8yo6a+xph/ugocGbQ+oUGj2qwQQByqzSYIyWFbEJ2tq9BpIMrzQJpWfTz9F68XlPeM1Pjm6y3J3TPIjeBvwCLbxj4Q","Parent":"AAAA","Dummy":"Z2VuZXNpcyBncm91cCBkdW1teQ==","Signature":null}`

func GenesisGroup() *Group {
	var group *Group
	json.Unmarshal([]byte(genesis), &group) //never mistake
	return group
}