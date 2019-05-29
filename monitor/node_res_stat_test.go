package monitor

import (
	"fmt"
	"github.com/codeskyblue/go-sh"
	"os"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2019/3/6 下午2:32
**  Description:
 */

func TestExecuteCmd(t *testing.T) {
	sess := sh.NewSession()
	sess.ShowCMD = true
	bs, err := sess.Command("top", "-b", "-n 1", fmt.Sprintf("-p %v", os.Getpid())).Command("grep", "gtas").Output()
	t.Log(string(bs), err)
}
