package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	Res "github.com/rroy233/go-res"
	"lottery2/Logger"
	sso "lottery2/SSO"
	"net/http"
	"strconv"
	"time"
)

var TZ = time.FixedZone("CST", 8*3600)

var ssoClient *sso.Client

func main()  {

	Logger.New()

	UseConfig()

	initRedis()
	initDB()

	checkDB()
	checkRedis()



	sysCacheAct()
	go systemCacheGiftWoker()

	ssoClient = sso.NewClient(config.General.Production,config.SSO.ServiceName,config.SSO.ClientId,config.SSO.ClientSecret)
	Logger.Debug.Println("抽奖程序后端已启动，正在监听 http://localhost:"+config.General.ListenPort)
	err := http.ListenAndServe(":"+config.General.ListenPort,nil)
	if err != nil {
		Logger.FATAL.Fatalln(err.Error())
	}
}


func setHeaders(w http.ResponseWriter){
	if config.General.Production == false {
		w.Header().Add("Access-Control-Allow-Origin","*")
	}else{
		w.Header().Add("Access-Control-Allow-Origin","https://app.roy233.com")
		w.Header().Add("Access-Control-Allow-Credentials","true")
	}
	w.Header().Add("Access-Control-Max-Age","600")
	w.Header().Add("Access-Control-Allow-Headers", "authorization, content-type");
}

// httpReturn 返回信息封装(纯文本)
func httpReturn (w *http.ResponseWriter,data string){
	(*w).Header().Set("Content-Type","application/json; charset=UTF-8")
	fmt.Fprintf(*w,data)
}


// getIP 获取ip
func getIP(r *http.Request) string{
	ip := r.Header.Get("X-Real-IP")
	if ip == ""{
		// 当请求头不存在即不存在代理时直接获取ip
		ip = r.RemoteAddr
	}
	return ip
}

// MD5_short 生成6位MD5
func MD5_short(v string)string{
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil)[0:5])
}

// MD5 生成MD5
func MD5(v string)string{
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil))
}

//httpReturnErrJson 返回信息封装（json格式的错误信息）
func httpReturnErrJson(w *http.ResponseWriter,data string){
	httpReturn(w,Res.New(-1).MsgText(data).String())
}

func getPostParams (r *http.Request) map[string]string{
	// 根据请求body创建一个json解析器实例
	decoder := json.NewDecoder(r.Body)

	// 用于存放参数key=value数据
	var p map[string]string

	// 解析参数 存入map
	decoder.Decode(&p)

	return p
}


func ts2DateString(ts string) string {
	timestamp,_ := strconv.ParseInt(ts,10,64)
	return time.Unix(timestamp,0).In(TZ).Format("2006-01-02 15:04:05")
}

func dateString2ts(datetime string) (int64,error){
	tmp,err :=  time.ParseInLocation("2006-01-02 15:04:05",datetime,TZ)
	if err != nil {
		return 0,err
	}
	return tmp.Unix(),nil
}


func generateToken() string{
	return MD5_short(strconv.FormatInt(time.Now().UnixNano(),10))
}