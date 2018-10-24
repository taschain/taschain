package yunkuai

import (
	"storage/tasdb"
	"sync"
	"core/datasource"
	"middleware/notify"
	"middleware/types"
)

const Yunkuai_DataType = 75585
const Yunkuai_s = "yunkuai_s"
const Yunkuai_t = "yunkuai_t"
const yunkuai = "yunkuai"

var (
	instanceLock = sync.RWMutex{}
	instance     *YunKuaiProcessor
)

func GetYunKuaiProcessor() *YunKuaiProcessor {
	if nil == instance {
		instanceLock.Lock()
		defer instanceLock.Unlock()

		index, _ := datasource.NewDatabase(yunkuai)
		instance = &YunKuaiProcessor{
			index: index,
		}

	}

	return instance
}

type YunKuaiProcessor struct {
	index tasdb.Database
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
		if tx.ExtraDataType == Yunkuai_DataType {
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
