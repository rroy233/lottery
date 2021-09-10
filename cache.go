package main

import (
	jjson "github.com/json-iterator/go"
	"lottery2/Logger"
	"os"
	"strconv"
	"time"
)

type gotInfoStruct struct {
	Gift_id int `json:"gift_id"`
	Gift_name string `json:"gift_name"`
	Name string `json:"name"`
	Uid int `json:"uid"`
}

//func cacheActWoker(){
//	Logger.Info.Println("[缓存]已开启活动信息缓存，更新时间为",config.General.Act_recache_time,"s")
//	for{
//		sysCacheAct()
//		time.Sleep(time.Duration(config.General.Act_recache_time)*time.Second)
//	}
//}

func systemCacheGiftWoker(){
	Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"]已开启奖品信息缓存，更新时间为",config.General.Giftlist_recache_time,"s")
	for{
		systemCacheGift()
		time.Sleep(time.Duration(config.General.Giftlist_recache_time)*time.Second)
	}
}

func giftSyncWorker(adminID int){
	Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"][admin:"+strconv.Itoa(adminID)+"]已开启奖品获奖数量redis->mysql同步，更新时间为",config.General.Gift_sync_time,"s")
	for{
		if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status").Val() != "1" {
			break
		}
		sysSyncGiftData(adminID)
		time.Sleep(time.Duration(config.General.Gift_sync_time)*time.Second)
	}
	Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"]已开启奖品获奖数量redis->mysql同步结束")
}

func onlineUserCounter(adminID int){
	maxOnline := int64(0)
	Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"]已开启在线用户检测进程，更新时间为6s")
	//初始化
	onlineKeyName := MD5_short(strconv.FormatInt(time.Now().Unix(),10))
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:onlineKeyName",onlineKeyName,-1)

	for{
		if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status").Val() != "1" {
			break
		}
		onlineKeyName = rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:onlineKeyName").Val()
		onlineUserCount := rdb.SCard(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:Online:"+onlineKeyName).Val()

		//缓存onlineUserCount供管理员查询
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:onlineUserCount",onlineUserCount,10*time.Second)

		//更新onlineKeyName
		onlineKeyName = MD5_short(strconv.FormatInt(time.Now().Unix(),10))
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:onlineKeyName",onlineKeyName,-1)

		if onlineUserCount > maxOnline {
			maxOnline = onlineUserCount
		}
		time.Sleep(5*time.Second)
	}
	Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"]在线用户检测进程结束，最高在线人数:",maxOnline)
}

// sysCacheAct 将mysql中的 act 缓存到Redis
func sysCacheAct(){
	UseConfig()
	sql :="SELECT * FROM `act`"
	acts := make([]dbact,0)
	err := db.Select(&acts,sql)
	if err != nil {
		Logger.FATAL.Println(err.Error())
	}

	checkRedis()
	for i, act := range acts{
		if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Status").Val() != "1"{
			tmp,err := jjson.Marshal(act)
			if err != nil {
				Logger.FATAL.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"][",i,"]创建缓存失败:ADMIN[", act.AdminBy,"]", act.Name,err.Error())
			}else{
				rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":SYSTEM:ACT_Cache",string(tmp),-1)
				Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"][",i,"]已创建活动缓存:ADMIN[", act.AdminBy,"]", act.Name)
			}
			Logger.Info.Println("[缓存PID:",os.Getpid(),",PPID",os.Getppid(),"][",i,"]已创建活动状态缓存:ADMIN[", act.AdminBy,"]", act.Name)
			rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Status", act.Status,-1)

			//定时相关初始化，转移到修改活动时设定
			if act.Open_type == 2 && rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Timer:openTS").Val()==""{
				rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Timer:Active",1,-1)
				rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Timer:Status",0,-1)
				rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(act.AdminBy)+":ACT:Timer:ID","",-1)
			}
		}

		//创建管理员缓存
		rdb.Set(ctx,"Lottery_2:SYS:Admin:"+strconv.Itoa(act.AdminBy), 1,-1)
	}
}

// systemCacheGift 将mysql中的 gift 数据缓存到Redis
func systemCacheGift()  {
	UseConfig()
	allGifts := make([]dbgift,0)
	checkDB()
	sql := "SELECT * FROM `gift`"
	err := db.Select(&allGifts,sql)
	if err != nil {
		Logger.FATAL.Println(err.Error())
	}
	checkRedis()

	//收集每个admin的奖品
	adminList := make(map[int][]dbgift,0)
	for _,v := range allGifts{
		adminList[v.AdminBy] = append(adminList[v.AdminBy],v)
	}

	//为每个admin都创建一条礼物缓存
	for adminBy,v := range adminList {
		tmp,err := jjson.Marshal(v)
		if err != nil {
			Logger.FATAL.Println("[全局缓存PID:",os.Getpid(),",PPID",os.Getppid(),"创建礼物缓存失败:ADMIN[",adminBy,"]",err.Error())
		}else{
			rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminBy)+":SYSTEM:GIFT_Cache",string(tmp),time.Duration(config.General.Giftlist_recache_time*2)*time.Second)
		}

		giftIDList := make([]int,0)
		for _,gift := range v {
			giftIDList = append(giftIDList,gift.Id)
		}
		tmp,err = jjson.Marshal(giftIDList)
		if err != nil {
			Logger.FATAL.Println("[全局缓存PID:",os.Getpid(),",PPID",os.Getppid(),"创建礼物id缓存失败:ADMIN[",adminBy,"]",err.Error())
		}else{
			rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminBy)+":SYSTEM:GIFT_ID",string(tmp),time.Duration(config.General.Giftlist_recache_time*2)*time.Second)
		}

	}
}

//扫描奖品信息，同步入mysql（覆盖）
func sysSyncGiftData(adminID int){
	checkDB()
	checkRedis()

	gifts := make([]dbgift,0)
	err := jjson.Unmarshal([]byte(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:GIFT_Cache").Val()),&gifts)
	if err != nil {
		Logger.FATAL.Println("[Redis-Mysql同步]解析失败",err.Error(),"，同步已中断")
		return
	}
	got := ""
	for i,gift := range gifts {
		got,err = rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Gift_"+strconv.Itoa(gift.Id)+":GOT").Result()
		if err != nil {
			Logger.FATAL.Println("[Redis-Mysql同步][",i,"]redis读取失败:",err.Error(),",已跳过")
			continue
		}
		_,err := db.Exec("UPDATE `gift` SET `got` = ? WHERE `gift`.`id` = ?;",got,gift.Id)
		if err != nil {
			Logger.FATAL.Fatalln("[Redis-Mysql同步][",i,"]sql执行错误：",err.Error())
		}
	}
}

//TODO
//同步用户获奖信息
//
//活动结束后将用户获奖记录存入mysql lucky_list表中
func syncGotData(adminID int){
	checkDB()
	checkRedis()

	keys := make([]string,0)
	gotInfo := new(gotInfoStruct)
	rdb.Keys(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":USER:Got:*").ScanSlice(&keys)
	if len(keys) == 0 {
		return
	}
	Logger.Info.Println("[用户获奖信息]待同步keys：",keys)
	for i,v := range keys {
		jjson.Unmarshal([]byte(rdb.Get(ctx,v).Val()),&gotInfo)
		_,err := db.Exec("INSERT INTO `lucky_list` (`id`, `uid`,`name`, `gift_id`,`gift_name`, `got_time`,`admin_by`) VALUES (NULL, ?,?,?, ?, ?,?);",gotInfo.Uid,gotInfo.Name,gotInfo.Gift_id,gotInfo.Gift_name,time.Now().Unix(),adminID)
		if err != nil {
			Logger.FATAL.Fatalln("[用户获奖信息][",i,"]Redis-mysql同步失败：",err.Error())
		}
		Logger.Info.Println("[用户获奖信息][",i,"]已同步：",gotInfo)
	}
}

//TODO
func getActStatus(adminID int) bool {
	if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status").Val() == "1" {
		return true
	}else{
		return false
	}
}
