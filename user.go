package main

import (
	"errors"
	jjson "github.com/json-iterator/go"
	Res "github.com/rroy233/go-res"
	"lottery2/Logger"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type AuthStruct struct {
	Uid int `json:"uid"`
	Token string `json:"token"`
	Exp_time int64 `json:"exp_time"`
	AdminID int `json:"admin_id"`
}

type myGiftStruct struct {
	ID int `json:"id"`
	Name string `json:"name"`
	GotTime string `json:"got_time"`
}

type vueDivider struct {
	Divider bool `json:"divider"`
	Inset bool `json:"inset"`
}

type giftInfoUser struct {
	Id int
	Name string
	Total int
}

func checkAuth(auth string) (bool,error) {
	tmp := new(AuthStruct)
	err := jjson.Unmarshal([]byte(auth),tmp)
	if err != nil {
		Logger.Error.Println("[验证服务]auth反序列化失败:",err.Error())
		return false,errors.New("登录状态验证失败(-1)")
	}
	if tmp.Token == MD5_short(strconv.Itoa(tmp.Uid)+strconv.FormatInt(tmp.Exp_time,10)+strconv.Itoa(tmp.AdminID)+"ROYYYY"){
		if rdb.Get(ctx,"Lottery_2:Login_Status_"+strconv.Itoa(tmp.AdminID)+":USER_"+strconv.Itoa(tmp.Uid)).Val() == tmp.Token {
			//判断当前登录状态是否有效+防止多设备
			return true,nil
		}
		return false,errors.New("当前设备登录状态已失效")
	}
	return false,errors.New("登录状态验证失败(-2)")
}

func verifyAuth(r *http.Request)(a AuthStruct,err error){
	if r.Method == "OPTIONS" {
		err = errors.New("method not allowed")
		return
	}
	err = nil
	ip := getIP(r)
	c,err := r.Cookie("user")
	if err != nil {
		Logger.Error.Println("[验证服务][失败]IP:"+ip+",空token访问")
		err = errors.New("验证失败(-1)")
		return
	}

	auth,_ := url.QueryUnescape(c.Value)

	if  auth == "" {
		Logger.Error.Println("[验证服务][失败]IP:"+ip+",空auth访问")
		err = errors.New("验证失败(-1)")
		return
	}else{
		err = jjson.Unmarshal([]byte(auth),&a)
		if err != nil {
			Logger.Info.Println("[验证服务][失败]IP:"+ip+",Auth解析失败,"+err.Error())
			err = errors.New("验证失败(-2)")
			return
		}
		ok := false
		if ok,err = checkAuth(auth);ok {
			if a.Exp_time < time.Now().Unix() {
				Logger.Info.Println("[验证服务][失败]UID:"+strconv.Itoa(a.Uid)+",凭证过期:", auth)
				err = errors.New("凭证过期")
				return
			}
		}else{
			Logger.Error.Println("[验证服务][失败]IP:"+ip+",Key校验失败:", auth)
			err = errors.New("登录凭证无效"+err.Error())
			return
		}
	}
	return
}

func actInfo(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := verifyAuth(r)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	checkRedis()

	Logger.Info.Println("[act_info]",strconv.Itoa(auth.Uid))

	//查询活动表
	act := dbact{}
	cache,err := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":SYSTEM:ACT_Cache").Result()
	if err != nil {
		Logger.Error.Println(err.Error())
		httpReturnErrJson(&w,"活动查询失败")
		return
	}
	err = jjson.Unmarshal([]byte(cache),&act)
	if err != nil {
		Logger.Error.Println(err.Error())
		httpReturnErrJson(&w,"系统异常(-1)")
		return
	}

	//查询奖品
	gifts := make([]dbgift,0)
	giftInfos := make([]giftInfoUser,0)
	cache,err = rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":SYSTEM:GIFT_Cache").Result()
	if err != nil {
		Logger.Error.Println(err.Error())
		httpReturnErrJson(&w,"奖品查询失败")
		return
	}
	err = jjson.Unmarshal([]byte(cache),&gifts)
	if err != nil {
		Logger.Error.Println(err.Error())
		httpReturnErrJson(&w,"系统异常(-1)")
		return
	}

	for _,v := range gifts {
		giftInfos = append(giftInfos,giftInfoUser{Id: v.Id,Name: v.Name,Total: v.Total})
	}

	gifto,_ := jjson.Marshal(giftInfos)

	actStatus,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Status").Val())
	ress := Res.New(0)
	ress.AddData("act_status",actStatus)
	ress.AddData("act_name",act.Name)
	ress.AddData("act_gift",string(gifto))
	ress.AddData("act_open_type",act.Open_type)
	timetemp,_ := strconv.ParseInt(act.Open_time,10,64)
	ress.AddData("act_open_time",time.Unix(timetemp,0).In(TZ).Format("2006-01-02 15:04:05"))

	timetemp,_ = strconv.ParseInt(act.End_time,10,64)
	ress.AddData("act_end_time",time.Unix(timetemp,0).In(TZ).Format("2006-01-02 15:04:05"))

	httpReturn(&w, ress.String())
}

func heartbeat(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := verifyAuth(r)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Status").Val())
	onlineKeyName := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":SYSTEM:onlineKeyName").Val()

	if actStatus == 1{
		rdb.SAdd(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":SYSTEM:Online:"+onlineKeyName,auth.Uid)
	}

	ress := Res.New(0)
	ress.AddData("ts",time.Now().Unix())

	//查询redis
	ress.AddData("act_status",actStatus)

	if actStatus == 1 {
		if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Got:"+strconv.Itoa(auth.Uid)).Val() == "" {
			ress.AddData("enable",1)
		}else{
			ress.AddData("enable",0)
		}
	}else{
		ress.AddData("enable",0)
	}

	httpReturn(&w,ress.String())
}

func lottery(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := verifyAuth(r)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	err = r.ParseForm()
	if err != nil {
		//logger
		httpReturnErrJson(&w,"参数解析出错")
		return
	}

	var ts int64

	params := getPostParams(r)

	Logger.Info.Println("[抽奖]UID:",auth.Uid,",param:",params)

	ts,err = strconv.ParseInt(params["ts"],10,64)
	if err != nil {
		httpReturnErrJson(&w,"参数无效(-1)")
		return
	}

	if time.Now().UnixNano()/1e6 - ts > 5000 {
		httpReturnErrJson(&w,"请求已过期")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Status").Val())

	if actStatus == 0 {
		httpReturnErrJson(&w,"活动未开始")
		return
	}

	if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Got:"+strconv.Itoa(auth.Uid)).Val() != "" {
		httpReturnErrJson(&w,"您已获奖，不用再抽了")
		return
	}

	if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Limit:"+strconv.Itoa(auth.Uid)).Val() != "" {
		httpReturnErrJson(&w,"你手速太快，我接受不了啦o(╥﹏╥)o")
		return
	}

	myPromise := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Got_Promise:"+strconv.Itoa(auth.Uid)).Val()

	actNum,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:GiftCount").Val())

	luckyNum := 0
	if myPromise != "" {
		Logger.Info.Println("[抽奖操作]UID:",auth.Uid,",ts:",ts,",检测到内定信息:", myPromise)
		luckyNum,_ = strconv.Atoi(myPromise)
		rdb.Decr(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":PROMISE")
	}else{
		giftIDList := make([]int,0)
		err = jjson.Unmarshal([]byte(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":SYSTEM:GIFT_ID").Val()),&giftIDList)
		if err != nil {
			httpReturn(&w,Res.New(0).AddData("info","系统异常，请联系管理员").String())
			return
		}
		luckyNum = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(actNum+1)
		Logger.Debug.Println("lucky_num:", luckyNum)
		if luckyNum == 0 {
			httpReturn(&w,Res.New(0).AddData("info","手气不好，再试一次？").String())
			rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Limit:"+strconv.Itoa(auth.Uid),1,2*time.Second)
			return
		}
		luckyNum = giftIDList[luckyNum-1]
	}

	giftName := rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":NAME").Val()
	giftGot,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":GOT").Val())
	giftTotal,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":TOTAL").Val())
	giftPromise,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":PROMISE").Val())

	if myPromise == "" {
		//非内定
		if giftGot + giftPromise >= giftTotal {
			httpReturn(&w,Res.New(0).AddData("info","手气不好，再试一次？").String())
			return
		}
	}

	gotInfo := map[string]interface{}{}
	gotInfo["gift_id"] = luckyNum
	gotInfo["gift_name"] = giftName
	gotInfo["uid"] = auth.Uid
	gotInfo["name"] = rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Name:"+strconv.Itoa(auth.Uid)).Val()

	tmp,_ := jjson.Marshal(gotInfo)

	//存储个人记录
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Got:"+strconv.Itoa(auth.Uid),string(tmp),-1)

	//向队列里推送，用于后台实时展示
	rdb.LPush(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:RealtimeList",string(tmp))

	//向对应奖项推送，用于后期同步
	err = rdb.SAdd(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":List",auth.Uid).Err()
	err = rdb.Incr(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":ACT:Gift_"+strconv.Itoa(luckyNum)+":GOT").Err()

	if err != nil {
		Logger.FATAL.Fatalln("Redis操作失败"+err.Error())
	}

	Logger.Info.Println("[抽奖操作]UID:",auth.Uid,",ts:",ts,",结果:", giftName)

	httpReturn(&w,Res.New(0).AddData("info","恭喜您获奖！请等待最终结果公布。").AddData("gift_name", giftName).String())

}

func myGifts(w http.ResponseWriter,r *http.Request) {
	setHeaders(w)
	auth,err := verifyAuth(r)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	vueList := make([]interface{},0)

	msg := ""

	Logger.Info.Println("[奖品查询]UID:",auth.Uid)

	if getActStatus(auth.AdminID) {
		//活动正在进行
		myGift := myGiftStruct{}
		gotInfo := gotInfoStruct{}
		err = jjson.Unmarshal([]byte(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(auth.AdminID)+":USER:Got:"+strconv.Itoa(auth.Uid)).Val()),&gotInfo)
		if err != nil {
			msg = "无记录"
		}else{
			myGift.ID = 1
			myGift.GotTime = "刚刚"
			myGift.Name = gotInfo.Gift_name
			if err != nil {
				Logger.FATAL.Println("取回获奖记录奖品封装失败",err.Error())
			}
			vueList = append(vueList,myGift)
			msg = "查询成功"
		}
	}else{
		Divider := vueDivider{true,true}
		myGift := myGiftStruct{}
		myGifts := make([]dbLuckyList,0)
		err = db.Select(&myGifts,"SELECT * FROM `lucky_list` WHERE `uid` = ?",auth.Uid)
		if err != nil {
			Logger.FATAL.Fatalln("取回获奖mysql查询失败",err.Error())
		}
		if len(myGifts) == 0 {
			msg = "无记录"
		}else {
			for i,gift := range myGifts {
				if i != 0 {
					vueList = append(vueList,Divider)
				}
				myGift.ID = i + 1
				myGift.GotTime = ts2DateString(gift.GotTime)
				myGift.Name = gift.GiftName
				vueList = append(vueList,myGift)
			}
			msg = "查询成功"
		}

	}
	ress,err := Res.New(0).MsgText(msg).Json(vueList)
	if err != nil {
		Logger.FATAL.Fatalln("取回获奖res封装失败",err.Error())
	}
	httpReturn(&w,ress.String())
}

