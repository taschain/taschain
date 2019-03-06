package monitor

import (
	"time"
	"os"
	"github.com/codeskyblue/go-sh"
	"fmt"
	"regexp"
	"strings"
	"strconv"
)

/*
**  Creator: pxf
**  Date: 2019/3/6 下午2:22
**  Description: 
*/
var spaceRe, _  = regexp.Compile("\\s+")

const (
	NtypeVerifier = 1
	NtypeProposal = 2
)

type NodeInfo struct {
	Type int
	VrfThreshold float64
	PStake 		uint64
	BlockHeight uint64
	GroupHeight uint64
	TxPoolCount int
}

type NodeResStat struct {
	Cpu float64
	Mem float64
	RcvBps float64
	TxBps  float64

	cmTicker *time.Ticker
	flowTicker *time.Ticker
}

func initNodeResStat() *NodeResStat {
	ns := &NodeResStat{
		cmTicker: time.NewTicker(time.Second*3),
		flowTicker: time.NewTicker(time.Second*5),
	}
	go ns.startStatLoop()
	return ns
}

func (ns *NodeResStat) startStatLoop()  {
	for {
		select {
		case <-ns.cmTicker.C:
			ns.statCpuAndMem()
		case <-ns.flowTicker.C:
			ns.statFlow()
		}
	}
}

func (s *NodeResStat) statCpuAndMem() {
	sess := sh.NewSession()
	sess.ShowCMD = true
	bs, err := sess.Command("top", "-b", "-n 1", fmt.Sprintf("-p %v", os.Getpid())).Command("grep", "gtas").Output()
	var ret string
	if bs != nil {
		ret = string(bs)
	}
	fmt.Printf("exe cmd:%v %v\n", ret, err)
	if err == nil {
		line := spaceRe.ReplaceAllString(string(bs), ",")
		arrs := strings.Split(line, ",")
		if len(arrs) < 10 {
			return
		}
		var cpu, mem float64
		cpu, _ = strconv.ParseFloat(arrs[8], 64)
		mems := arrs[5]
		if mems[len(mems)-1:] == "g" {
			f,_ := strconv.ParseFloat(mems[:len(mems)-1], 64)
			mem = f*1000
		} else {
			f,_ := strconv.ParseFloat(mems, 64)
			mem = f/1000
		}
		s.Cpu = cpu
		s.Mem = mem
		fmt.Printf("cpu %v, mem %v\n", cpu, mem)
	} else {

	}
	return
}

func (s *NodeResStat) statFlow() {
	sess := sh.NewSession()
	sess.ShowCMD = true
	bs, err := sess.Command("sar", "-n DEV", "1", "2", fmt.Sprintf("-p %v", os.Getpid())).Command("grep", "eth").Output()
	var ret string
	if bs != nil {
		ret = string(bs)
	}
	fmt.Printf("exe cmd:%v %v\n", ret, err)
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(bs)), "\n")
		if len(lines) < 1 {
			return
		}
		line := spaceRe.ReplaceAllString(lines[len(lines)-1], ",")
		arrs := strings.Split(line, ",")
		if len(arrs) < 8 {
			return
		}
		s.RcvBps, _ = strconv.ParseFloat(arrs[4], 64)
		s.TxBps, _ = strconv.ParseFloat(arrs[5], 64)
		fmt.Printf("rcv %v, tx %v\n", s.RcvBps, s.TxBps)
	} else {
		fmt.Printf("exe cmd err:%v\n", err)
	}
	return
}