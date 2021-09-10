package Logger

import (
	"io"
	"log"
	"os"
	"time"
)

var (
	Debug *log.Logger
	Info *log.Logger
	Error *log.Logger
	FATAL *log.Logger
)

var nextDate string
var nowDate string
var nextDateUnix int64
var waitTime time.Duration

var logFile *os.File


// New 主程序启动时需要调用这个函数来初始化
func New()  {
	nextDate = ""
	nextDateUnix = 0

	setNewLogger()
	go watcher()
}

// watcher 用于在后台运行的日志监控进程
func watcher(){
	for{
		if nextDate == "" || time.Now().Unix()>= nextDateUnix { //初次运行或已经过了下个日期
			t := time.Now()

			tm1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 5, 0, t.Location())
			tm2 := tm1.AddDate(0, 0, 1)//次日凌晨

			nextDate = tm2.Format("2006-01-02")
			nextDateUnix = tm2.Unix()

			waitTime = time.Until(tm2)

			Debug.Println("[系统服务][日志监控进程]"+"已确定下一个苏醒时间")

			time.Sleep(waitTime)//睡眠直至第二天凌晨醒来
		}
		_ = logFile.Close()
		setNewLogger()
	}
}

// setNewLogger 开启新的日志记录线程
func setNewLogger(){

	nowDate = getTodayDateString()
	var err error

	//日志输出文件
	logFile, err = os.OpenFile("./log/"+nowDate+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Faild to open error logger file:", err)
	}

	//重新定义
	Debug = log.New(io.MultiWriter(os.Stderr), "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	Info = log.New(io.MultiWriter(logFile,os.Stderr), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(logFile, os.Stderr), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	FATAL = log.New(io.MultiWriter(logFile, os.Stderr), "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}

//getTodayDateString 获取今日日期string
func getTodayDateString() string {
	return time.Now().Format("2006-01-02")
}