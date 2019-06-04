package cli

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/mediator"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware/types"
)

func convertTransaction(tx *types.Transaction) *Transaction {
	trans := &Transaction{
		Hash:          tx.Hash,
		Source:        tx.Source,
		Target:        tx.Target,
		Type:          tx.Type,
		GasLimit:      tx.GasLimit,
		GasPrice:      tx.GasPrice,
		Data:          tx.Data,
		ExtraData:     tx.ExtraData,
		ExtraDataType: tx.ExtraDataType,
		Nonce:         tx.Nonce,
		Value:         common.RA2TAS(tx.Value),
	}
	return trans
}

func convertBlockHeader(b *types.Block) *Block {
	bh := b.Header
	block := &Block{
		Height:  bh.Height,
		Hash:    bh.Hash,
		PreHash: bh.PreHash,
		CurTime: bh.CurTime.Local(),
		PreTime: bh.PreTime().Local(),
		Castor:  groupsig.DeserializeID(bh.Castor),
		GroupID: groupsig.DeserializeID(bh.GroupID),
		Prove:   common.ToHex(bh.ProveValue),
		TotalQN: bh.TotalQN,
		TxNum:   uint64(len(b.Transactions)),
		//Qn: mediator.Proc.CalcBlockHeaderQN(bh),
		StateRoot:   bh.StateTree,
		TxRoot:      bh.TxTree,
		ReceiptRoot: bh.ReceiptTree,
		Random:      common.ToHex(bh.Random),
	}
	return block
}

func convertBonusTransaction(tx *types.Transaction) *BonusTransaction {
	if tx.Type != types.TransactionTypeBonus {
		return nil
	}
	gid, ids, bhash, value := mediator.Proc.MainChain.GetBonusManager().ParseBonusTransaction(tx)

	targets := make([]groupsig.ID, len(ids))
	for i, id := range ids {
		targets[i] = groupsig.DeserializeID(id)
	}
	return &BonusTransaction{
		Hash:      tx.Hash,
		BlockHash: bhash,
		GroupID:   groupsig.DeserializeID(gid),
		TargetIDs: targets,
		Value:     value,
	}
}

func genMinerBalance(id groupsig.ID, bh *types.BlockHeader) *MinerBonusBalance {
	mb := &MinerBonusBalance{
		ID: id,
	}
	db, err := mediator.Proc.MainChain.GetAccountDBByHash(bh.Hash)
	if err != nil {
		common.DefaultLogger.Errorf("GetAccountDBByHash err %v, hash %v", err, bh.Hash)
		return mb
	}
	mb.CurrBalance = db.GetBalance(id.ToAddress())
	preDB, err := mediator.Proc.MainChain.GetAccountDBByHash(bh.PreHash)
	if err != nil {
		common.DefaultLogger.Errorf("GetAccountDBByHash err %v hash %v", err, bh.PreHash)
		return mb
	}
	mb.PreBalance = preDB.GetBalance(id.ToAddress())
	return mb
}

func IDFromSign(sign string) []byte {
	return []byte{}
}

func sendTransaction(trans *types.Transaction) error {
	if trans.Sign == nil {
		return fmt.Errorf("transaction sign is empty")
	}

	if ok, err := core.BlockChainImpl.GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		return err
	}
	return nil
}
