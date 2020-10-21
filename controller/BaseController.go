package controller

import (
	"github.com/astaxie/beego"
	"github.com/beego/i18n"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-xorm/xorm"
	"strconv"
	"strings"
	"wallet-token/enums"
	"wallet-token/models"
	"wallet-vote/dbUtil"
	"wallet-vote/nodeUtil"
)

type baseController struct {
	beego.Controller
	x *xorm.Engine
	s *xorm.Session
	controllerName string
	actionName     string
	User nodeUtil.UserInfo
	i18n.Locale
}

var apiKey = beego.AppConfig.String("passUrl")

func (p *baseController) Prepare()  {
	_, actionName := p.GetControllerAndAction()
	al := p.Ctx.Request.Header.Get("Accept-Language")
	//vn := p.Ctx.Request.Header.Get("version")

	if al == "zh-CN" {
		p.Lang = al
	}else if al == "ko-KR"{
		p.Lang = "ko-KR"
	}else if al == "en-US"{
		p.Lang = "en-US"
	}else{
		p.Lang = "zh-CN"
	}
	p.x =dbUtil.Engine
	p.s = dbUtil.Engine.NewSession()
	p.s.Begin()
	/*version, b := ctxVersion(vn)
	if b {
		ErrVerResponse(p.Controller, version)
		return
	}*/
	index := strings.Index(apiKey, actionName)
	if index == -1 {
		s := p.Ctx.Request.Header.Get("Accept-date")
		token := p.Ctx.Input.Header("AccessToken-Agent")
		user, err := nodeUtil.ParseToken(s,token)
		if err!=nil {
			if err.Error() == jwt.ErrExpiredKey.Error() {
				ErrResponse1008(p.s,p.Controller, p.Tr("userExp"))
				return
			}else{
				ErrResponse1008(p.s,p.Controller, p.Tr("guif"))
				return
			}
		}else{
			value := dbUtil.GetValue(enums.UTOKEN+strconv.FormatInt(user.Id, 10), 3)
			if value == nil {
				ErrResponse1008(p.s,p.Controller, p.Tr("guif"))
				return
			}
			if value.(string) != user.Token{
				ErrResponse1008(p.s,p.Controller, p.Tr("tokenErr"))
				return
			}else{
				p.User = user
			}
		}
	}
}

func ctxVersion(v string)(models.SSysVersion,bool){
	value := dbUtil.GetValue(enums.VER, 5)
	if value.(models.SSysVersion).VersionId!=v {
		return value.(models.SSysVersion),true
	}
	return models.SSysVersion{},false
}

func CheckVersion(v string){

}