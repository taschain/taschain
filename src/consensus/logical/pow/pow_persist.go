package pow

import (
	"consensus/groupsig"
	"consensus/model"
	"common"
	"encoding/json"
)

/*
**  Creator: pxf
**  Date: 2018/8/14 下午12:01
**  Description: 
*/

type PreConfirmedPowResult struct {
	BlockHash common.Hash
	TotalLevel uint32
	Hash 	common.Hash
	GSign 	groupsig.Signature
	NonceSeq []model.MinerNonce
}

func (pcp *PreConfirmedPowResult) GetMinerNonce(id groupsig.ID) (int, *model.MinerNonce) {
	for i, mn := range pcp.NonceSeq {
		if mn.MinerID.IsEqual(id) {
			return i, &mn
		}
	}
	return -1, nil
}

func (pcp *PreConfirmedPowResult) CheckEqual(nonceSeq []model.MinerNonce) bool {
	tmp := model.ConsensusPowConfirmMessage{
		BaseHash: pcp.BlockHash,
		NonceSeq: nonceSeq,
	}
	return tmp.GenHash() == pcp.Hash
}


func (w *PowWorker) PersistConfirm() bool {
	confirm := w.GetConfirmed()

	nonceSeq := make([]model.MinerNonce, 0)
	for _, r := range confirm.results {
		nonceSeq = append(nonceSeq, model.MinerNonce{MinerID: r.minerID, Nonce: r.nonce})
	}

	pr := PreConfirmedPowResult{
		TotalLevel: confirm.totalLevel,
		GSign: *confirm.gSign,
		NonceSeq: nonceSeq,
		BlockHash: w.BH.Hash,
		Hash: confirm.resultHash,
	}

	bs, err := json.Marshal(pr)
	if err != nil {
		return false
	}
	w.storage.Put(w.BH.Hash.Bytes(), bs)
	return true
}

func (w *PowWorker) LoadConfirm() *PreConfirmedPowResult {
    pr := new(PreConfirmedPowResult)
    bs,err := w.storage.Get(w.BH.Hash.Bytes())
	if err != nil {
		return nil
	}
	if bs == nil || len(bs) == 0 {
		return nil
	}
    err = json.Unmarshal(bs, pr)
	if err != nil {
		return nil
	}
    return pr
}

