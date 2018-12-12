package cli

import (
	"middleware/types"
	"consensus/groupsig"
	"consensus/mediator"
	"log"
)

/*
**  Creator: pxf
**  Date: 2018/10/16 下午3:05
**  Description: 
*/

func convertTransaction(tx *types.Transaction) *Transaction {
	trans := &Transaction{
		Hash: tx.Hash,
		Source: tx.Source,
		Target: tx.Target,
		Type: tx.Type,
		GasLimit: tx.GasLimit,
		GasPrice: tx.GasPrice,
		Data: tx.Data,
		ExtraData: tx.ExtraData,
		ExtraDataType: tx.ExtraDataType,
		Nonce: tx.Nonce,
		Value: tx.Value,
	}
	return trans
}

func convertBlockHeader(bh *types.BlockHeader) *Block {
	block := &Block{
		Height: bh.Height,
		Hash: bh.Hash,
		PreHash: bh.PreHash,
		CurTime: bh.CurTime,
		PreTime: bh.PreTime,
		Castor: groupsig.DeserializeId(bh.Castor),
		GroupID: groupsig.DeserializeId(bh.GroupId),
		Prove: bh.ProveValue,
		Txs: bh.Transactions,
		TotalQN: bh.TotalQN,
		//Qn: mediator.Proc.CalcBlockHeaderQN(bh),
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
		targets[i] = groupsig.DeserializeId(id)
	}
	return &BonusTransaction{
		Hash: tx.Hash,
		BlockHash: bhash,
		GroupID: groupsig.DeserializeId(gid),
		TargetIDs: targets,
		Value: value,
	}
}

func genMinerBalance(id groupsig.ID, bh *types.BlockHeader) *MinerBonusBalance {
	mb :=  &MinerBonusBalance{
		ID: id,
	}
	db, err := mediator.Proc.MainChain.GetAccountDBByHash(bh.Hash)
	if err != nil {
		log.Printf("GetAccountDBByHash err %v, hash %v", err, bh.Hash)
		return mb
	}
	mb.CurrBalance = db.GetBalance(id.ToAddress())
	preDB, err := mediator.Proc.MainChain.GetAccountDBByHash(bh.PreHash)
	if err != nil {
		log.Printf("GetAccountDBByHash err %v hash %v", err, bh.PreHash)
		return mb
	}
	mb.PreBalance = preDB.GetBalance(id.ToAddress())
	return mb
}