package main

import (
	"bytes"
	"io/ioutil"
	"lottery2/Logger"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func viewUserIndex(w http.ResponseWriter,r *http.Request){
	//静态文件判断
	if r.URL.String() != "/"{
		_,err := os.Stat("./static"+r.URL.String())
		if err != nil {
			http.NotFound(w,r)
			return
		}
		f,err := os.Open("./static"+r.URL.String())
		if err != nil {
			http.NotFound(w,r)
			return
		}
		data,err := ioutil.ReadAll(f)
		if err != nil {
			http.NotFound(w,r)
			return
		}
		defer f.Close()
		w.Write(data)
		return
	}
	_,err := verifyAuth(r)
	if err != nil {
		http.Redirect(w,r,"/login",302)
		return
	}else{
		w.Write(views("index"))
		return
	}
}

func viewUserLogin(w http.ResponseWriter,r *http.Request){
	_,err := verifyAuth(r)
	if err == nil {
		http.Redirect(w,r,"/",302)
		return
	}
	ticket := r.FormValue("ticket")
	if ticket == ""{
		httpReturn(&w,"请扫描管理员提供的二维码进行登录")
		return
	}
	cookie := &http.Cookie{
		Name: "ticket",
		Value: ticket,
		Path: "/",
		HttpOnly: true,
		Expires: time.Now().Add(2*time.Minute),
	}
	http.SetCookie(w,cookie)
	w.Write(views("login"))
}

func viewUserRegister(w http.ResponseWriter,r *http.Request){
	_,err := verifyAuth(r)
	if err == nil {
		http.Redirect(w,r,"/",302)
		return
	}
	ticket := r.FormValue("ticket")
	if ticket == ""{
		httpReturn(&w,"请扫描管理员提供的二维码进行注册")
		return
	}
	cookie := &http.Cookie{
		Name: "ticket",
		Value: ticket,
		Path: "/",
		HttpOnly: true,
		Expires: time.Now().Add(2*time.Minute),
	}
	http.SetCookie(w,cookie)
	w.Write(views("register"))
}


func viewAdminDash(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		ssoRedirect(w,r)
		return
	}
	w.Write(views("admin.index"))
}

func viewAdminLottery(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		ssoRedirect(w,r)
		return
	}
	w.Write(views("admin.lottery"))
}


func viewAdminGifts(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		ssoRedirect(w,r)
		return
	}
	w.Write(views("admin.gifts_new"))
}

func viewAdminUser(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		ssoRedirect(w,r)
		return
	}
	w.Write(views("admin.user"))
}

func viewAdminSetting(w http.ResponseWriter,r *http.Request){
	if viewAuthAdmin(r) != true{
		ssoRedirect(w,r)
		return
	}
	w.Write(views("admin.setting"))
}

//模板加载函数
func views(template string,params ...map[string]string) (html []byte){
	name := ""
	data := make([]string,0)
	if strings.Index(template,".") != -1 {
		data = strings.Split(template,".")
		for _,n := range data {
			name = name + "/" + n
		}
	}else{
		name = "/"+template
	}
	file,err := os.Open("./template"+name+".html")
	defer file.Close()
	if err != nil {
		Logger.FATAL.Println("模板读取失败:",err.Error())
		html = []byte("模板读取失败")
		return
	}
	html,_ = ioutil.ReadAll(file)
	html = bytes.Replace(html,[]byte("{{api_url}}"),[]byte(config.General.BaseUrl),-1)

	if len(params) != 0 {
		for k,v := range params[0]{
			html = bytes.Replace(html,[]byte("{{"+k+"}}"),[]byte(v),-1)
		}
	}

	return
}


func viewAuthUser(r *http.Request) bool {
	_,err := verifyAuth(r)
	if err != nil {
		return false
	}
	return true
}

func viewAuthAdmin(r *http.Request) bool {
	c,err := r.Cookie("token")
	if err != nil {
		return false
	}
	cookie,_ := url.QueryUnescape(c.Value)
	err = ssoClient.VeryKey(cookie)
	if err != nil {
		return false
	}
	return true
}

func ssoRedirect(w http.ResponseWriter,r *http.Request){
	checkRedis()
	state := MD5_short(strconv.FormatInt(time.Now().UnixNano(),10))
	callback := r.URL.String()
	cookie := &http.Cookie{
		Name: "sso_"+state,
		Value: url.QueryEscape(callback),
		HttpOnly: true,
		Path: "/",
		Expires: time.Now().Add(5*time.Minute),
	}
	http.SetCookie(w,cookie)


	rdb.Set(ctx,"LOTTERY_2:SYS:SSO_State:"+state,"1",5*time.Minute)
	//w.Write(views("sso",map[string]string{"redirect_url":ssoClient.RedirectUrl(state)}))
	http.Redirect(w,r,ssoClient.RedirectUrl(state),302)
}
