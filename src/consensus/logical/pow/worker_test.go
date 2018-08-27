package pow

import (
	"testing"
	"consensus/groupsig"
	"common"
	"time"
	"consensus/model"
	"sync/atomic"
	"runtime"
)

/*
**  Creator: pxf
**  Date: 2018/8/23 下午2:32
**  Description: 
*/

func TestWorker(t *testing.T) {
	runtime.GOMAXPROCS(2)
	groupsig.Init(1)

	gid := groupsig.NewIDFromInt(0)
	uid := groupsig.NewIDFromInt(1)
	worker := NewPowWorker(nil, model.NewGroupMinerID(*gid, *uid))


	worker.Prepare(common.Hash{}, 0, time.Now(),  3)
	worker.powStatus = STATUS_RUNNING
	go func() {
		time.Sleep(10*time.Second)
		t.Log("set stop...")
		atomic.CompareAndSwapInt32(&worker.powStatus, STATUS_RUNNING, STATUS_STOP)
	}()
	go worker.run()

	t.Log("start loop...")
	for {
		select {
		case <- worker.stopCh:
			t.Log("stopped..")
			return
		case cmd := <-worker.CmdCh:
			switch cmd {
			case CMD_POW_CONFIRM:

			case CMD_POW_RESULT:
				t.Log("nonce",  worker.Nonce)
			}

		}

	}
}