package monitor

import (
	"common"
	"github.com/gohouse/gorose"
	"fmt"
	"time"
	"sync"
	"log"
	"sync/atomic"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2019/2/13 下午4:58
**  Description: 
*/

type MonitorService struct {
	enable bool
	cfg 	*gorose.DbConfigSingle
	queue  []*LogEntry
	lastSend time.Time
	nodeId string
	status int32
	mu 	sync.Mutex

	resStat *NodeResStat
	nodeInfo *NodeInfo
	lastUpdate time.Time

	internalNodeIds map[string]bool
}

const (
	LogTypeProposal = 1
	LogTypeBlockBroadcast = 2
	LogTypeBonusBroadcast = 3
	LogTypeCreateGroup = 4
	LogTypeCreateGroupSignTimeout = 5
	LogTypeInitGroupRevPieceTimeout = 6
	LogTypeGroupRecoverFromResponse = 7
)

const TableName = "logs"

type LogEntry struct {
	LogType int
	Operator string
	OperTime time.Time
	Height uint64
	Hash  string
	PreHash string
	Proposer string
	Verifier string
	Ext 	string
}

func (le *LogEntry) toMap() map[string]interface{} {
    m := make(map[string]interface{})
    m["LogType"] = le.LogType
    m["Operator"] = le.Operator
    m["OperTime"] = le.OperTime
    m["Height"] = le.Height
    m["Hash"] = le.Hash
    m["PreHash"] = le.PreHash
    m["Proposer"] = le.Proposer
    m["Verifier"] = le.Verifier
    m["Ext"] = le.Ext
    return m
}

var Instance = &MonitorService{}

func InitLogService(nodeId string) {
	Instance = &MonitorService{
		nodeId:   nodeId,
		queue:    make([]*LogEntry, 0),
		lastSend: time.Now(),
		enable:   true,
		resStat:  initNodeResStat(),
	}
	rHost := common.GlobalConf.GetString("gtas", "log_db_host", "120.78.127.246")
	rPort := common.GlobalConf.GetInt("gtas", "log_db_port", 3806)
	rDB := common.GlobalConf.GetString("gtas", "log_db_db", "taschain2")
	rUser := common.GlobalConf.GetString("gtas", "log_db_user", "root")
	rPass := common.GlobalConf.GetString("gtas", "log_db_password", "TASchain1003")
	Instance.cfg = &gorose.DbConfigSingle{
		Driver:          "mysql", // 驱动: mysql/sqlite/oracle/mssql/postgres
		EnableQueryLog:  false,    // 是否开启sql日志
		SetMaxOpenConns: 0,       // (连接池)最大打开的连接数，默认值为0表示不限制
		SetMaxIdleConns: 0,       // (连接池)闲置的连接数
		Prefix:          "",      // 表前缀
		Dsn:             fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8&parseTime=true", rUser, rPass, rHost, rPort, rDB), // 数据库链接username:password@protocol(address)/dbname?param=value
	}

	Instance.loadInternalNodesIds()
}



func (ms *MonitorService) saveLogs(logs []*LogEntry) {
	var err error
	defer func() {
		if err != nil {
			common.DefaultLogger.Errorf("save logs fail, err=%v, size %v", err, len(logs))
		} else {
			common.DefaultLogger.Infof("save logs success, size %v", len(logs))
		}
		ms.lastSend = time.Now()
		atomic.StoreInt32(&ms.status, 0)
	}()
	if !atomic.CompareAndSwapInt32(&ms.status, 0, 1) {
		return
	}

	connection, err := gorose.Open(ms.cfg)
	if err != nil {
		return
	}
	if connection == nil {
		err = fmt.Errorf("nil connection")
		return
	}
	defer connection.Close()

	sess := connection.NewSession()

	dm := make([]map[string]interface{}, 0)
	for _, log := range logs {
		dm = append(dm, log.toMap())
	}
	_, err = sess.Table(TableName).Data(dm).Insert()
}

func (ms *MonitorService) doAddLog(log *LogEntry)  {
	if !ms.MonitorEnable() {
		return
	}
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.queue = append(ms.queue, log)
	if len(ms.queue) >= 5 || time.Since(ms.lastSend).Seconds() > 15 {
		//go ms.saveLogs(ms.queue)
		ms.queue = make([]*LogEntry, 0)
	}
}

func (ms *MonitorService) AddLog(log *LogEntry) {
    log.Operator = ms.nodeId
    log.OperTime = time.Now()
    ms.doAddLog(log)
}

func (ms *MonitorService) MonitorEnable() bool {
    return ms.enable && ms.cfg != nil && ms.cfg.Dsn != ""
}

func (ms *MonitorService) loadInternalNodesIds() {
	connection, err := gorose.Open(ms.cfg)
	if err != nil {
		return
	}
	if connection == nil {
		err = fmt.Errorf("nil connection")
		return
	}
	defer connection.Close()

	sess := connection.NewSession()
	ret, err := sess.Table("tas_node_ids").Limit(1000).Get()
	 m := make(map[string]bool)

	 ids := make([]string, 0)
	if ret != nil {
		for _, d := range ret {
			id := d["minerId"].(string)
			m[id] = true
			ids = append(ids, id)
		}
	}
	ms.internalNodeIds = m

	log.Println("load internal nodes ", ids)
}

func (ms *MonitorService) AddLogIfNotInternalNodes(log *LogEntry)  {
	if _, ok := ms.internalNodeIds[log.Proposer]; !ok {
		ms.AddLog(log)
		common.DefaultLogger.Infof("addlog of not internal nodes %v", log.Proposer)
	}
}

func (ms *MonitorService) IsFirstNInternalNodesInGroup(mems []groupsig.ID, n int) bool {
	cnt := 0
	for _, mem := range mems {
		if _, ok := ms.internalNodeIds[mem.GetHexString()]; ok {
			cnt++
			if cnt >= n {
				break
			}
			if mem.GetHexString() == ms.nodeId {
				return true
			}
		}
	}
	return false
}

func (ms *MonitorService) UpdateNodeInfo(ni *NodeInfo)  {
	if !ms.MonitorEnable() {
		return
	}

    ms.nodeInfo = ni
	if time.Since(ms.lastUpdate).Seconds() > 2 {
		ms.lastUpdate = time.Now()
		connection, err := gorose.Open(ms.cfg)
		if err != nil {
			return
		}
		if connection == nil {
			err = fmt.Errorf("nil connection")
			return
		}
		defer connection.Close()

		sess := connection.NewSession()
		dm := make(map[string]interface{})
		dm["MinerId"] = ms.nodeId
		dm["NType"] = ms.nodeInfo.Type
		dm["VrfThreshold"] = ms.nodeInfo.VrfThreshold
		dm["PStake"] = ms.nodeInfo.PStake
		dm["BlockHeight"] = ms.nodeInfo.BlockHeight
		dm["GroupHeight"] = ms.nodeInfo.GroupHeight
		dm["TxPoolCount"] = ms.nodeInfo.TxPoolCount
		dm["Cpu"] = ms.resStat.Cpu
		dm["Mem"] = ms.resStat.Mem
		dm["RcvBps"] = ms.resStat.RcvBps
		dm["TxBps"] = ms.resStat.TxBps
		dm["UpdateTime"] = time.Now()

		affet, err := sess.Table("nodes").Data(dm).Update()
		if err == nil {
			if affet <= 0 {
				sess.Table("nodes").Data(dm).Insert()
			}
			fmt.Println("update nodes success, sql=%v", sess.LastSql)
		} else {
			fmt.Println("update nodes fail, sql=%v", sess.LastSql)
		}
	}
}