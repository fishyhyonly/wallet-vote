package main

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/logs"
	"github.com/beego/i18n"
	_ "wallet-vote/dbUtil"
	_ "wallet-vote/nodeTask"
	_ "wallet-vote/routers"
)

func main(){
	log := logs.NewLogger()
	log.SetLogger(logs.AdapterConsole,`{"level":1,"color":false}`)
	logs.SetLogger(logs.AdapterFile, `{"filename":"vote.log","maxsize":512000,"maxlines":10000,"daily":true}`)
	logs.EnableFuncCallDepth(true)
	logs.Async()
	logs.Async(1e3)
	beego.Any("/",func(ctx *context.Context){
		ctx.Output.Body([]byte("bar"))
	})
	/*beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:   []string{"http://192.*.*.*:*","http://*:*","http://localhost:*","http://127.0.0.1:*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type","Accept-Language","AccessToken-Agent","Accept-date"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		AllowCredentials: true,
	}))*/
	i18n.SetMessage("zh-CN", "conf/locale_zh-CN.ini")
	i18n.SetMessage("en-US", "conf/locale_en-US.ini")
	i18n.SetMessage("ko-KR", "conf/locale_ko-KR.ini")
	beego.BConfig.RecoverPanic = true
	beego.Run()
}