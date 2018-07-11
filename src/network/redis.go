package network

import (
	"github.com/gomodule/redigo/redis"
	"common"
	"fmt"
	"network/p2p"
	"taslog"
)

const (
	HMAP_KEY = "node_hash"
	SET_KEY = "node_set"
)

func getRedisConnection() (redis.Conn, error) {
	redisIp := common.GlobalConf.GetString("test", "redis_ip", "127.0.0.1")
	redisPort := common.GlobalConf.GetString("test", "redis_port", "6379")
	return redis.Dial("tcp", redisIp + ":" + redisPort)
}

func NodeOnline(node *p2p.Node) error{
	logger := taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	conn,err := getRedisConnection()
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	defer conn.Close()

	conn.Do("hset", HMAP_KEY, node.Id, node.PublicKey.ToBytes())
	conn.Do("sadd", SET_KEY, node.Id)
	logger.Info("node %s online, write to redis", node.Id)
	return nil
}

func NodeOffline(id string) error {
	conn,err := getRedisConnection()
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return err
	}
	defer conn.Close()

	conn.Do("hdel", HMAP_KEY, id)
	conn.Do("srem", SET_KEY, id)
	return nil
}

func GetAllNodeIds() ([][]byte, error) {
	conn,err := getRedisConnection()
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return nil, err
	}
	defer conn.Close()
	r,err :=conn.Do("smembers", SET_KEY)
	return redis.ByteSlices(r, err)
}

func GetPubKeyById(id string) ([]byte, error) {
	conn,err := getRedisConnection()
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return nil, err
	}
	defer conn.Close()
	r,err := conn.Do("hget", HMAP_KEY, id)
	return redis.Bytes(r, err)
}