package main

import (
	"errors"
	jjson "github.com/json-iterator/go"
	Res "github.com/rroy233/go-res"
	"github.com/skip2/go-qrcode"
	"lottery2/Logger"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type adminAuth struct {
	Userid int `json:"userid"`
	Username string `json:"username"`
	UserGroup string `json:"user_group"`
	ExpTime int64 `json:"exp_time"`
	Key string `json:"key"`
}

func LoginController(w http.ResponseWriter,r *http.Request){
	setHeaders(w)

	if r.Method != "POST"{
		httpReturn(&w,Res.New(-1).MsgText("method not allowed.").String())
		return
	}
	ip := getIP(r)

	params := getPostParams(r)
	Logger.Info.Println("[登录]IP:",ip,"尝试登录，params:",params)

	c,err := r.Cookie("ticket")
	if err != nil {
		httpReturnErrJson(&w,"登录信息无效")
		return
	}
	ticket,_ := url.QueryUnescape(c.Value)
	name := params["name"]
	phone := params["phonenumber"]

	if ticket=="" || name == "" || phone == "" {
		httpReturnErrJson(&w,"参数为空")
		return
	}

	//限制操作频率
	checkRedis()

	//查看黑名单
	if rdb.Get(ctx,"BLACKLIST:"+ip).Val() != "" {
		httpReturnErrJson(&w,"您已被封禁30分钟")
		return
	}

	loginLimit := rdb.Get(ctx,"LOGIN:lottery:"+ip+":"+name).Val()
	if  loginLimit == "" {
		//创建记录
		rdb.Set(ctx,"LOGIN:lottery:"+ip+":"+name,1,30*time.Second)
	}else{
		loginLimitInt,_ := strconv.Atoi(loginLimit)
		if loginLimitInt > 4 {
			//放入黑名单，30分钟解除
			rdb.Set(ctx,"BLACKLIST:"+ip,"111",30*time.Minute)
			Logger.Info.Println("[黑名单]IP:",ip)
			httpReturnErrJson(&w,"您已被封禁30分钟")
			return
		}
	}

	checkDB()
	checkRedis()
	//校验ticket
	adminID := rdb.Get(ctx,"LOGIN:Lottery_Ticket:"+ticket).Val()
	if adminID == ""{
		httpReturnErrJson(&w,"Ticket无效")
		return
	}

	var sql string
	user := dbuser{}
	sql = "SELECT * FROM `user` WHERE `name` = ? AND `phone_number` = ? AND `admin_by`=?"
	err = db.Get(&user,sql,name,phone,adminID)
	if err != nil {
		rdb.Incr(ctx,"LOGIN:lottery:"+ip+":"+name)
		Logger.Info.Println("[登录]"+err.Error(),params)
		httpReturn(&w,Res.New(-1).MsgText("登录失败").String())
		return
	}

	auth := AuthStruct{}
	auth.Uid = user.Id
	auth.Exp_time = time.Now().Add(1*time.Hour).Unix()
	auth.AdminID = user.AdminBy
	auth.Token = MD5_short(strconv.Itoa(auth.Uid)+strconv.FormatInt(auth.Exp_time,10)+strconv.Itoa(auth.AdminID)+"ROYYYY")

	//存储用户有效登录状态到redis，同时限制设备数
	rdb.Set(ctx,"Lottery_2:Login_Status_"+strconv.Itoa(user.AdminBy)+":USER_"+strconv.Itoa(auth.Uid),auth.Token,1*time.Hour)

	tmp,_ := Res.New(0).Json(auth)
	httpReturn(&w,tmp.String())
}

func RegController(w http.ResponseWriter,r *http.Request){
	setHeaders(w)

	if r.Method != "POST"{
		httpReturn(&w,Res.New(-1).MsgText("method not allowed.").String())
		return
	}
	ip := getIP(r)

	params := getPostParams(r)
	Logger.Info.Println("[注册]IP:",ip,"尝试注册，params:",params)

	c,err := r.Cookie("ticket")
	if err != nil {
		httpReturnErrJson(&w,"参数无效")
		return
	}
	ticket,_ := url.QueryUnescape(c.Value)
	name := params["name"]
	phone := params["phonenumber"]

	if ticket=="" || name == "" || phone == "" {
		httpReturnErrJson(&w,"参数为空")
		return
	}

	checkDB()
	checkRedis()
	//校验ticket
	adminID := rdb.Get(ctx,"REG:Lottery_Ticket:"+ticket).Val()
	if adminID == ""{
		httpReturnErrJson(&w,"Ticket无效")
		return
	}

	var sql string
	id := -1
	sql = "SELECT `id` FROM `user` WHERE `name` = ? AND `admin_by`=?"
	err = db.Get(&id,sql,name,adminID)
	if err == nil {
		Logger.Info.Println("[注册]重复注册拦截"+err.Error(),params)
		httpReturn(&w,Res.New(-1).MsgText("您已注册过").String())
		return
	}

	sql = "INSERT INTO `user` (`id`, `name`, `phone_number`, `gift_promise`, `admin_by`) VALUES (NULL, ?, ?, '-1', ?);"
	result,err := db.Exec(sql,name,phone,adminID)
	if err != nil {
		Logger.Info.Println("[注册]增加用户失败"+err.Error(),params)
		httpReturn(&w,Res.New(-1).MsgText("注册失败").String())
		return
	}

	uid,_ := result.LastInsertId()
	auth := AuthStruct{}
	auth.Uid = int(uid)
	auth.Exp_time = time.Now().Add(1*time.Hour).Unix()
	auth.AdminID,_ = strconv.Atoi(adminID)
	auth.Token = MD5_short(strconv.Itoa(auth.Uid)+strconv.FormatInt(auth.Exp_time,10)+strconv.Itoa(auth.AdminID)+"ROYYYY")

	//存储用户有效登录状态到redis，同时限制设备数
	rdb.Set(ctx,"Lottery_2:Login_Status_"+adminID+":USER_"+strconv.Itoa(auth.Uid),auth.Token,1*time.Hour)

	tmp,_ := Res.New(0).Json(auth)
	httpReturn(&w,tmp.String())
}


func SSOController(w http.ResponseWriter,r *http.Request){
	state := r.FormValue("state")
	accessToken := r.FormValue("access_token")
	if state == "" || accessToken == "" {
		httpReturn(&w,"参数无效(-1)")
		return
	}


	err := rdb.Get(ctx,"LOTTERY_2:SYS:SSO_State:"+state).Err()
	if err != nil {
		httpReturn(&w,"参数无效(-2)")
		return
	}
	callback := "/admin/"
	tmp,err := r.Cookie("sso_"+state)
	if err == nil {
		callback,_ = url.QueryUnescape(tmp.Value)
	}

	userInfo,err := ssoClient.GetUserInfo(accessToken)
	if err != nil {
		httpReturn(&w,"参数无效(-3)")
		return
	}



	auth := new(adminAuth)
	auth.Userid = userInfo.Userid
	auth.Username = userInfo.Username
	auth.UserGroup = userInfo.UserGroup
	auth.ExpTime = userInfo.ExpTime
	auth.Key = userInfo.Key

	Logger.Info.Println("[登录]SSO登录:",auth)

	//判断是否需要初始化
	if rdb.Get(ctx,"Lottery_2:SYS:Admin:"+strconv.Itoa(auth.Userid)).Val() != "1"{
		//初始化
		err = initAdmin(auth.Userid)
		if err != nil {
			Logger.Info.Println("[管理员初始化]初始化失败",err)
			httpReturn(&w,"管理员初始化失败："+err.Error())
			return
		}
		Logger.Info.Println("[管理员初始化]初始化成功")
	}

	tmp1,err := jjson.Marshal(auth)
	newCookie := &http.Cookie{
		Name: "token",
		Value: url.QueryEscape(string(tmp1)),
		Path: "/",
		HttpOnly: true,
		Expires: time.Now().Add(3*time.Hour),
	}
	http.SetCookie(w,newCookie)

	http.Redirect(w,r,callback,302)
	return

}

func QrCodeController(w http.ResponseWriter,r *http.Request){
	qrToken := r.FormValue("token")
	if qrToken == "" {
		httpReturn(&w,"参数无效")
		return
	}

	content,err := rdb.Get(ctx,"QrCode:token:"+qrToken).Result()
	if err != nil {
		httpReturnErrJson(&w,"链接无效或已过期")
		return
	}

	pic,err := createQrcode(content)
	if err != nil {
		httpReturnErrJson(&w,"图片生成失败")
		return
	}
	w.Header().Set("Content-Type","image/png")
	w.Write(pic)
}

func LogoutController(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		httpReturnErrJson(&w,"登出失败")
		return
	}
	cookie := &http.Cookie{
		Name: "token",
		Value: "",
		Path: "/",
		Expires: time.Now(),
	}
	http.SetCookie(w,cookie)
	httpReturn(&w,Res.New(0).String())
}

func createQrcode(data string)([]byte,error){
	pic,err := qrcode.Encode(data,qrcode.Medium,256)
	if err != nil {
		return nil,errors.New("生成二维码失败")
	}
	return pic,nil
}

//管理员请求获取用户的登录入口，返回二维码连接
func adminMakeTicket(w http.ResponseWriter,r *http.Request){
	auth,err := ssoClient.ParseCookie(r,"token")
	if err != nil {
		Logger.Info.Println("[验证服务][管理员]验证失败：",r,err.Error())
		httpReturnErrJson(&w,"验证失败")
		return
	}

	params := getPostParams(r)
	actionType := params["actionType"]


	checkRedis()
	//resCache := rdb.Get(ctx,"LOGIN:Lottery_Ticket:admin_"+auth.GetUID()).Val()
	//if resCache != ""{
	//	//已经生成过
	//	httpReturn(&w,resCache)
	//	return
	//}

	//未生成
	qrToken := MD5_short(auth.Key+strconv.FormatInt(time.Now().Unix(),10)+"_qrtoken")
	ticket := MD5_short(auth.Key+strconv.FormatInt(time.Now().Unix(),10)+"_ticket")

	redirect_url:= ""
	if actionType=="login"{
		actionType = "登录"
		redirect_url = config.General.BaseUrl+"/login?ticket="+ticket
		rdb.Set(ctx,"LOGIN:Lottery_Ticket:"+ticket,auth.Userid,2*time.Minute)
	}else if actionType == "register"{
		actionType = "注册"
		redirect_url = config.General.BaseUrl+"/register?ticket="+ticket
		rdb.Set(ctx,"REG:Lottery_Ticket:"+ticket,auth.Userid,2*time.Minute)
	}else{
		httpReturnErrJson(&w,"参数无效")
		return
	}

	rdb.Set(ctx,"QrCode:token:"+qrToken,redirect_url,2*time.Minute)
	rdb.Set(ctx,"LOGIN:Lottery_Ticket:"+ticket,auth.Userid,2*time.Minute)

	ress := Res.New(0).AddData("type",actionType).AddData("url","/qrcode?token="+qrToken).AddData("redirect_url",redirect_url).String()
	rdb.Set(ctx,"LOGIN:Lottery_Ticket:admin_"+auth.GetUID(),ress,2*time.Minute)

	httpReturn(&w,ress)
}

func initAdmin(adminID int) error {
	checkDB()
	var err error

	//创建活动
	sql:= "INSERT INTO `act` (`id`, `name`, `status`, `open_type`, `open_time`, `end_time`, `admin_by`) VALUES (NULL, '默认活动', '0', '1', ?, ?, ?);"
	_,err = db.Exec(sql,time.Now().Unix(),time.Now().Add(5*time.Minute).Unix(),adminID)
	if err != nil {
		return err
	}

	//创建奖品
	sql = "INSERT INTO `gift` (`id`, `name`, `total`, `got`, `promise`, `admin_by`) VALUES (NULL, '默认奖品', '10', '0', '0', ?);"
	_,err = db.Exec(sql,adminID)
	if err != nil {
		return err
	}

	//创建用户
	sql = "INSERT INTO `user` (`id`, `name`, `phone_number`, `gift_promise`, `admin_by`) VALUES (NULL, '技术支持', 'oisroy233@gmail.com', '-1', ?);"
	_,err = db.Exec(sql,adminID)
	if err != nil {
		return err
	}

	sysCacheAct()

	return err
}