package yunkuai

import (
	"storage/tasdb"
	"sync"

	"middleware/notify"
	"middleware/types"
	"bytes"
)

const Yunkuai_DataType = 75585
const Yunkuai_s = "yunkuai_s"
const Yunkuai_t = "yunkuai_t"
const yunkuai = "yunkuai"
const yunkuai_version = "yunkuai_version"

var (
	instanceLock = sync.RWMutex{}
	instance     *YunKuaiProcessor
)

func GetYunKuaiProcessor() *YunKuaiProcessor {
	if nil == instance {
		instanceLock.Lock()
		defer instanceLock.Unlock()

		index, _ := tasdb.NewDatabase(yunkuai)
		version, _ := tasdb.NewDatabase(yunkuai_version)
		instance = &YunKuaiProcessor{
			index:   index,
			version: version,
		}

	}

	return instance
}

type YunKuaiProcessor struct {
	index   tasdb.Database
	version tasdb.Database
}

func (y *YunKuaiProcessor) AfterBlockOnBlock(message notify.Message) {
	if nil == message {
		return
	}

	b := message.GetData().(types.Block)
	txs := b.Transactions
	if nil == txs || 0 == len(txs) {
		return
	}

	for _, tx := range txs {
		if tx.Type == types.TransactionYunkuai {
			// 保存索引
			y.index.Put(tx.Data, tx.Hash.Bytes())
		}
	}
}

func (y *YunKuaiProcessor) Contains(index string) bool {
	flag, _ := y.index.Has([]byte(index))

	return flag
}

func (y *YunKuaiProcessor) Get(index string) []byte {
	data, _ := y.index.Get([]byte(index))

	return data
}

func (y *YunKuaiProcessor) getVersion(index string) byte {
	data, e := y.version.Get([]byte(index))
	if e != nil || nil == data {
		return 0
	}

	return data[0]
}

func (y *YunKuaiProcessor) GenerateLastestKey(index string) string {
	version := y.getVersion(index)

	var buf bytes.Buffer
	buf.WriteString(index)
	buf.WriteByte(version)
	return buf.String()

}

func (y *YunKuaiProcessor) GenerateNewKey(index string) string {
	version := y.getVersion(index) + 1

	data := make([]byte, 1)
	data[0] = version
	y.version.Put([]byte(index), data)

	var buf bytes.Buffer
	buf.WriteString(index)
	buf.WriteByte(version)
	return buf.String()

}