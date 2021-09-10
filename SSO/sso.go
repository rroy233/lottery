package sso

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	jjson "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	production bool
	serviceName string
	clientID string
	clientSecret string
}
type userInfo struct {
	Userid int `json:"userid"`
	Username string `json:"username"`
	Email string `json:"email"`
	Avatar    string `json:"avatar"`
	UserGroup string `json:"user_group"`
	ExpTime int64 `json:"exp_time"`
	Key string `json:"key"`
}

var keyPosition = []string{"userid","username","email","avatar","user_group","exp_time","key"}


type ssoResp struct {
	Status int `json:"status"`
	Data string `json:"data"`
	Msg string `json:"msg"`
}

var authServer string

// NewClient 创建一个SSO客户端实例
func NewClient(production bool,serviceName string,clientID string,clientSecret string) *Client {
	authServer = "https://account.roy233.com"
	return &Client{
		production: production,
		serviceName: serviceName,
		clientID: clientID,
		clientSecret: clientSecret,
	}
}

// NewUser 创建一个空的用户实例
func NewUser() (user *userInfo){
	user = new(userInfo)
	return user
}

// GetUserInfo 向鉴权服务器获取用户信息
func (c *Client) GetUserInfo(accessToken string) (user *userInfo,err error) {
	resp,err := http.Get(authServer+"/auth/userinfo?access_token="+accessToken+"&client_secret="+c.clientSecret)
	if err != nil {
		return nil,err
	}
	defer resp.Body.Close()
	data,err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil,err
	}

	user = new(userInfo)
	rs := new(ssoResp)
	err = jjson.Unmarshal(data,rs)
	if err != nil {
		return nil,err
	}
	if rs.Status == -1 {
		return nil,errors.New("凭证失效")
	}

	err = jjson.Unmarshal([]byte(rs.Data),user)
	if err != nil {
		return nil,err
	}
	user.ExpTime = time.Now().Add(3*time.Hour).Unix()
	user.Key = ""

	config := &jjson.Config{EscapeHTML: false}
	tmp,err := config.Froze().Marshal(user)
	if err != nil {
		return nil,err
	}

	user.Key = md5Short(string(tmp)+c.clientSecret)

	return
}

// VeryKey 验证登录信息是否有效
func (c *Client) VeryKey(data string) (err error) {
	user := new(userInfo)
	err = jjson.Unmarshal([]byte(data),user)
	if err != nil {
		//return errors.New("参数无效")
		return err
	}

	if user.ExpTime < time.Now().Unix() {
		return errors.New("凭证已过期")
	}

	keyGot := user.Key
	user.Key = ""

	tmp,err := jjson.Marshal(user)

	if keyGot != md5Short(string(tmp)+c.clientSecret){
		return errors.New("凭证无效")
	}

	return
}

// RedirectUrl 拼接重定向链接
func (c *Client) RedirectUrl(state string) string {
	return authServer+"/sso?client_id=" + c.clientID + "&state=" + state
}

// ParseCookie 解析cookie，生成userInfo实例
func (c *Client) ParseCookie(r *http.Request, cookieName string) (*userInfo,error) {
	//读取
	cookie,err := r.Cookie(cookieName)
	if err != nil {
		return nil,errors.New("cookie解析失败")
	}
	token,_ := url.QueryUnescape(cookie.Value)

	//验证
	err = c.VeryKey(token)
	if err != nil {
		return nil,errors.New("userInfo无效")
	}

	//装载
	user := new(userInfo)
	err = jjson.Unmarshal([]byte(token),user)
	if err != nil {
		return nil,errors.New("userInfo解析失败")
	}

	return user,nil
}

func (u userInfo) GetUID() string {
	return strconv.Itoa(u.Userid)
}

// MD5_short 生成6位MD5
func md5Short(v string)string{
	d := []byte(v)
	m := md5.New()
	m.Write(d)
	return hex.EncodeToString(m.Sum(nil)[0:5])
}
