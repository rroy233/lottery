package main

import (
	"lottery2/Logger"
	"os"
	"strconv"
	"time"
)


func actWatcher(adminID int,timerID string){
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Timer:Status",1,-1)
	Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]进程开启")
	openTS,_ := strconv.ParseInt(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Timer:openTS").Val(),10,64)
	endTS,_ := strconv.ParseInt(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Timer:endTS").Val(),10,64)
	var waitTime time.Duration
	for{
		if checkJob(adminID,timerID) == false {
			break
		}
		if time.Now().UnixNano() <= openTS{
			//未开始，睡眠至活动开始
			waitTime = time.Unix(0,openTS).Sub(time.Now())
			Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]正在等待活动开始,TS:",openTS,":",waitTime.Seconds(),"s")
			time.Sleep(waitTime)
			if checkJob(adminID,timerID) == false {
				break
			}
			//开启活动
			err := actOpener(adminID)
			if err != nil {
				Logger.FATAL.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]活动自动开启失败！！",err.Error())
				return
			}
			Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]活动自动开始成功")
			time.Sleep(500*time.Millisecond)
		}

		if  time.Now().UnixNano()> openTS && time.Now().UnixNano() <= endTS {
			//阻塞，等待结束
			waitTime = time.Until(time.Unix(0,endTS))
			Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]正在等待活动结束,TS:",endTS,":",waitTime.Seconds(),"s")
			time.Sleep(waitTime)
			if checkJob(adminID,timerID) == false {
				break
			}
			//关闭活动
			err := actCloser(adminID)
			if err != nil {
				Logger.FATAL.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]活动自动关闭失败！！",err.Error())
				return
			}
			Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]活动自动结束成功")
			time.Sleep(500*time.Millisecond)
			break
		}
	}
	Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]进程结束")
}

func checkJob(adminID int,timerID string)bool{
	active := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Timer:Active").Val()
	if active == "0"{
		Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]Active="+active+"，提前终止")
		return  false
	}
	id := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Timer:ID").Val()
	if id == timerID {
		return true
	}
	Logger.Info.Println("[活动监控进程][admin:"+strconv.Itoa(adminID)+" timerID:"+timerID+"]Active="+active+"，ID变更为"+id+"，提前终止")
	return false
}


func trashCleaner(fileName string,minute int){
	Logger.Info.Println("[自动清理]已收到任务清理:",fileName)
	time.Sleep(time.Duration(minute)*time.Minute)
	err := os.Remove(fileName)
	if err != nil {
		Logger.Info.Println("[自动清理]清理:",fileName,"失败:",err)
	}else{
		Logger.Info.Println("[自动清理]清理:",fileName,"成功:")
	}
}
