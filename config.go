package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"lottery2/Logger"
)

var config *configStruct

// configStruct 配置文件结构体
type configStruct struct {
	General struct{
		Production     bool   `yaml:"production"`
		BaseUrl string `yaml:"base_url"`
		ListenPort string `yaml:"listen_port"`
		Act_recache_time     int   `yaml:"act_recache_time"`
		Giftlist_recache_time     int   `yaml:"giftlist_recache_time"`
		Gift_sync_time  int   `yaml:"gift_sync_time"`
	}`yaml:"genenral"`
	Db struct{
		Server string `yaml:"server"`
		Port string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Db string `yaml:"db"`
		RedisAddr string `yaml:"redis_addr"`
		RedisPwd string `yaml:"redis_pwd"`
		RedisDb int `yaml:"redis_db"`
	}`yaml:"db"`
	SSO struct{
		ServiceName   string `yaml:"service_name"`
		ClientId     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	}`yaml:"sso"`
}

// getConfig 读取配置文件并返回
func getConfig()(c *configStruct){
	confFile,err := ioutil.ReadFile("config.yaml")
	if err != nil {
		Logger.FATAL.Fatalln("配置文件加载失败！")
	}
	err = yaml.Unmarshal(confFile,&c)
	if err != nil {
		Logger.FATAL.Fatalln("配置文件加载失败！("+err.Error()+")")
	}
	return
}


func UseConfig(){
	config = getConfig()
}