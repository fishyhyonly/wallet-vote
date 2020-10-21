package controller

import (
	"encoding/json"
	"fmt"
	"wallet-token/enums"
	"wallet-token/models"
	"wallet-vote/model"
	"wallet-vote/nodeUtil"
	"wallet-vote/voteEnums"
)

type UserVoteController struct {
	baseController
}

//获取用户
func (this *UserVoteController)GetUser(){
	type userInfo struct {
		UserId int64
		NodeId int64
		SupNodeId int64
		PowPrice float64
		LockPrice float64
		VoteArea interface{}
		VoteNum float64
		TotalProfit float64
		VoteAddr string
		IsMac int
	}
	var userData userInfo
	user := model.VUser{}
	b, _ := this.x.Id(this.User.Id).Get(&user)
	if !b{
		userData = userInfo{UserId:this.User.Id,
			PowPrice:0,
			LockPrice:0,
			VoteArea:voteEnums.VALUEDEFAULT,
			VoteNum:0,
			TotalProfit:0,
			VoteAddr:voteEnums.VALUEDEFAULT,
		IsMac:0}
	}else{
		userData = userInfo{UserId:this.User.Id,
			NodeId:user.NodeId,
			SupNodeId:user.SupNodeId,
			LockPrice:user.PowPrice,
			PowPrice:user.LockPrice,
			VoteNum:user.VoteNum,
			TotalProfit:user.TotalProfit,
			VoteAddr:user.VoteAddr,
			IsMac:user.IsMac}
		if user.VoteArea != voteEnums.VALUEDEFAULT {
			var ad model.UserVote
			json.Unmarshal([]byte(user.VoteArea),&ad)
			userData.VoteArea = ad
		}else {
			userData.VoteArea =voteEnums.VALUEDEFAULT
		}
	}
	ObjectResponse(this.s,this.Controller, userData)
	return
}

//获取地址
func (this *UserVoteController)GetAddr(){
	uid := this.User.Id
	type alist struct {
		Id int64
		WalletAddr string
	}
	addrList := make([]alist,0)
	this.x.Table("w_wallet_base").Cols("id", "wallet_addr").Where("user_id = ? and sort = ? and show_type=? and wallet_type = ? ", uid, 1, enums.SHOW_TYPE_XS,enums.KTO).Find(&addrList)
	ObjectResponse(this.s,this.Controller,addrList)
	return
}

//绑定地址
func (this *UserVoteController)BindAddr(){
	if nodeUtil.TimeSection() {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("sysLock"))
		return
	}
	walletId,err := this.GetInt64("walletId")
	pwd := this.GetString("pwd")
	if err != nil {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("paraErr"))
		return
	}
	user := model.VUser{}
	b, _ := this.x.Id(this.User.Id).Get(&user)
	if !b {
		user = model.VUser{UserId:this.User.Id,
			PowPrice:0,
			LockPrice:0,
			VoteArea:voteEnums.VALUEDEFAULT,
			VoteNum:0,
			TotalProfit:0,
			VoteAddr:voteEnums.VALUEDEFAULT}
	}
	if user.VoteAddr!=voteEnums.VALUEDEFAULT {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("addrBind"))
		return
	}
	wb := models.WWalletBase{}
	bw, _ := this.x.Table("w_wallet_base").Cols("id", "pid", "password", "wallet_addr", "key_p").Where("id=? and user_id=? and show_type = ? ",walletId,user.UserId,enums.SHOW_TYPE_XS).Get(&wb)
	if !bw{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("walletNo"))
		return
	}
	eb, _ := this.x.Table("v_user").Where("vote_addr=?", wb.WalletAddr).Exist()
	if eb{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("addrBindAgain"))
		return
	}
	if nodeUtil.Md5(pwd)!=wb.Password{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("payPwdErr"))
		return
	}
	cu := models.CUser{}
	this.x.Table("c_user").Cols("phone", "phone", "email").Where("id=? ",this.User.Id).Get(&cu)
	user.UserPhone = cu.Phone
	user.UserEmail = cu.Email
	user.UserOther = cu.Other
	user.VoteAddr = wb.WalletAddr
	user.WalletId = wb.Pid
	user.VotePri = wb.KeyP
	var i int64
	if b {
		i, _ = this.s.Where("user_id=?",user.UserId).Update(&user)
	}else{
		i, _ = this.s.Insert(&user)
	}
	if i!=1 {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("getNodeErr"))
		return
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

//绑定节点
func (this *UserVoteController)BindNode(){
	if nodeUtil.TimeSection() {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("sysLock"))
		return
	}
	voteId,err := this.GetInt64("voteId")
	pwd := this.GetString("pwd")
	if err != nil {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("paraErr"))
		return
	}
	user := model.VUser{}
	this.x.Id(this.User.Id).Get(&user)
	var p string
	_, err = this.x.Table("w_wallet_base").Cols("Password").Where("id =?",user.WalletId).Get(&p)
	if nodeUtil.Md5(pwd)!=p{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("payPwdErr"))
		return
	}
	if user.VoteArea != voteEnums.VALUEDEFAULT {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("addrBind"))
		return
	}
	vote := model.VNodeInfo{}
	this.x.Id(voteId).Get(&vote)
	if vote.Status==voteEnums.VoteStatus_ing && vote.Stage==voteEnums.VoteStage_hx {
		uv := model.UserVote{Id:voteId,Area:vote.Area,Image:vote.Image,Name:vote.Name}
		data, _ := json.Marshal(uv)
		user.VoteArea = string(data)
		user.NodeId = vote.Id
		_, err := this.s.Where("user_id=?",user.UserId).Update(&user)
		if err!=nil {
			fmt.Println("节点绑定错误=",err)
		}
	}else{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("voteExp"))
		return
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}