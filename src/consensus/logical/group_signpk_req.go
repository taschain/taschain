package logical

import (
	"time"
	"consensus/groupsig"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2019/1/21 下午3:40
**  Description: 
*/

type signPKReqRecord struct {
	reqTime time.Time
	reqUid groupsig.ID
}

func (r *signPKReqRecord) reqTimeout() bool {
    return time.Now().After(r.reqTime.Add(60*time.Second))
}

var recordMap sync.Map //idHex -> signPKReqRecord

func addSignPkReq(id groupsig.ID) bool {
	r := &signPKReqRecord{
		reqTime: time.Now(),
		reqUid:  id,
	}
	_, load := recordMap.LoadOrStore(id.GetHexString(), r)
	return !load
}

func removeSignPkRecord(id groupsig.ID)  {
    recordMap.Delete(id.GetHexString())
}

func cleanSignPkReqRecord() {
	recordMap.Range(func(key, value interface{}) bool {
		r := value.(*signPKReqRecord)
		if r.reqTimeout() {
			recordMap.Delete(key)
		}
		return true
	})
}