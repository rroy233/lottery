package main

import (
	sql2 "database/sql"
	"errors"
	"fmt"
	jjson "github.com/json-iterator/go"
	"github.com/rroy233/go-res"
	"github.com/xuri/excelize/v2"
	"io"
	"lottery2/Logger"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type userItemStruct struct {
	Id int `json:"id"`
	Name string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	Promise string `json:"promise"`
}

type giftDisplayStruct struct {
	Id int `json:"id"`
	Name string`json:"name"`
	Got int`json:"got"`
	Total int`json:"total"`
}

type GiftEditStruct struct {
	Id int `json:"id"`
	Name string`json:"name"`
	Total int`json:"total"`
}

type ListStruct struct {
	Id int `json:"id"`
	gotInfoStruct
}

type RealTimeDataStruct struct {
	ActStatus       int `json:"act_status"`
	ActRealtimeData struct{
		Total int  `json:"total"`
		Got int `json:"got"`
		Online int `json:"online"`
	} `json:"act_realtime_data"`
	LuckyList []ListStruct `json:"lucky_list"`
	Gifts []giftDisplayStruct `json:"gifts"`
	Time string `json:"time"`
}

type giftNameList struct {
	Name string `json:"name"`
	Id int `json:"id"`
}

type dateTime struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

type actInfoStruct struct {
	ActId int `json:"act_id"`
	ActName string `json:"act_name"`
	ActOpenType int `json:"act_open_type"`
	ActOpenDateTime dateTime `json:"act_open_date_time"`
	ActEndDateTime dateTime `json:"act_end_date_time"`
	Timer struct{
		Status int `json:"status"`
		Id string `json:"id"`
	}`json:"timer"`
}

const (
	KB int64 = 1024
	MB int64 = 1024*1024
)

func adminActInfo(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	checkRedis()

	Logger.Info.Println("[act_info]",strconv.Itoa(auth.Userid))

	//查询活动表
	act := dbact{}
	cache,err := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":SYSTEM:ACT_Cache").Result()
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
	cache,err = rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":SYSTEM:GIFT_Cache").Result()
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

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())

	ress := res.New(0)
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

func dashboardCheck(w http.ResponseWriter,r *http.Request){
	setHeaders(w)

	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())
	onlineUserCount,_ := strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":SYSTEM:onlineUserCount").Val())
	if actStatus == 0 {
		httpReturnErrJson(&w,"活动尚未开始")
		return
	}

	//keep-alive
	//w.Header().Add("Connection","Keep-Alive")

	realTimeDataPack := RealTimeDataStruct{}
	realTimeDataPack.ActStatus = actStatus
	realTimeDataPack.ActRealtimeData.Got = 0
	realTimeDataPack.ActRealtimeData.Total = 0
	realTimeDataPack.Time = time.Now().Format("2006-01-02 15:04:05")
	realTimeDataPack.ActRealtimeData.Online = int(onlineUserCount)

	gotInfos := make([]ListStruct,0)
	liskItem := ListStruct{}
	tmpGot := gotInfoStruct{}
	tmp := rdb.LRange(ctx,"Lottery_2:"+auth.GetUID()+":ACT:RealtimeList",0,10).Val()
	totalNum := len(tmp)
	for _,v := range tmp {
		err = jjson.Unmarshal([]byte(v),&tmpGot)
		liskItem.Id = totalNum
		totalNum--
		liskItem.gotInfoStruct = tmpGot
		if err != nil {
			Logger.FATAL.Fatalln("gotInfos装载失败：",err.Error())
		}
		gotInfos = append(gotInfos,liskItem)
	}

	for _,v := range gotInfos {
		realTimeDataPack.LuckyList = append(realTimeDataPack.LuckyList,v)
		realTimeDataPack.ActRealtimeData.Got ++
	}

	//遍历礼物
	tempGift := giftDisplayStruct{}
	gifts := make([]dbgift,0)
	jjson.Unmarshal([]byte(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":SYSTEM:GIFT_Cache").Val()),&gifts)
	if len(gifts) == 0 {
		Logger.FATAL.Println("奖品缓存异常！:",err.Error())
	}
	for _,gift := range gifts {
		got := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Gift_"+strconv.Itoa(gift.Id)+":GOT").Val()
		tempGift.Id = gift.Id
		tempGift.Name = gift.Name
		tempGift.Total = gift.Total
		tempGift.Got,err = strconv.Atoi(got)
		if err != nil {
			Logger.FATAL.Fatalln("int-string转换失败："+err.Error())
		}
		realTimeDataPack.Gifts = append(realTimeDataPack.Gifts,tempGift)
		realTimeDataPack.ActRealtimeData.Total += gift.Total
	}

	ress,err := res.New(0).Json(realTimeDataPack)
	if err != nil {
		Logger.FATAL.Fatalln("最终结果封装失败：",err.Error())
	}
	httpReturn(&w,ress.String())
}

func adminGetUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	uid := r.FormValue("id")

	if uid == "" {
		httpReturnErrJson(&w,"参数无效")
		return
	}

	var user dbuser

	sql := "SELECT * FROM `user` WHERE `id` = ? and `admin_by`=?"
	err = db.Get(&user,sql,uid,auth.Userid)
	if err != nil {
		Logger.Info.Println("[管理员]用户信息取回失败，",err.Error())
		httpReturnErrJson(&w,"查询失败")
		return
	}

	ress := res.New(0)
	ress.AddData("id",user.Id)
	ress.AddData("name",user.Name)
	ress.AddData("phone_number",user.Phone_number)
	ress.AddData("gift_promise",user.Gift_promise)

	httpReturn(&w,ress.String())

}

func adminAddUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())
	if actStatus == 1 {
		httpReturnErrJson(&w,"活动正在进行中，现有用户信息已全量缓存，无法添加用户。")
		return
	}

	params := getPostParams(r)

	name := params["name"]
	phoneNumber := params["phone_number"]
	giftPromise := params["gift_promise"]

	if name == "" || phoneNumber == ""  {
		httpReturnErrJson(&w,"存在空参数")
		return
	}
	if giftPromise == "" {
		giftPromise = "-1"
	}

	Logger.Info.Println("[管理员]UID:",auth.Userid,"正在添加用户:", params)

	checkDB()

	//检查内定
	if giftPromise != "-1" {
		gift := dbgift{}
		//TODO 检查是否存在安全缺陷，越权访问
		err = db.Get(&gift,"SELECT * FROM `gift` WHERE `id`=? and `admin_by` = ?",giftPromise,auth.Userid)
		if err != nil {
			Logger.Error.Println("[管理员]新增用户失败，查询奖品总数失败:",err.Error(),auth)
			httpReturnErrJson(&w,"新增失败，查询奖品信息失败")
			return
		}

		Logger.Info.Println("[管理员]新增用户，新增内定，奖品id:",giftPromise)
		if gift.Promise + 1 > gift.Total {
			Logger.Error.Println("[管理员]新增用户失败，内定数超过奖品总数",auth)
			httpReturnErrJson(&w,"该奖项内定数超过奖品总数")
			return
		}

		result,err := db.Exec("UPDATE `gift` SET `promise` = `promise` + 1 WHERE `id` = ? and `admin_by`=?;",giftPromise,auth.Userid)
		if err != nil {
			Logger.Error.Println("[管理员]新增用户失败，修改内定数量失败",auth)
			httpReturnErrJson(&w,"系统异常，请联系管理员")
			return
		}
		if getAffectedRows(result) == 0 {
			httpReturnErrJson(&w,"添加失败,找不到要内定的奖项")
			return
		}
	}

	sql:= "INSERT INTO `user` (`id`, `name`, `phone_number`, `gift_promise`,`admin_by`) VALUES (NULL, ?, ?, ?,?);"
	_,err = db.Exec(sql,name, phoneNumber, giftPromise,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]添加用户失败,",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	Logger.Info.Println("[管理员]",auth,"添加用户成功")
	httpReturn(&w,res.New(0).MsgText("添加成功").String())

}

func adminEditUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())
	if actStatus == 1 {
		httpReturnErrJson(&w,"活动正在进行中，现有用户信息已全量缓存，无法编辑用户。")
		return
	}

	params := getPostParams(r)

	id := params["id"]
	name := params["name"]
	phoneNumber := params["phone_number"]
	giftPromise := params["gift_promise"]

	if id == "" || name == "" || phoneNumber == ""  {
		httpReturnErrJson(&w,"存在空参数")
		return
	}
	if i,_ := strconv.Atoi(giftPromise);i <= 0 {
		giftPromise = "-1"
	}

	checkDB()
	promiseO := ""
	//包括了对用户归属的验证
	sql := "SELECT `gift_promise` FROM `user` WHERE `id`=? and `admin_by` = ?"
	err = db.Get(&promiseO,sql,id,auth.Userid)
	if err != nil {
		Logger.FATAL.Println("查询原有内定信息失败！:",err.Error(),auth)
		httpReturnErrJson(&w,"编辑失败")
		return
	}

	//判断奖品是否存在，由于缓存生效需要时间
	if giftPromise != "-1"{
		tmp := ""
		sql = "select `name` from `gift` where `id`=? and `admin_by` = ?"
		err = db.Get(&tmp,sql,giftPromise,auth.Userid)
		if err != nil {
			Logger.FATAL.Println("[编辑用户]查询奖品信息失败！:",err.Error(),auth)
			httpReturnErrJson(&w,"编辑失败，查询奖品信息失败")
			return
		}
	}

	sql= "UPDATE `user` SET `name` = ?, `phone_number` = ?, `gift_promise`=?  WHERE `user`.`id` = ?;"
	result,err := db.Exec(sql,name, phoneNumber, giftPromise,id)
	if err != nil {
		Logger.Error.Println("[管理员]编辑用户失败,",err.Error(),auth)
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}


	if giftPromise != "-1" {
		gift := dbgift{}
		err = db.Get(&gift,"SELECT * FROM `gift` WHERE `id`=? and `admin_by` = ?",giftPromise,auth.Userid)
		if err != nil {
			Logger.Error.Println("[管理员]新增用户，查询奖品总数失败:",err.Error())
			httpReturnErrJson(&w,"参数无效")
			return
		}
		if promiseO != giftPromise && gift.Promise + 1 > gift.Total {
			Logger.Error.Println("[管理员]新增用户，内定数超过奖品总数")
			httpReturnErrJson(&w,"该奖项内定数超过奖品总数")
			return
		}
	}

	//检查内定变更
	if promiseO == "-1" && giftPromise != "-1"{
		//新增内定
		result,err = db.Exec("UPDATE `gift` SET `promise` = `promise` + 1 WHERE `id` = ? and `admin_by` = ?;",giftPromise,auth.Userid)
		if err != nil {
			Logger.FATAL.Println("处理内定信息失败,",err.Error())
			httpReturnErrJson(&w,"系统异常，请联系管理员")
			return
		}
	}else if promiseO != "-1" && giftPromise != "-1" && giftPromise != promiseO{
		//内定信息更改
		result,err = db.Exec("UPDATE `gift` SET `promise` = `promise` - 1 WHERE `id` = ? and `admin_by` = ?;",promiseO,auth.Userid)
		if err != nil {
			Logger.FATAL.Println("处理内定信息失败,",err.Error())
			httpReturnErrJson(&w,"系统异常，请联系管理员")
			return
		}

		result,err = db.Exec("UPDATE `gift` SET `promise` = `promise` + 1 WHERE `id` = ? and `admin_by` = ?;",giftPromise,auth.Userid)
		if err != nil {
			Logger.FATAL.Println("处理内定信息失败,",err.Error())
			httpReturnErrJson(&w,"系统异常，请联系管理员")
			return
		}
	}else if promiseO != "-1" && giftPromise == "-1" {
		//撤销内定
		result,err = db.Exec("UPDATE `gift` SET `promise` = `promise` - 1 WHERE `id` = ? and `admin_by` = ?;",promiseO,auth.Userid)
		if err != nil {
			Logger.FATAL.Println("处理内定信息失败,",err.Error())
			httpReturnErrJson(&w,"系统异常，请联系管理员")
			return
		}
	}

	Logger.Info.Println("[管理员]UID:",auth.GetUID(),"编辑用户,params:",params,"ID:",id,"affect_rows:",getAffectedRows(result))
	httpReturn(&w,res.New(0).MsgText("编辑成功").String())
}

func adminBulkAddUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}
	if auth.UserGroup == "6"{
		Logger.Info.Println("[验证服务][管理员]展示版本限制：",r)
		httpReturnErrJson(&w,"demo用户无法使用此功能")
		return
	}

	r.ParseMultipartForm(32 << 20)//32MB
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	if handler.Size >= (100*KB){
		httpReturn(&w,res.New(-1).MsgText("文件超过限制大小").String())
		return
	}
	tmpToken := MD5_short(strconv.FormatInt(time.Now().UnixNano(),10))
	f, err := os.OpenFile("./upload/"+tmpToken+".xlsx", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	io.Copy(f,file)
	defer f.Close()
	defer func() {
		err = os.Remove("./upload/"+tmpToken+".xlsx")
		if err != nil {
			Logger.Info.Println("[批量导入][管理员]删除文件失败：",err.Error())
		}
	}()
	xlsxFile, err := excelize.OpenFile("./upload/"+tmpToken+".xlsx")
	if err != nil {
		Logger.Info.Println("[批量导入][管理员]读取文件失败：",r,err.Error())
		httpReturnErrJson(&w,"文件读取失败(-1)")
		return
	}

	//取出现有的用户名
	checkDB()
	checkExist := false
	tmp := make([]string,0)
	dbNames := make(map[string]int,0)
	fileNames := make(map[string]int,0)

	//读取数据库中name列表
	err = db.Select(&tmp,"select `name` from `user` where `admin_by` = ?",auth.Userid)
	if err == nil && len(tmp) != 0{
		//说明现有用户，需要和现成的比对
		checkExist = true
		for _,v := range tmp{
			dbNames[v] = 1
		}
	}

	//读取表格
	rows := make([][]string,0)
	cols := make([][]string,0)
	rows,err = xlsxFile.GetRows("Sheet1")
	cols,err = xlsxFile.GetCols("Sheet1")
	if err != nil {
		Logger.Info.Println("[批量导入][管理员]读取工作表失败：",r,err.Error())
		httpReturnErrJson(&w,"文件读取失败(-1)")
		return
	}

	//读取表格中的姓名
	for _,name := range cols[0]{
		fileNames[name] = 1
		if checkExist == true && dbNames[name] == 1 {
			Logger.Info.Println("[批量导入][管理员]存在与现有用户冲突的数据")
			httpReturnErrJson(&w,"存在与现有用户冲突的数据")
			return
		}
	}
	if len(fileNames) != len(cols[0]) {
		Logger.Info.Println("[批量导入][管理员]存在重复的姓名")
		httpReturnErrJson(&w,"存在重复的姓名")
		return
	}

	sql:= "INSERT INTO `user` (`id`, `name`, `phone_number`, `gift_promise`,`admin_by`) VALUES (NULL, ?, ?, ?,?);"
	var result sql2.Result
	total := len(cols[0])
	done := int64(0)
	for i:= 1;i<total;i++{
		result,err = db.Exec(sql,rows[i][0],rows[i][1],"-1",auth.Userid)
		if err != nil {
			Logger.Info.Println("[批量导入][管理员",auth.Userid,"][",i,"]导入失败:",err)
		}else{
			Logger.Info.Println("[批量导入][管理员",auth.Userid,"][",i,"]导入:",rows[i],"成功")
			done += getAffectedRows(result)
		}
	}

	if done != int64(total)-1 {
		httpReturn(&w,res.New(-1).MsgText("系统异常").String())
		return
	}

	httpReturn(&w,res.New(0).MsgText("已成功导入"+strconv.FormatInt(done,10)+"条用户信息").String())
	return

}

func adminDelUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	params := getPostParams(r)

	id := params["id"]

	if id == ""{
		httpReturnErrJson(&w,"参数不为空")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())
	if actStatus == 1 {
		httpReturnErrJson(&w,"活动正在进行中，现有用户信息已全量缓存，无法删除用户。")
		return
	}

	checkDB()

	sql := "SELECT `gift_promise` FROM `user` WHERE `id` = ? and `admin_by` = ?"
	giftPromise := 0
	err = db.Get(&giftPromise,sql,id,auth.Userid)

	if err != nil {
		Logger.Error.Println("[管理员]删除用户失败,系统异常",err.Error(),auth)
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	sql = "DELETE FROM `user` WHERE `id` = ? and `admin_by` = ?"
	result,err := db.Exec(sql,id,auth.Userid)
	if getAffectedRows(result) == 0 {
		Logger.Info.Println("[管理员]删除用户失败,用户不存在")
		httpReturnErrJson(&w,"删除失败，用户不存在")
		return
	}

	//吊销当前用户的登录令牌
	if rdb.Exists(ctx,"Lottery_2:Login_Status_"+auth.GetUID()+":USER_"+id).Val() == 1{
		rdb.Del(ctx,"Lottery_2:Login_Status_"+auth.GetUID()+":USER_"+id)
	}

	//处理查内定
	result,err = db.Exec("UPDATE `gift` SET `promise` = `promise` - 1 WHERE `id` = ? and `admin_by`=?;",giftPromise,auth.Userid)
	if err != nil {
		Logger.FATAL.Println("[管理员]删除用户操作，内定信息处理错误",err)
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	Logger.Info.Println("[管理员]UID:",auth.Userid,"删除用户",id,"成功")
	httpReturn(&w,res.New(0).MsgText("删除成功").String())
}

func adminBulkDelUser(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	actStatus,_ :=  strconv.Atoi(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val())
	if actStatus == 1 {
		httpReturnErrJson(&w,"活动正在进行中，现有用户信息已全量缓存，无法批量删除用户。")
		return
	}

	checkDB()

	sql := "DELETE FROM `user` WHERE `admin_by` = ?"
	_,err = db.Exec(sql,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]批量删除用户失败,",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}
	Logger.Info.Println("[管理员]UID:",auth.Userid,"。执行批量删除操作！")

	//吊销所有用户的登录凭证
	userLSKeys := make([]string,0)
	rdb.Keys(ctx,"Lottery_2:Login_Status_"+auth.GetUID()+":*").ScanSlice(&userLSKeys)
	for i := range userLSKeys{
		rdb.Del(ctx,userLSKeys[i])
		Logger.Info.Println("[管理员]已吊销登录凭证:",userLSKeys[i])
	}

	//重置内定信息
	_,err = db.Exec("UPDATE `gift` SET `promise` = 0 WHERE `admin_by` =?",auth.Userid)

	httpReturn(&w,res.New(0).MsgText("删除成功").String())
}

func adminGetAct(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	checkDB()

	actinfo := actInfoStruct{}
	act := dbact{}
	sql:= "SELECT * FROM `act` where `admin_by` = ?"
	err = db.Get(&act,sql,auth.Userid)
	if err != nil {
		Logger.FATAL.Println("[管理员]查询活动失败",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	actinfo.ActId = act.Id
	actinfo.ActName = act.Name
	actinfo.ActOpenType = act.Open_type

	actinfo.ActOpenDateTime.Date = strings.Split(ts2DateString(act.Open_time)," ")[0]
	actinfo.ActOpenDateTime.Time = strings.Split(ts2DateString(act.Open_time)," ")[1]
	actinfo.ActOpenDateTime.Time = actinfo.ActOpenDateTime.Time[:len(actinfo.ActOpenDateTime.Time)-3]


	actinfo.ActEndDateTime.Date = strings.Split(ts2DateString(act.End_time)," ")[0]
	actinfo.ActEndDateTime.Time = strings.Split(ts2DateString(act.End_time)," ")[1]
	actinfo.ActEndDateTime.Time = actinfo.ActEndDateTime.Time[:len(actinfo.ActEndDateTime.Time)-3]

	timerID := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:ID").Val()
	timerActive := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active").Val()
	if timerID != "" && timerActive=="1"{
		actinfo.Timer.Status = 1
		actinfo.Timer.Id = timerID
	}else{
		actinfo.Timer.Status = 0
		actinfo.Timer.Id = ""
	}

	ress,err := res.New(0).Json(actinfo)
	if err != nil {
		Logger.FATAL.Println("[管理员]查询活动，时间封装失败",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	httpReturn(&w,ress.String())

}

//TODO 修改活动运作机制
func adminEditAct(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}


	params := getPostParams(r)
	tmp := params["act_info"]
	if err != nil {
		httpReturnErrJson(&w,"参数获取失败")
		return
	}

	if tmp == "" {
		httpReturnErrJson(&w,"存在空参数")
		return
	}

	act := actInfoStruct{}
	err = jjson.Unmarshal([]byte(tmp),&act)
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	id := act.ActId
	actName := act.ActName
	actOpenType := act.ActOpenType

	actOpenTime := act.ActOpenDateTime.Date + " " + act.ActOpenDateTime.Time+":00"
	actEndTime := act.ActEndDateTime.Date + " " + act.ActEndDateTime.Time+":00"

	if actOpenTime == "" || actEndTime == "" {
		httpReturnErrJson(&w,"请填写活动开始时间和结束时间")
		return
	}

	//TODO 判断合法性
	//时间戳转换
	openTime, err := time.ParseInLocation("2006-01-02 15:04:05", actOpenTime,TZ)
	if err != nil {
		Logger.FATAL.Println("[管理员]时间转换出错",err.Error())
		httpReturnErrJson(&w,"时间转换出错(actOpenTime)")
		return
	}
	actOpenTime = strconv.FormatInt(openTime.Unix(),10)
	endTime, err := time.ParseInLocation("2006-01-02 15:04:05", actEndTime,TZ)
	if err != nil {
		Logger.FATAL.Println("[管理员]时间转换出错",err.Error())
		httpReturnErrJson(&w,"时间转换出错(actEndTime)")
		return
	}
	actEndTime = strconv.FormatInt(endTime.Unix(),10)
	now := time.Now().Unix()
	if openTime.Unix() < now || now >endTime.Unix() || openTime.Unix() > endTime.Unix() {
		Logger.FATAL.Println("[管理员]时间不合法")
		httpReturnErrJson(&w,"时间不合法")
		return
	}



	newGoroutine := false
	timerActive := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active").Val()
	timerID := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:ID").Val()
	if actOpenType == 2 {
		//定时
		if timerActive == "1" && timerID != ""{
			openTS,_ := strconv.ParseInt(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:openTS").Val(),10,64)
			endTS,_ := strconv.ParseInt(rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:endTS").Val(),10,64)
			//定时生效
			if openTS != openTime.UnixNano() || endTS != endTime.UnixNano(){
				//时间不一致
				newGoroutine = true
			}
		}else {
			//无定时生效
			newGoroutine = true
		}
		Logger.Debug.Println("set active")
		rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active",1,-1)
	}else{
		//手动
		if timerActive == "1" && timerID != ""{
			//定时生效
			rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:ID","",-1)
		}
		rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active",0,-1)
	}

	if newGoroutine{
		//创建新的
		timerID := MD5_short(strconv.FormatInt(time.Now().UnixNano(),10))
		rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:ID",timerID,-1)
		rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:openTS",openTime.UnixNano(),-1)
		rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:endTS",endTime.UnixNano(),-1)

		go actWatcher(auth.Userid,timerID)
	}

	sql := "UPDATE `act` SET `name` = ?, `open_type`=?, `open_time`=?, `end_time`=?  WHERE `id` = ? and `admin_by`=?;"
	result,err := db.Exec(sql,actName,actOpenType,actOpenTime,actEndTime,id,auth.Userid)
	if err != nil {
		Logger.FATAL.Println("[管理员]查询活动失败:",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}
	rows,_ := result.RowsAffected()
	if rows == 0 {
		httpReturnErrJson(&w,"什么都没更新")
		return
	}

	//强制同步
	sysCacheAct()

	httpReturn(&w,res.New(0).MsgText("更新成功").String())
}

func adminGetAllUser(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	dbUsers := make([]dbuser,0)
	users := make([]userItemStruct,0)
	userItem := userItemStruct{}

	checkDB()

	err = db.Select(&dbUsers,"SELECT `id`,`name`,`phone_number`,`gift_promise` FROM `user` where `admin_by`=?",auth.Userid)
	if err != nil {
		Logger.FATAL.Println("[管理员]取回所有用户信息异常：",err.Error())
	}
	count := 0
	for _,user := range dbUsers{
		userItem.Id = user.Id
		userItem.Name = user.Name
		userItem.PhoneNumber = user.Phone_number
		userItem.Promise = user.Gift_promise
		users = append(users,userItem)
		count++
	}

	ress := res.New(0)
	ress.AddData("count",count)
	tmp,err := jjson.Marshal(users)
	if err != nil {
		Logger.FATAL.Println("[管理员]取回所有用户信息时，json格式化异常：",err.Error())
	}
	ress.AddData("users",string(tmp))

	httpReturn(&w,ress.String())

}

func adminGetLuckyList(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	checkDB()

	List := make([]dbLuckyList,0)
	err = db.Select(&List,"SELECT * FROM `lucky_list` where `admin_by`=?",auth.Userid)
	if err != nil {
		Logger.FATAL.Println("中奖记录查询失败")
	}

	count := 0
	for i,item := range List {
		List[i].GotTime = ts2DateString(item.GotTime)
		count ++;
	}

	tmp,err := jjson.Marshal(List)

	if err != nil {
		Logger.FATAL.Println("中奖记录封装失败")
	}

	httpReturn(&w,res.New(0).AddData("data",string(tmp)).AddData("count",count).String())

}

func adminExportLuckyList(w http.ResponseWriter,r *http.Request)  {
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	checkDB()

	List := make([]dbLuckyList,0)
	err = db.Select(&List,"SELECT * FROM `lucky_list` where `admin_by`=?",auth.Userid)
	if err != nil {
		Logger.FATAL.Println("中奖记录查询失败")
	}

	count := 0
	for i,item := range List {
		List[i].GotTime = ts2DateString(item.GotTime)
		count ++;
	}

	if count == 0{
		httpReturnErrJson(&w,"无可导出的数据")
		return
	}

	f := excelize.NewFile()
	index := f.NewSheet("Sheet1")
	f.SetCellValue("Sheet1", "A1", "序号")
	f.SetCellValue("Sheet1", "B1", "姓名")
	f.SetCellValue("Sheet1", "C1", "奖品id")
	f.SetCellValue("Sheet1", "D1", "奖项")
	f.SetCellValue("Sheet1", "E1", "中奖时间")
	f.SetCellValue("Sheet1", "F1", "管理员")
	f.SetCellValue("Sheet1", "G1", "导出时间")

	line := 2
	for i,item := range List{
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d",line), i+1)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d",line), item.Name)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d",line), item.GiftId)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d",line), item.GiftName)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d",line), item.GotTime)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d",line), auth.Username)
		f.SetCellValue("Sheet1", fmt.Sprintf("G%d",line), time.Now().Format("2006-01-02 15:04:05"))
		line++
	}

	f.SetActiveSheet(index)
	token := generateToken()
	if err := f.SaveAs("./static/export/lottery_"+token+".xlsx"); err != nil {
		Logger.Error.Println("[导出中奖信息][管理员",auth.Userid,"]导出excel失败",err)
	}
	go trashCleaner("./static/export/lottery_"+token+".xlsx",1)
	httpReturn(&w,res.New(0).AddData("url",config.General.BaseUrl+"/export/lottery_"+token+".xlsx").String())

}

func adminGetAllGifts(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	Gifts := make([]dbgift,0)

	err = db.Select(&Gifts,"SELECT * FROM `gift` where `admin_by`=?",auth.Userid)
	if err != nil {
		Logger.FATAL.Println("奖品信息查询失败:",err.Error())
	}

	ress := res.New(0)
	ress.AddData("count",len(Gifts))
	tmp,err := jjson.Marshal(Gifts)
	if err != nil {
		Logger.FATAL.Println("奖品信息封装失败:",err.Error())
	}
	ress.AddData("gifts",string(tmp))

	httpReturn(&w,ress.String())
}

func adminGetGifts(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	id := r.FormValue("id")

	if id == "" {
		httpReturnErrJson(&w,"参数无效")
		return
	}

	//giftItem := GiftEditStruct{}
	gift := dbgift{}
	err = db.Get(&gift,"SELECT * FROM `gift` WHERE `id` = ? and `admin_by`=?",id,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]拉取奖品信息失败:",err.Error())
		httpReturnErrJson(&w,"查询失败(-1)")
		return
	}
	//giftItem.Id = gift.Id
	//giftItem.Name = gift.Name
	//giftItem.Total = gift.Total

	ress,err := res.New(0).Json(gift)
	if err != nil {
		Logger.Error.Println("[管理员]封装奖品信息失败:",err.Error())
		httpReturnErrJson(&w,"查询失败(-2)")
		return
	}

	httpReturn(&w,ress.String())
}

func adminGetGiftsNameList(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	giftOptions := make([]giftNameList,0)
	nameList := make(map[string]int,0)
	dbgiftTemp := make([]dbgift,0)

	checkRedis()
	tmp := rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":SYSTEM:GIFT_Cache").Val()

	err = jjson.Unmarshal([]byte(tmp),&dbgiftTemp)
	if err != nil {
		Logger.FATAL.Println("gifts name list 取回奖品信息失败:",err.Error())
		httpReturnErrJson(&w,"从Redis取回奖品信息失败")
		return
	}

	nameList["-1"] = 0
	giftOptions = append(giftOptions,giftNameList{Name: "无",Id: -1})
	for i,gift := range dbgiftTemp {
		nameList[strconv.Itoa(gift.Id)] = i + 1
		giftOptions = append(giftOptions,giftNameList{Name: gift.Name,Id: gift.Id})
	}

	tmp1,err := jjson.Marshal(nameList)
	tmp2,err := jjson.Marshal(giftOptions)
	ress := res.New(0).AddData("name_list",string(tmp1)).AddData("options",string(tmp2))
	if err != nil {
		Logger.FATAL.Println("gifts name list 封装奖品信息失败:",err.Error())
		httpReturnErrJson(&w,"封装奖品信息失败")
		return
	}

	httpReturn(&w,ress.String())

}

func adminEditGifts(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	params := getPostParams(r)

	Logger.Info.Println("[修改奖品]UID:",auth.Userid,",params:",params)
	id := params["id"]
	name := params["name"]
	total,err := strconv.ParseInt(params["total"],10,32)

	if id == "" || name == "" || total == 0 {
		httpReturnErrJson(&w,"存在空参数")
		return
	}

	checkDB()

	//检查更改是否合法
	gift := dbgift{}
	err = db.Get(&gift,"SELECT * FROM `gift` WHERE `id`=? and `admin_by`=?",id,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]编辑奖品，查询奖品信息失败:",err.Error())
		httpReturnErrJson(&w,"参数无效")
		return
	}
	if gift.Got + gift.Promise > int(total) {
		Logger.Error.Println("[管理员]编辑奖品，奖品总数设置不合理")
		httpReturnErrJson(&w,"奖品总数设置不合理")
		return
	}

	sql:= "UPDATE `gift` SET `name` = ?, `total` = ? WHERE `id` = ? and `admin_by`=?;"
	result,err := db.Exec(sql,name, total,id,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]编辑奖品失败,",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}
	Logger.Info.Println("[管理员]UID:",auth.GetUID(),"编辑奖品,params:",params,"ID:",id,"affect_rows:",getAffectedRows(result))
	httpReturn(&w,res.New(0).MsgText("编辑成功，缓存生效需要10s").String())
}

func adminAddGifts(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	err = r.ParseForm()
	if err != nil {
		httpReturnErrJson(&w,"参数解析失败")
		return
	}

	params := getPostParams(r)

	Logger.Info.Println("[新增奖品]UID:",auth.Userid,",params:",params)
	name := params["name"]
	total,err := strconv.ParseInt(params["total"],10,32)

	if name == "" || total == 0 {
		httpReturnErrJson(&w,"参数无效")
		return
	}

	checkDB()
	sql:= "INSERT INTO `gift` (`id`, `name`, `total`, `got`,`promise`,`admin_by`) VALUES (NULL, ?, ?, ?,?,?);"
	result,err := db.Exec(sql,name,total,0,0,auth.Userid)
	if err != nil {
		Logger.FATAL.Println("[新增奖品]异常1：",auth,err.Error())
		httpReturnErrJson(&w,"新增失败")
		return
	}

	if getAffectedRows(result) == 1 {
		httpReturn(&w,res.New(0).MsgText("新增奖品成功").String())
		return
	}else{
		Logger.FATAL.Println("[新增奖品]异常2：",auth,err.Error())
		httpReturnErrJson(&w,"新增异常")
		return
	}

}

func adminDelGifts(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	params := getPostParams(r)

	id := params["id"]

	if id == ""{
		httpReturnErrJson(&w,"参数不为空")
		return
	}

	checkDB()

	sql := "SELECT * FROM `gift` WHERE `id` = ? and `admin_by` = ?"
	gift := new(dbgift)
	err = db.Get(gift,sql,id,auth.Userid)

	if err != nil {
		Logger.Error.Println("[管理员]删除奖品失败,系统异常",err.Error(),auth)
		httpReturnErrJson(&w,"奖品信息查询失败")
		return
	}

	if gift.Got != 0 {
		Logger.Error.Println("[管理员]删除奖品失败,got!=0,",auth)
		httpReturnErrJson(&w,"删除失败，请清空中奖信息再删除")
		return
	}

	if gift.Promise != 0 {
		Logger.Error.Println("[管理员]删除奖品失败,Promise!=0,",auth)
		httpReturnErrJson(&w,"删除失败，请撤销当前奖项的内定再试")
		return
	}

	sql = "DELETE FROM `gift` WHERE `id` = ? and `admin_by` = ?"
	result,err := db.Exec(sql,id,auth.Userid)
	if getAffectedRows(result) == 0 {
		Logger.Info.Println("[管理员]删除用户失败,奖品不存在")
		httpReturnErrJson(&w,"删除失败，奖品不存在")
		return
	}

	Logger.Info.Println("[管理员]UID:",auth.Userid,"删除奖品",id,"成功")
	httpReturn(&w,res.New(0).MsgText("删除成功").String())
}

func openAct(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	//强制终止自动控制子程序
	rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active",0,-1)

	err = actOpener(auth.Userid)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	httpReturn(&w,res.New(0).MsgText("开启成功").String())

}

func CloseAct(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	//强制终止自动控制子程序
	rdb.Set(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Timer:Active",0,-1)

	if rdb.Get(ctx,"Lottery_2:"+auth.GetUID()+":ACT:Status").Val() == "0" {
		httpReturnErrJson(&w,"当前为已关闭状态")
		return
	}

	err = actCloser(auth.Userid)
	if err != nil {
		httpReturnErrJson(&w,err.Error())
		return
	}

	httpReturn(&w,res.New(0).MsgText("关闭成功").String())

}

//TODO
func ResetAct(w http.ResponseWriter,r *http.Request){
	setHeaders(w)
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	checkDB()

	sql := "DELETE FROM `lucky_list` WHERE `admin_by` = ?"
	_,err = db.Exec(sql,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]批量删除中奖信息失败,",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}

	sql= "UPDATE `gift` SET `got` = '0' WHERE `admin_by`=?;"
	_,err = db.Exec(sql,auth.Userid)
	if err != nil {
		Logger.Error.Println("[管理员]批量删除中奖信息失败,",err.Error())
		httpReturnErrJson(&w,"系统异常，请联系管理员")
		return
	}


	Logger.Info.Println("[管理员]UID:",auth.Userid,"。执行批量删除中奖记录操作！")

	httpReturn(&w,res.New(0).MsgText("删除成功！").String())
}


func actOpener(adminID int)(err error){
	checkRedis()
	checkDB()

	systemCacheGift()
	sysCacheAct()

	if rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status").Val() == "1" {
		err = errors.New("当前为已开启状态")
		return
	}

	//活动开关 永久
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status",1,-1)

	//扫描奖品信息，为每个奖项设置缓存区
	giftAvailableNum := 0
	giftCount := 0
	gifts := make([]dbgift,0)
	err = jjson.Unmarshal([]byte(rdb.Get(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":SYSTEM:GIFT_Cache").Val()),&gifts)
	for i,gift := range gifts {
		Logger.Info.Println("[开启活动][",i,"]已缓存奖品,id:",gift.Id)
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Gift_"+strconv.Itoa(gift.Id)+":NAME",gift.Name,-1)
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Gift_"+strconv.Itoa(gift.Id)+":GOT",0,-1)
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Gift_"+strconv.Itoa(gift.Id)+":TOTAL",gift.Total,-1)
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Gift_"+strconv.Itoa(gift.Id)+":PROMISE",gift.Promise,-1)
		giftAvailableNum += gift.Total
		giftCount++
	}

	//添加promise缓存+缓存用户名
	users := make([]dbuser,0)
	err = db.Select(&users,"SELECT * FROM `user` where `admin_by` = ?",adminID)
	if err != nil {
		Logger.FATAL.Fatalln("活动启动失败：缓存用户时，取出失败：",err.Error())
	}
	for i,user := range users {
		rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":USER:Name:"+strconv.Itoa(user.Id),user.Name,-1)
		if user.Gift_promise != "-1" {
			Logger.Info.Println("[开启活动][",i,"]添加promise缓存,用户id:",user.Id,"保奖品ID:",user.Gift_promise)
			rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":USER:Got_Promise:"+strconv.Itoa(user.Id),user.Gift_promise,-1)
		}
	}

	//可获奖总人数
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Total", giftAvailableNum,-1)
	//奖项数量
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:GiftCount", giftCount,-1)

	go giftSyncWorker(adminID)
	go onlineUserCounter(adminID)

	return
}

func actCloser(adminID int)(err error){
	//更改活动状态
	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status",0,-1)

	rdb.Del(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Total")

	sysSyncGiftData(adminID)    //同步每个商品的已获得数量
	syncGotData(adminID) //为每个用户创建对应的获奖记录

	keys := make([]string,0)
	rdb.Keys(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":*").ScanSlice(&keys)

	for i,v := range keys {
		Logger.Info.Println("[关闭活动][admin:",adminID,"][",i,"]Redis已删除：",v)
		rdb.Del(ctx,v)
	}

	systemCacheGift()
	sysCacheAct()

	rdb.Set(ctx,"Lottery_2:"+strconv.Itoa(adminID)+":ACT:Status",0,-1)
	return
}