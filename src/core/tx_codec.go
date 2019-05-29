package core

import (
	"math"
	"middleware/types"
	"bytes"
	"github.com/vmihailenco/msgpack"
	"runtime"
	"sync"
	"utility"
	"fmt"
	"io"
	"common"
)

/*
**  Creator: pxf
**  Date: 2019/3/20 上午11:32
**  Description: 
*/

const codecVersion = 1

func marshalTx(tx *types.Transaction) ([]byte, error) {
	return msgpack.Marshal(tx)
}

func unmarshalTx(data []byte) (*types.Transaction, error) {
	var tx types.Transaction
	err := msgpack.Unmarshal(data, &tx)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func concurentMarshalTxs(txs []*types.Transaction) map[common.Hash]interface{} {
	taskNum := runtime.NumCPU()*2
	ret := make(map[common.Hash]interface{})
	mu := sync.Mutex{}
	step := int(math.Ceil(float64(len(txs))/float64(taskNum)))
	wg := sync.WaitGroup{}
	for begin := 0; begin < len(txs); {
		end := begin + step
		if end > len(txs) {
			end = len(txs)
		}
		sublist := txs[begin:end]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, tx := range sublist {
				txBytes, err := marshalTx(tx)
				var v interface{}
				if err != nil {
					v = err
				} else {
					v = txBytes
				}
				if v == nil {
					panic("nil value")
				}
				mu.Lock()
				ret[tx.Hash] = v
				mu.Unlock()
			}
		}()
		begin = end
	}
	wg.Wait()
	return ret
}

func encodeBlockTransactions(b *types.Block) ([]byte, error) {
	dataBuf := bytes.NewBuffer([]byte{})
	//先写版本号
	dataBuf.Write(utility.UInt16ToByte(uint16(codecVersion)))
	//先写交易数量
	dataBuf.Write(utility.UInt16ToByte(uint16(len(b.Transactions))))
	//再写每个交易长度
	//最后写交易数据
	if len(b.Transactions) > 0 {

		//txBytesMap := concurentMarshalTxs(b.Transactions)

		txBuf := bytes.NewBuffer([]byte{})
		for _, tx := range b.Transactions {
			txBytes, err := marshalTx(tx)
			if err != nil {
				return nil, err
			}
			dataBuf.Write(utility.UInt16ToByte(uint16(len(txBytes))))
			txBuf.Write(txBytes)
			//v := txBytesMap[tx.Hash]
			//switch v.(type) {
			//case []byte:
			//	txBytes := v.([]byte)
			//	dataBuf.Write(utility.UInt16ToByte(uint16(len(txBytes))))
			//	txBuf.Write(txBytes)
			//default:
			//	err := v.(error)
			//	fmt.Println(err)
			//	return nil, err
			//}


		}
		dataBuf.Write(txBuf.Bytes())
	}

	return dataBuf.Bytes(), nil
}

func decodeBlockTransactions(data []byte) ([]*types.Transaction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	txs := make([]*types.Transaction, 0)
	dataReader := bytes.NewReader(data)

	twoBytes := make([]byte, 2)
	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	version := utility.ByteToUInt16(twoBytes)
	if version != codecVersion {
		return nil, fmt.Errorf("version error")
	}

	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	txNum := utility.ByteToUInt16(twoBytes)
	if txNum == 0 {
		return txs, nil
	}

	lenBytes := make([]byte, txNum*2)
	_, err := io.ReadFull(dataReader, lenBytes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(txNum); i++ {
		txLen := utility.ByteToUInt16(lenBytes[2*i:2*(i+1)])
		txBytes := make([]byte, txLen)
		_, err := io.ReadFull(dataReader, txBytes)
		if err != nil {
			return nil, err
		}
		tx, err := unmarshalTx(txBytes)
		if err != nil {
			return nil,err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func decodeTransaction(idx int, txHash common.Hash, data []byte) (*types.Transaction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	dataReader := bytes.NewReader(data)

	twoBytes := make([]byte, 2)
	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	version := utility.ByteToUInt16(twoBytes)
	if version != codecVersion {
		return nil, fmt.Errorf("version error")
	}

	if _, err := io.ReadFull(dataReader, twoBytes); err != nil {
		return nil, err
	}
	txNum := utility.ByteToUInt16(twoBytes)
	if txNum == 0 {
		return nil, fmt.Errorf("txNum is zero")
	}

	lenBytes := make([]byte, txNum*2)
	_, err := io.ReadFull(dataReader, lenBytes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(txNum); i++ {
		txLen := utility.ByteToUInt16(lenBytes[2*i:2*(i+1)])
		txBytes := make([]byte, txLen)
		_, err := io.ReadFull(dataReader, txBytes)
		if err != nil {
			return nil, err
		}
		if idx == i {
			tx, err := unmarshalTx(txBytes)
			if err != nil {
				return nil,err
			}
			if tx.Hash != txHash {
				return nil, fmt.Errorf("tx hash err")
			}
			return tx ,nil
		}
	}
	return nil, fmt.Errorf("tx not found")
}