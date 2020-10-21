package dbUtil

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/labstack/gommon/log"
	"os"
	"strings"
	"wallet-token/enums"
	"wallet-token/models"
	"wallet-vote/model"
	"wallet-vote/voteEnums"
)

var pool *redis.Pool
func init(){
	pool = &redis.Pool{
		MaxIdle:     500,
		MaxActive:   2000,
		IdleTimeout: 120,
		Dial: func() (redis.Conn, error) {
			coon, err := redis.Dial("tcp", "127.0.0.1:6379")
			if err!=nil{
				fmt.Println(err)
				os.Exit(1)
			}
			coon.Send("auth","0x817A=7%BJ#x%H=1rK")
			return coon,err
		},
	}
}

func SetValueTime(key string,value interface{},t int64){
	rc := pool.Get()
	defer rc.Close()
	key = strings.ToLower(key)
	rc.Do("set",key,value,"EX",t)
}

func GetValue(key string,t int)interface{}{
	rc := pool.Get()
	defer rc.Close()
	i, e := rc.Do("get", key)
	if e!=nil {
		log.Print(e)
		return 0
	}
	if t == 1 {
		reply, _ := redis.String(i,e)
		return reply
	}else if t == 2 {
		reply, _:= redis.Int64(i,e)
		return reply
	}else if t == 3{
		i, e := rc.Do("get", key)
		if e!=nil {
			fmt.Println("e=",e)
			return 0
		}
		reply, _:= redis.String(i,e)
		return reply
	}else if t == 5{
		r, _ := redis.Bytes(rc.Do("GET", key))
		var v *models.SSysVersion
		json.Unmarshal(r, &v)
		return *v
	}
	return 0
}

func SetMachine(ver *model.VMachine){
	rc := pool.Get()
	defer rc.Close()
	data, _ := json.Marshal(&ver)
	rc.Do("set",voteEnums.Machine_addr+ver.Addr,data)
}

func ExistHash(hash string,success bool)bool{
	rc := pool.Get()
	defer rc.Close()
	hash = strings.ToLower(hash)
	if success {
		reply, err := redis.String(rc.Do("get",enums.HASH_SUCC+hash))
		if err!=nil {
			fmt.Println("get hash SUCC err=",err)
		}
		return strings.EqualFold(reply,hash)
	}else{
		reply, err := redis.String(rc.Do("get",enums.HASH_WAIT+hash))
		if err!=nil {
			fmt.Println("get hash WAIT err=",err)
		}
		return strings.EqualFold(reply,hash)
	}
}
func SetHash(hash string,success bool){
	rc := pool.Get()
	defer rc.Close()
	hash = strings.ToLower(hash)
	if success {
		_, err := rc.Do("set", enums.HASH_SUCC+hash, hash, "EX", 3600*24*2)
		if err!=nil {
			fmt.Println("set hash SUCC err=",err)
		}
	}else{
		_, err :=rc.Do("set",enums.HASH_WAIT+hash,hash,"EX",3600*24*2)
		if err!=nil {
			fmt.Println("set hash WAIT err=",err)
		}
	}
}



func GetMachine(key string)interface{}{
	rc := pool.Get()
	defer rc.Close()
	r, e := rc.Do("get",voteEnums.Machine_addr+key)
	reply, _ := redis.Bytes(r,e)
	if reply == nil || len(reply) <= 0 {
		return nil
	}
	var m *model.VMachine
	err := json.Unmarshal(reply, &m)
	if err!=nil {
		fmt.Println("GetContract Unmarshal 错误=",err)
		return nil
	}
	return *m
}

func DeleteMachine(key string){
	rc := pool.Get()
	defer rc.Close()
	rc.Do("DEL",voteEnums.Machine_addr+key)
}