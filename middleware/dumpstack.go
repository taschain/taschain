package middleware

import (
	"os"
	"runtime"
	"time"
)

// how to use?
// kill -USR1 pid
// tail stack.log

const (
	timeFormat = "2006-01-02 15:04:05"
)

var (
	stdFile = "./stack.log"
)

func dumpStacks() {
	buf := make([]byte, 1638400)
	buf = buf[:runtime.Stack(buf, true)]
	writeStack(buf)
}

func writeStack(buf []byte) {
	fd, _ := os.OpenFile(stdFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)

	now := time.Now().Format(timeFormat)
	fd.WriteString("\n\n\n\n\n")
	fd.WriteString(now + " stdout:" + "\n\n")
	fd.Write(buf)
	fd.Close()
}
