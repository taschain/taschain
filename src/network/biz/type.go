package biz

import (
	"time"
	"common"
)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash

	SourceId string

	RequestTime time.Time
}
