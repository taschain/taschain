package network

import (
	"github.com/gomodule/redigo/redis"
	"common"
	"taslog"
	"log"
	"fmt"
)

const (
	HMAP_KEY = "node_hash"
	SET_KEY = "node_set"
)

func getRedisConnection() (redis.Conn, error) {
	redisIp := common.GlobalConf.GetString("test", "redis_ip", "118.31.60.210")
	redisPort := common.GlobalConf.GetString("test", "redis_port", "6379")
	return redis.Dial("tcp", redisIp + ":" + redisPort)
}

func NodeOnline(id []byte, pubKey []byte) error{
	logger := taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("instance", "index", ""))
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return err
	}
	defer conn.Close()
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	conn.Do("hset", prefix+HMAP_KEY, id, pubKey)
	conn.Do("sadd", prefix+SET_KEY, id)
	logger.Info("node %s online, write to redis", string(id))
	return nil
}

func NodeOffline(id []byte) error {
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return err
	}
	defer conn.Close()
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	conn.Do("hdel", prefix+HMAP_KEY, id)
	conn.Do("srem", prefix+SET_KEY, id)
	return nil
}

func GetAllNodeIds() ([][]byte, error) {
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return nil, err
	}
	defer conn.Close()
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	r,err :=conn.Do("smembers", prefix+SET_KEY)
	return redis.ByteSlices(r, err)
}

func GetPubKeyById(id []byte) ([]byte, error) {
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return nil, err
	}
	defer conn.Close()
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	r,err := conn.Do("hget", prefix+HMAP_KEY, id)
	return redis.Bytes(r, err)
}

func GetPubKeyByIds(ids [][]byte) ([][]byte, error) {
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return nil, err
	}
	defer conn.Close()
	args := make([]interface{},len(ids) + 1)
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	args[0] = prefix+HMAP_KEY
	for i := 1; i <= len(ids); i++{
		args[i] = ids[i - 1]
	}
	r,err := conn.Do("hmget", args...)
	return redis.ByteSlices(r, err)
}

func CleanRedisData() {
	conn,err := getRedisConnection()
	if err != nil {
		log.Println("Connect to redis error", err)
		return
	}
	defer conn.Close()
	prefix := common.GlobalConf.GetString("test", "prefix", "")
	_, err = conn.Do("del", prefix+HMAP_KEY, prefix + SET_KEY)
	fmt.Printf("redis del:%s,%s",prefix+HMAP_KEY,prefix + SET_KEY)
	if err != nil {
		log.Printf("exec redis del command fail %v", err)
	}
}