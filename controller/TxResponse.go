package controller

import (
	"github.com/astaxie/beego"
	"github.com/go-xorm/xorm"
	"wallet-token/base"
)

func TxResponse(s *xorm.Session,o beego.Controller,msg string){
	s.Commit()
	s.Close()
	o.Data["json"] = base.Response{Errcode:0,Msg:msg}
	o.ServeJSON()
	return
}

func ErrResponse1008(s *xorm.Session,b beego.Controller,msg string){
	s.Commit()
	s.Close()
	b.Data["json"] = base.Response{Errcode:1008,Msg:msg}
	b.ServeJSON()
	return
}

func ErrVerResponse(b beego.Controller,i interface{}){
	b.Data["json"] = base.ObjectResponse{Response:base.Response{Errcode:2001,Msg:"版本信息提示"},Data:i}
	b.ServeJSON()
	return
}

func TxErrResponse(s *xorm.Session,b beego.Controller,msg string){
	s.Rollback()
	s.Close()
	b.Data["json"] = base.Response{Errcode:1001,Msg:msg}
	b.ServeJSON()
	return
}

func TxErrMsgResponse(s *xorm.Session,b beego.Controller,msg string){
	s.Rollback()
	s.Close()
	b.Data["json"] = base.Response{Errcode:1001,Msg:msg}
	b.ServeJSON()
	return
}

func TxObjectResponse(s *xorm.Session,o beego.Controller,i interface{}){
	s.Commit()
	s.Close()
	o.Data["json"] = base.ObjectResponse{Response:base.Response{Errcode:0,Msg:"操作成功"},Data:i}
	o.ServeJSON()
	return
}
func ObjectResponse(s *xorm.Session,o beego.Controller,i interface{}){
	s.Commit()
	s.Close()
	o.Data["json"] = base.ObjectResponse{Response:base.Response{Errcode:0,Msg:"操作成功"},Data:i}
	o.ServeJSON()
	return
}

