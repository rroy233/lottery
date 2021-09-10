package main

import (
	"context"
	"database/sql"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"lottery2/Logger"
	"time"
)

var db *sqlx.DB //mysql client
var rdb *redis.Client//redis client
var ctx = context.Background()

type dbuser struct {
	Id int `db:"id"`
	Name string `db:"name"`
	Phone_number string `db:"phone_number"`
	Gift_promise string `db:"gift_promise"`
	AdminBy int `db:"admin_by"`
}

type dbact struct {
	Id int `db:"id"`
	Name string `db:"name"`
	Status int `db:"status"`
	Open_type int `db:"open_type"`
	Open_time string `db:"open_time"`
	End_time string `db:"end_time"`
	AdminBy int `db:"admin_by"`
}

type dbgift struct {
	Id int `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	Total int `db:"total" json:"total"`
	Got int `db:"got" json:"got"`
	Promise int `db:"promise" json:"promise"`
	AdminBy int `db:"admin_by" json:"admin_by"`
}

type dbLuckyList struct {
	Id int `db:"id"`
	Uid int `db:"uid"`
	Name string `db:"name"`
	GiftId int `db:"gift_id"`
	GiftName string `db:"gift_name"`
	GotTime string `db:"got_time"`
	AdminBy int `db:"admin_by"`
}

// initDB mysql初始化
func initDB() {
	UseConfig()
	var err error
	dsn := config.Db.Username+":"+config.Db.Password+"@tcp("+config.Db.Server+":"+config.Db.Port+")/"+config.Db.Db+"?charset=utf8mb4&parseTime=True"
	db,err = sqlx.Connect("mysql",dsn)
	if err != nil {
		Logger.FATAL.Println("[系统服务][异常]Mysql启动失败"+err.Error())
		return
	}
	Logger.Info.Println("[系统服务][成功]Mysql已连接")

	//最大闲置时间
	db.SetConnMaxIdleTime(5*time.Second)
	//设置连接池最大连接数
	db.SetMaxOpenConns(1000)
	//设置连接池最大空闲连接数
	db.SetMaxIdleConns(20)
	return
}

func initRedis(){
	rdb = redis.NewClient(&redis.Options{
		Addr: config.Db.RedisAddr,
		Password: config.Db.RedisPwd,
		DB: config.Db.RedisDb,
	})
	err := rdb.Ping(ctx).Err()
	if err != nil {
		Logger.FATAL.Fatalln("[系统服务][异常]Redis启动失败")
		return
	}
	Logger.Info.Println("[系统服务][成功]Redis已连接")
	return
}

// checkDB 检查mysql是否有连接
func checkDB(){
	if db == nil {
		Logger.FATAL.Fatalln("Mysql 初始化失败")
	}
	err := db.Ping()
	if err != nil {
		Logger.Info.Println("[系统服务]Mysql重新建立连接")
	}
}

// checkRedis 检查redis是否有连接
func checkRedis(){
	if rdb == nil {
		Logger.FATAL.Fatalln("Redis 初始化失败")
	}
	err := rdb.Ping(ctx).Err()
	if err != nil {
		Logger.Info.Println("[系统服务]Redis重新建立连接")
	}
}

func getAffectedRows(r sql.Result) int64 {
	tmp,_ := r.RowsAffected()
	return tmp
}


