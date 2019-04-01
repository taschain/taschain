package core

import (
	"middleware/types"
	"bytes"
	"github.com/vmihailenco/msgpack"
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

func encodeBlockTransactions(b *types.Block) ([]byte, error) {
	bh := b.Header
	dataBuf := bytes.NewBuffer([]byte{})
	//先写交易数量
	dataBuf.Write(utility.UInt16ToByte(uint16(len(bh.Transactions))))
	//再写每个交易长度
	//最后写交易数据
	if len(bh.Transactions) > 0 {
		txBuf := bytes.NewBuffer([]byte{})
		for i, txHash := range bh.Transactions {
			tx := b.Transactions[i]
			if tx.Hash != txHash {
				panic("tx hash error")
			}
			txBytes, err := marshalTx(tx)
			if err != nil {
				return nil, err
			}

			dataBuf.Write(utility.UInt16ToByte(uint16(len(txBytes))))
			txBuf.Write(txBytes)

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

func decodeTransaction(bh *types.BlockHeader, txHash common.Hash, data []byte) (*types.Transaction, error) {
	var idx = -1
	for i, tx := range bh.Transactions {
		if tx == txHash {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("tx hash not exist in block")
	}

	txNum := utility.ByteToUInt16(data[:2])
	if int(txNum) != len(bh.Transactions) {
		return nil, fmt.Errorf("tx num not equal to header txs num: %v-%v", txNum, len(bh.Transactions))
	}
	if txNum == 0 {
		return nil, nil
	}

	lenBytes := data[2:2+int(txNum)*2]

	offset := 2+int(txNum)*2
	txLen := 0
	for i := 0; true; i++ {
		len := utility.ByteToUInt16(lenBytes[2*i:2*(i+1)])
		if i == idx {
			txLen = int(len)
			break
		} else {
			offset += int(len)
		}
	}

	txBytes := data[offset:offset+txLen]

	tx, err := unmarshalTx(txBytes)
	if err != nil {
		return nil, err
	}
	if tx.Hash != txHash {
		return nil, fmt.Errorf("tx hash err")
	}
	return tx, nil
}