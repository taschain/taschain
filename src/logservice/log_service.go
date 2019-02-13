package logservice

import (
	"common"
	"github.com/gohouse/gorose"
	"fmt"
	"time"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2019/2/13 下午4:58
**  Description: 
*/

type LogService struct {
	enable bool
	cfg 	*gorose.DbConfigSingle
	queue  []*LogEntry
	lastSend time.Time
	nodeId string
	mu 	sync.Mutex
}

const (
	LogTypeProposal = 1
	LogTypeBlockBroadcast = 2
	LogTypeBonusBroadcast = 3
	LogTypeCreateGroup = 4
	LogTypeAddOnChain = 5
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

var Instance = &LogService{}

func InitLogService(nodeId string) {
	Instance = &LogService{
		nodeId: nodeId,
		queue:  make([]*LogEntry, 0),
		lastSend: time.Now(),
		enable: true,
	}
	rHost := common.GlobalConf.GetString("gtas", "log_db_host", "120.78.127.246")
	rPort := common.GlobalConf.GetInt("gtas", "log_db_port", 3806)
	rDB := common.GlobalConf.GetString("gtas", "log_db_db", "taschain")
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
}


func (ls *LogService) saveLogs(logs []*LogEntry) {
	var err error
	defer func() {
		if err != nil {
			common.DefaultLogger.Errorf("save logs fail, err=%v, size %v", err, len(logs))
		} else {
			common.DefaultLogger.Infof("save logs success, size %v", len(logs))
		}
		ls.lastSend = time.Now()
	}()

	connection, err := gorose.Open(ls.cfg)
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

func (ls *LogService) AddLog(log *LogEntry) {
	if !ls.enable || ls.cfg == nil || ls.cfg.Dsn == "" {
		return
	}
    ls.mu.Lock()
    defer ls.mu.Unlock()
    log.Operator = ls.nodeId
    log.OperTime = time.Now()
    ls.queue = append(ls.queue, log)
	if len(ls.queue) >= 5 || time.Since(ls.lastSend).Seconds() > 15 {
		go ls.saveLogs(ls.queue)
		ls.queue = make([]*LogEntry, 0)
	}
}