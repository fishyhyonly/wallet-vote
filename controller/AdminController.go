package controller

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/utils"
	"github.com/go-gomail/gomail"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	"github.com/tealeg/xlsx"
	"io"
	"os"
	"strconv"
	"time"
	"wallet-token/enums"
	"wallet-token/models"
	"wallet-vote/dbUtil"
	"wallet-vote/ktoApi"
	"wallet-vote/model"
	"wallet-vote/nodeUtil"
	"wallet-vote/voteEnums"
)

type AdminController struct {
	baseController
}
type UserToken struct {
	Token string `json:"token"`
	Date int64 `json:"data"`
	Username string
}
func (this *AdminController)Login(){
	username := this.GetString("username")
	password := this.GetString("password")
	u := new(models.CUser)
	this.x.Id(1).Get(u)
	if u.Phone!=username || u.Password!=nodeUtil.Md5(password) {
		TxErrMsgResponse(this.s, this.Controller,"用户名或密码错误")
		return
	}
	var ver models.SSysVersion
	this.x.Table("s_sys_version").Get(&ver)
	md5 := nodeUtil.Md5(strconv.FormatInt(time.Now().UnixNano(), 10))
	dbUtil.SetValueTime(enums.UTOKEN+strconv.FormatInt(u.Id,10),md5,3600*24*21)
	token, _ := nodeUtil.CreateToken(nodeUtil.UserInfo{Id:u.Id, Token:md5,Version:ver.VersionId})
	ObjectResponse(this.s,this.Controller, UserToken{token,1,username})
	return
}
//查询用户kto地址
func (this *AdminController)QueryUserInfo(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	name := this.GetString("nameName")
	user := new(models.CUser)
	b, _ := dbUtil.Engine.Where("phone=? or email=? or other=?", name,name,name).Get(user)
	if !b {
		TxErrMsgResponse(this.s, this.Controller,"用户不存在")
		return
	}
	type alist struct {
		Id int64
		WalletAddr string
	}
	addrList := make([]alist,0)
	this.x.Table("w_wallet_base").Cols("id", "wallet_addr").Where("user_id = ? and show_type=? and wallet_type = ? ", user.Id, enums.SHOW_TYPE_XS,enums.KTO).Find(&addrList)
	userInfo := struct {
		Uid int64
		AddrList interface{}
	}{user.Id,addrList}
	ObjectResponse(this.s,this.Controller,userInfo)
	return
}

//创建节点
func (this *AdminController)CreateNode(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	name := this.GetString("name")
	ip := this.GetString("ip")
	area := this.GetString("area")
	userId,_ := this.GetInt64("userId")
	walletId,_ := this.GetInt64("walletId")
	endTime := this.GetString("endTime")
	voteAmount,er := this.GetFloat("voteAmount")
	lockAmount,er := this.GetFloat("lockAmount")
	if er != nil {
		TxErrMsgResponse(this.s, this.Controller,"数量格式错误")
		return
	}
	f, h, err := this.GetFile("imageName")
	if err != nil {
		logs.Info(err)
		TxErrMsgResponse(this.s, this.Controller,"图片处理失败")
		return
	}
	if h.Size>10*1024 {
		TxErrMsgResponse(this.s, this.Controller,"请上传10k以内的图片")
		return
	}
	defer f.Close()
	nodeT := new(model.VNodeInfo)
	b,_:=dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_hx).GroupBy("end_time").Get(nodeT)
	if b {
		if nodeT.EndTime!=endTime {
			TxErrMsgResponse(this.s, this.Controller,"请统一候选节点解仓时间")
			return
		}
	}
	/*if lockAmount>voteAmount {
		TxErrMsgResponse(this.s, this.Controller,"锁仓数量不能高于投票数量")
		return
	}*/
	s, _ := nodeUtil.NewSnowflake(0)
	nodeId := s.Generate()
	wb := models.WWalletBase{}
	user := model.VUser{}
	ub, _ := this.x.Id(userId).Get(&user)
	bw, _ := this.x.Table("w_wallet_base").Cols("id", "pid", "password", "wallet_addr", "key_p").Where("id=? and user_id=? and show_type = ? ",walletId,userId,enums.SHOW_TYPE_XS).Get(&wb)
	if !bw {
		TxErrMsgResponse(this.s, this.Controller,"钱包地址不存在,请确保改地址未删除")
		return
	}
	if !ub {
		cu := models.CUser{}
		this.x.Table("c_user").Cols("phone", "email", "other").Where("id=? ",userId).Get(&cu)
		uv := model.UserVote{Id:nodeId,Area:area,Image:h.Filename,Name:name}
		data, _ := json.Marshal(uv)

		user = model.VUser{UserId:userId,
			PowPrice:0,
			LockPrice:0,
			UserPhone:cu.Phone,
			UserEmail:cu.Email,
			UserOther:cu.Other,
			NodeId:nodeId,
			VoteArea:string(data),
			VoteNum:voteAmount,
			VoteAddr:wb.WalletAddr,
			WalletId:wb.Pid,
			VotePri:wb.KeyP,
			IsMac:voteEnums.ISMACY,
		}
		i, err := this.s.Insert(&user)
		if i!=1 {
			logs.Info(err)
			TxErrMsgResponse(this.s, this.Controller,"节点创建失败")
			return
		}
	}else{
		if user.VoteAddr==voteEnums.VALUEDEFAULT {
			uv := model.UserVote{Id:nodeId,Area:area,Image:h.Filename,Name:name}
			data, _ := json.Marshal(uv)
			user.VoteArea=string(data)
			user.VoteAddr=wb.WalletAddr
			user.NodeId=nodeId
			user.VoteNum=voteAmount
			user.VoteAddr=wb.WalletAddr
			user.WalletId=wb.Pid
			user.VotePri=wb.KeyP
			user.IsMac=voteEnums.ISMACY
		}else{
			if user.WalletId!=walletId {
				TxErrMsgResponse(this.s, this.Controller,"该用户已绑定了钱包地址，地址为【"+user.VoteAddr+"】")
				return
			}
		}
		if user.VoteArea!=voteEnums.VALUEDEFAULT {
			TxErrMsgResponse(this.s, this.Controller,"该用户已绑定了节点，不能再创建节点")
			return
		}
		dbUtil.Engine.Id(user.UserId).Update(&user)
	}
	balance := ktoApi.GetAbleBalance(wb.WalletAddr)
	bl, _ := strconv.ParseFloat(balance, 10)
	if bl<voteAmount {
		TxErrMsgResponse(this.s, this.Controller,"余额不足")
		return
	}
	fb := ktoApi.FrozenBalance(wb.WalletAddr, voteAmount)
	//bf :=ktoApi.FrozenBalanceOther(wb.WalletAddr,voteAmount)
	if !(fb) {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
		return
	}
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	nodeInfo := model.VNodeInfo{
		Id:nodeId,
		Name:name,
		Ip:ip,
		Area:area,
		Image:h.Filename,
		ChainAddr:wb.WalletAddr,
		UserId:userId,
		TotalAmount:voteAmount,
		LockAmount:lockAmount,
		MacAmount:voteAmount,
		IssueNum:1,
		Status:voteEnums.VoteStatus_ing, //1进行中，2历史
		Stage:voteEnums.VoteStage_hx,
		EndTime:endTime,
		CreateTime:t,
	}
	bill := createLockBill(userId,nodeInfo.Id,user.UserPhone,user.UserEmail,user.UserOther,name, area, h.Filename, voteAmount)
	this.s.Insert(&bill)
	i, _ := this.s.Insert(&nodeInfo)
	if i != 1 {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("voteExp"))
		return
	}
	this.SaveToFile("imageName", "static/image/" + h.Filename)
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func createLockBill(uid,nodeId int64,p,e,o,name,area,image string,amount float64)model.VVoteBill{
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	s, _ := nodeUtil.NewSnowflake(0)
	voteBill := model.VVoteBill{
		Id:s.Generate(),
		UserId:uid,
		UserPhone:p,
		UserEmail:e,
		UserOther:o,
		NodeName:name,
		NodeId:nodeId,
		NodeArea:area,
		NodeImage:image,
		Amount:amount,
		IssueNum:1,
		BillType:voteEnums.Bill_TYPE_TP,
		AmountType:voteEnums.Bill_TYPE_TP,
		CreateTime:t,
	}
	return voteBill
}

//查询节点
func (this *AdminController)QueryNodeList(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	name := this.GetString("name")
	addr := this.GetString("addr")
	s,_ :=this.GetInt("size")
	l,_ :=this.GetInt("limit")
	status,stage,issueNum,e := func()(int,int,int,int){
		e := 0
		status,err := this.GetInt("status")
		if err!=nil {
			e++
		}
		stage,err := this.GetInt("stage")
		if err!=nil {
			e++
		}
		issueNum,err := this.GetInt("issueNum")
		if err!=nil {
			e++
		}
		return status,stage,issueNum,e
	}()
	if e!=0 {
		TxErrMsgResponse(this.s, this.Controller,"参数格式错误")
		return
	}
	where := this.x.Where("1=?", 1)
	whereC := this.x.Where("1=?", 1)
	whereT := this.x.Where("1=?", 1)
	if status != 0 {
		where.And("status = ?",status)
		whereC.And("status = ?",status)
		whereT.And("status = ?",status)
	}
	if stage != 0 {
		where.And("stage = ?",status)
		whereC.And("stage = ?",status)
		whereT.And("stage = ?",status)
	}
	if issueNum != 0 {
		where.And("issue_num = ?",issueNum)
		whereC.And("issue_num = ?",issueNum)
		whereT.And("issue_num = ?",issueNum)
	}
	if name != "" {
		where.And("name = ?",name)
		whereC.And("name = ?",name)
		whereT.And("name = ?",name)
	}
	if addr != "" {
		where.And("chain_addr = ?",addr)
		whereC.And("chain_addr = ?",addr)
		whereT.And("chain_addr = ?",addr)
	}
	nodec := new(model.VNodeInfo)
	nodeList := new([]model.VNodeInfo)
	where.Limit(l,s*l).Desc("total_amount").Asc("end_time").Find(nodeList)
	count,_ := whereC.Count(nodec)
	t, _ :=whereT.Sum(nodec,"total_amount")
	p :=models.Page{Limit:l,Size:s,Total:count,TotalAmount:t,DataList:nodeList}
	ObjectResponse(this.s,this.Controller,p)
	return
}

//查询节点下用户
func (this *AdminController)QueryNodeUser(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	nodeId,_ := this.GetInt64("nodeId")
	s,_ :=this.GetInt("size")
	l,_ :=this.GetInt("limit")
	queryType:= this.GetString("queryType")
	queryValue:= this.GetString("queryValue")
	addr:= this.GetString("addr")
	user := new([]model.VUser)
	userc := new(model.VUser)
	omit := this.x.Omit("vote_pri", "wallet_id")
	omitC := this.x.Omit("vote_pri", "wallet_id")
	omitTV := this.x.Omit("vote_pri", "wallet_id")
	omitTS := this.x.Omit("vote_pri", "wallet_id")
	if nodeId!=0 {
		nodeInfo := new(model.VNodeInfo)
		this.x.Id(nodeId).Get(nodeInfo)
		if nodeInfo.Stage == voteEnums.VoteStage_cj {
			omit.And("sup_node_id=?",nodeId)
			omitC.And("sup_node_id=?",nodeId)
			omitTV.And("sup_node_id=?",nodeId)
			omitTS.And("sup_node_id=?",nodeId)
		}else{
			omit.And("node_id=?",nodeId)
			omitC.And("node_id=?",nodeId)
			omitTV.And("node_id=?",nodeId)
			omitTS.And("node_id=?",nodeId)
		}

	}
	if queryType!="" {
		omit.And(queryType+"=?",queryValue)
		omitC.And(queryType+"=?",queryValue)
		omitTV.And(queryType+"=?",queryValue)
		omitTS.And(queryType+"=?",queryValue)
	}
	if addr!="" {
		omit.And("vote_addr=?",addr)
		omitC.And("vote_addr=?",addr)
		omitTV.And("vote_addr=?",addr)
		omitTS.And("vote_addr=?",addr)
	}
	omit.Limit(l,s*l).Desc("vote_num").Find(user)
	c, _ := omitC.Count(userc)
	tv, _ := omitTV.Sum(userc, "vote_num")
	ts, _ := omitTV.Sum(userc, "sup_vote_num")
	ul := struct {
		Limit    int         `json:"limit"`
		Size     int         `json:"size"`
		Total    int64         `json:"total"`
		TotalVote    float64         `json:"totalVote"`
		TotalSup    float64         `json:"totalSup"`
		DataList interface{} `json:"dataList"`
	}{l,s,c,tv,ts,user}
	ObjectResponse(this.s,this.Controller,ul)
}

//用户投票流水
func (this *AdminController)QueryVoteBill(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	s,_ :=this.GetInt("size")
	l,_ :=this.GetInt("limit")
	nodeName:= this.GetString("nodeName")
	queryType:= this.GetString("queryType")
	queryValue:= this.GetString("queryValue")
	amountType,_ :=this.GetInt("amountType")
	where := this.x.Where("amount_type = ?", amountType)
	whereC := this.x.Where("amount_type = ?", amountType)
	whereT := this.x.Where("amount_type = ?", amountType)
	if nodeName!="" {
		where.And("node_name = ?",nodeName)
		whereC.And("node_name = ?",nodeName)
		whereT.And("node_name = ?",nodeName)
	}
	if queryType!="" {
		where.And(queryType+" = ?",queryValue)
		whereC.And(queryType+" = ?",queryValue)
		whereT.And(queryType+" = ?",queryValue)
	}
	bill := new([]model.VVoteBill)
	billc := new(model.VVoteBill)
	where.Limit(l,s*l).Desc("create_time").Find(bill)
	c, _ := whereC.Count(billc)
	t, _ := whereT.Sum(billc, "amount")
	p :=models.Page{Limit:l,Size:s,Total:c,TotalAmount:t,DataList:bill}
	ObjectResponse(this.s,this.Controller,p)
}

//查询矿机
func (this *AdminController)QueryMachine(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	mac := new([]model.VMachine)
	this.x.Find(mac)
	TxObjectResponse(this.s, this.Controller,mac)
	return
}

//改变矿机
func (this *AdminController)ChangeMachine(){
	if this.User.Id!=1 {
		TxErrMsgResponse(this.s, this.Controller,"没有权限")
		return
	}
	optType := this.GetString("optType")
	newAddr := this.GetString("newAddr")
	ip := this.GetString("ip")
	if optType == "add" {
		m := model.VMachine{Addr:newAddr,Ip:ip}
		dbUtil.SetMachine(&m)
		this.x.Insert(&m)
	}else if optType == "edit" {
		oldAddr := this.GetString("oldAddr")
		machine := dbUtil.GetMachine(oldAddr)
		if machine!=nil {
			dbUtil.DeleteMachine(oldAddr)
		}
		m := model.VMachine{Addr:newAddr,Ip:ip}
		dbUtil.SetMachine(&m)
		ma:=new(model.VMachine)
		this.x.Where("addr=?",oldAddr).Get(ma)
		ma.Addr = newAddr
		ma.Ip = ip
		_, err := this.x.Where("addr=?",oldAddr).Update(ma)
		fmt.Println(err)
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)SetNode(){
	nodeId,_ := this.GetInt64("nodeId")
	//1正常，2创世
	ty,_ := this.GetInt("type")
	var node model.VNodeInfo
	this.x.Id(nodeId).Get(&node)
	s, _ := nodeUtil.NewSnowflake(0)
	newNodeId := s.Generate()
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	var newNode model.VNodeInfo
	if ty== 1 {
		newNode = model.VNodeInfo{
			Id:newNodeId,
			Name:node.Name,
			Ip:node.Ip,
			Area:node.Area,
			Image:node.Image,
			UserId:node.UserId,
			ChainAddr:node.ChainAddr,
			TotalAmount:node.LockAmount,
			LockAmount:0,
			MacAmount:node.LockAmount,
			IssueNum:node.IssueNum+1,
			Status:node.Status,
			Stage:voteEnums.VoteStage_hx,
			EndTime:time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			CreateTime:t,
		}
		this.x.Insert(&newNode)
	}

	node.Stage = voteEnums.VoteStage_cj
	node.EndTime = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	this.x.Id(node.Id).Update(&node)
	var user []model.VUser
	this.x.Where("node_id=?",node.Id).Find(&user)
	for _, v := range user {
		v.SupNodeId = v.NodeId
		v.SupVoteNum = v.VoteNum
		v.VoteNum = -1
		v.SupProfit = -1
		if ty == 1 {
			v.NodeId = newNodeId
			if v.IsMac == voteEnums.ISMACY{
				v.VoteNum = newNode.TotalAmount
				voteBill := model.VVoteBill{
					Id:s.Generate(),
					UserId:v.UserId,
					UserPhone:v.UserPhone,
					UserEmail:v.UserEmail,
					UserOther:v.UserOther,
					NodeName:newNode.Name,
					NodeId:newNode.Id,
					NodeArea:newNode.Area,
					NodeImage:newNode.Image,
					Amount:newNode.TotalAmount,
					IssueNum:newNode.IssueNum,
					BillType:voteEnums.Bill_TYPE_TP,
					AmountType:voteEnums.Bill_TYPE_TP,
					CreateTime:t,
				}
				dbUtil.Engine.Insert(&voteBill)
			}
		}
		_, err := dbUtil.Engine.Id(v.UserId).Update(&v)
		fmt.Println(err)
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)SetNodeUser(){
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	var newSN []model.VNodeInfo
	//前13名的候选节点
	dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_hx).Desc("total_amount").Limit(13,0).Find(&newSN)
	//落选的节点
	var lowHX []model.VNodeInfo
	dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_hx).Desc("total_amount").Limit(60,13).Find(&lowHX)
	s, _ := nodeUtil.NewSnowflake(0)
	for _, node := range newSN {
		//产生新的节点
		newNodeId := s.Generate()
		newNode := model.VNodeInfo{
			Id:newNodeId,
			Name:node.Name,
			Ip:node.Ip,
			Area:node.Area,
			Image:node.Image,
			UserId:node.UserId,
			ChainAddr:node.ChainAddr,
			TotalAmount:0,
			LockAmount:0,
			MacAmount:0,
			IssueNum:node.IssueNum+1,
			Status:node.Status,
			Stage:voteEnums.VoteStage_hx,
			EndTime:time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			CreateTime:t,
		}

		var user []model.VUser
		this.x.Where("node_id=?",node.Id).Find(&user)
		for _, v := range user {
			v.SupNodeId = v.NodeId
			if v.VoteNum==0 {
				v.SupVoteNum = 0
			}else{
				v.SupVoteNum = v.VoteNum
			}
			v.VoteNum = 0
			v.SupProfit = 0
			v.NodeId = newNodeId
			_, err := this.s.Where("user_id=?",v.UserId).Cols("sup_node_id","sup_vote_num","vote_num","sup_profit","node_id").Update(&v)
			if err!=nil{
				logs.Info("候选用户竞选失败=",err,v.UserId)
			}
		}
		node.Stage=voteEnums.VoteStage_cj
		node.EndTime=time.Now().AddDate(0, 1, 0).Format("2006-01-02")
		_, err2 := this.s.Where("id=?",node.Id).Update(&node)
		if err2!=nil {
			logs.Info("错误2=",err2)
		}
		_, err3 := this.s.Insert(&newNode)
		if err3!=nil{
			logs.Info("错误3=",err3)
		}
	}


	for _, node := range lowHX {
		newNodeId := s.Generate()
		newNode := model.VNodeInfo{
			Id:newNodeId,
			Name:node.Name,
			Ip:node.Ip,
			Area:node.Area,
			Image:node.Image,
			UserId:node.UserId,
			ChainAddr:node.ChainAddr,
			TotalAmount:0,
			LockAmount:0,
			MacAmount:0,
			IssueNum:node.IssueNum+1,
			Status:node.Status,
			Stage:voteEnums.VoteStage_hx,
			EndTime:time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
			CreateTime:t,
		}

		var user []model.VUser
		this.x.Where("node_id=?",node.Id).Find(&user)
		for _, v := range user {
			v.SupNodeId = 0
			v.SupVoteNum = 0
			v.VoteNum = 0
			v.SupProfit = 0
			v.NodeId = newNodeId
			_, err := this.s.Where("user_id=?",v.UserId).Cols("sup_node_id","sup_vote_num","vote_num","sup_profit","node_id").Update(&v)
			if err!=nil {
				logs.Info("错误11=",err)
			}
			f := queryDongjie(v.VoteAddr)
			if f>=v.VoteNum {
				jiedong(v.VoteAddr,v.VoteNum)
			}else{
				jiedong(v.VoteAddr,f)
				logs.Info("出现小于",v.VoteAddr,v.VoteNum)
			}

		}
		node.Status=voteEnums.VoteStatus_succ
		_, err22 := this.s.Where("id=?",node.Id).Update(&node)
		if err22!=nil {
			logs.Info("错误22=",err22,node.Id)
		}
		_, err33 := this.s.Insert(&newNode)
		if err33!=nil {
			logs.Info("错误33=",err33)
		}
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

//超级节点替换
func (this *AdminController)SupNodeLose(){
	//进行中的超级节点
	var oldSN []model.VNodeInfo
	dbUtil.Engine.Where("status=? and stage=? and end_time=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj,"2020-09-23").Desc("total_amount").Find(&oldSN)
	for _, node:= range oldSN {
		var user []model.VUser
		this.x.Where("sup_node_id=?",node.Id).Find(&user)
		for _, v := range user {
			f := queryDongjie(v.VoteAddr)
			if f>=v.SupVoteNum {
				jiedong(v.VoteAddr,v.SupVoteNum)
			}else{
				jiedong(v.VoteAddr,f)
			}
		}
		node.Status=voteEnums.VoteStatus_succ
		_, err222 := this.s.Where("id=?",node.Id).Update(&node)
		if err222!=nil {
			logs.Info("错误222=",err222,node.Id)
		}
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

//节点复投
func (this *AdminController)SetNodeAgain(){
	nodeId,_ := this.GetInt64("nodeId")
	var nodeInfo model.VNodeInfo
	this.x.Id(nodeId).Get(&nodeInfo)
	var user model.VUser
	this.x.Id(nodeInfo.UserId).Get(&user)
	var newNodeInfo model.VNodeInfo
	this.x.Id(user.NodeId).Get(&newNodeInfo)
	user.VoteNum = user.VoteNum+nodeInfo.LockAmount
	i, errs := this.s.Where("user_id=?",user.UserId).Update(&user)
	if errs!=nil {
		logs.Info("个人更新锁仓失败,err=",errs,user.UserId,nodeInfo.LockAmount)
		TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
		return
	}
	newNodeInfo.TotalAmount = newNodeInfo.TotalAmount+nodeInfo.LockAmount
	newNodeInfo.LockAmount = nodeInfo.LockAmount
	ta,er:=this.s.Where("id=?",newNodeInfo.Id).Update(&newNodeInfo)
	if er!=nil {
		logs.Info("节点更新锁仓失败,err=",er,newNodeInfo.Id,nodeInfo.LockAmount)
		TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
		return
	}
	s, _ := nodeUtil.NewSnowflake(0)
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	voteBill := model.VVoteBill{
		Id:s.Generate(),
		UserId:user.UserId,
		UserPhone:user.UserPhone,
		UserEmail:user.UserEmail,
		UserOther:user.UserOther,
		NodeName:newNodeInfo.Name,
		NodeId:newNodeInfo.Id,
		NodeArea:newNodeInfo.Area,
		NodeImage:newNodeInfo.Image,
		Amount:nodeInfo.LockAmount,
		IssueNum:newNodeInfo.IssueNum,
		BillType:voteEnums.Bill_TYPE_TP,
		AmountType:voteEnums.Bill_TYPE_TP,
		CreateTime:t,
	}
	n, _ := this.s.Insert(&voteBill)
	if i !=1 || n!=1 || ta!=1{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("voteErr"))
		return
	}
	b := ktoApi.FrozenBalance(user.VoteAddr, nodeInfo.LockAmount)
	//bf :=ktoApi.FrozenBalanceOther(user.VoteAddr,amount)
	if !(b) {
		logs.Info("锁仓失败=",b,user.VoteAddr,nodeInfo.LockAmount)
		TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
		return
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)ChekUserFrooz(){
	var user []model.VUser
	dbUtil.Engine.Where("vote_addr!='-' ").Find(&user)
	for _, v := range user {
		//fmt.Println(k)
		var t float64
		if v.NodeId==v.SupNodeId {
			t=v.SupVoteNum
		}else{
			t=v.SupVoteNum+v.VoteNum
		}
		f := queryDongjie(v.VoteAddr)
		var i int
		if f>t {
			i = 1
			if t==0 {
				s := ktoApi.GetFrozeAllAmount(v.VoteAddr)
				b := ktoApi.UnlockBalanceAllAmount(v.VoteAddr,s )
				fmt.Println("all b=",b)
				//jiedong(v.VoteAddr,f)
			}
			logs.Info("i=",i,"f=",f,"tn=",t,"sn=",v.SupVoteNum,"vn=",v.VoteNum,"userId=",v.UserId,"addr=",v.VoteAddr)
			/*if f-t>3 {
				//split := nodeUtil.Fload64Split(f - t)
				//v2, _ := strconv.ParseFloat(split, 64)
				jiedong(v.VoteAddr,f - t)
			}*/

		}else if f==t{
			i = 0
			//logs.Info("i=",i,"f=",f,"tn=",t,"sn=",v.SupVoteNum,"vn=",v.VoteNum,"userId=",v.UserId,"addr=",v.VoteAddr)
		}else{
			i = -1
			/*if f==0 {
				b := ktoApi.FrozenBalance(v.VoteAddr, t)
				logs.Info("冻结状态:",b)
			}*/
			logs.Info("i=",i,"f=",f,"tn=",t,"sn=",v.SupVoteNum,"vn=",v.VoteNum,"userId=",v.UserId,"addr=",v.VoteAddr)
		}

	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)ChekAddrRpcFrooz(){
	var user []model.VUser
	dbUtil.Engine.Where("vote_addr!='-' ").Find(&user)
	for k, v := range user {
		fmt.Println(k)
		f := queryDongjie(v.VoteAddr)
		i1, i2, i3, i4 := queryOtherDongjie(v.VoteAddr)
		b := f == i1&& i1== i2 &&i2== i3&&i3 == i4
		if !b {
			logs.Info(v.VoteAddr,f,i1,i2,i3,i4)
			/*if f!=0 && i1!=0 {
				if f>i1 {
					b:=otherDongjie(v.VoteAddr,f-i1)
					logs.Info("otherJieDong=",b)
				}else{
					logs.Info("有问题")
				}
			}*/
		}
	}
}

func otherDongjie(addr string,amount float64)bool{
	b := ktoApi.FrozenBalanceOther(addr, amount)
	return b
}

func otherJieDong(addr string,amount float64)bool{
	//b := ktoApi.UnlockBalanceOther(addr, amount)
	return false
}

func queryDongjie(addr string)float64{
	froze := ktoApi.GetFroze(addr)
	v2, _ := strconv.ParseFloat(froze, 64)
	//logs.Info("froze=",v2,"addr=",addr)
	return v2
}

func queryOtherDongjie(addr string)(float64,float64,float64,float64){
	/*f1,f2,f3,f4 := ktoApi.GetOtherFroze(addr)

	v1, _ := strconv.ParseFloat(f1, 64)
	v2, _ := strconv.ParseFloat(f2, 64)
	v3, _ := strconv.ParseFloat(f3, 64)
	v4, _ := strconv.ParseFloat(f4, 64)*/
	//logs.Info("froze=",v2,"addr=",addr)
	return 0,0,0,0
}

func jiedong(addr string,amount float64){
	b := ktoApi.UnlockBalance(addr, amount)
	logs.Info("b=",b,"addr=",addr,"amount=",amount)
}

func (this *AdminController)TestImage(){

	/*var user []model.VUser
	dbUtil.Engine.Where("vote_num>0 or sup_vote_num>0 ").Find(&user)
	for _, v := range user {
		s, _, _, _ := ktoApi.GetOtherFroze(v.VoteAddr)
		//m,_ :=strconv.ParseFloat(s,10)
		if s>0 {
			b := ktoApi.UnlockBalanceOther(v.VoteAddr,s )
			if !b {
				fmt.Println("其他=",b,s,v.VoteAddr)
			}
		}
	}*/
	filename, err := ExportExcel()
	fmt.Println("err=",err)
	this.Ctx.Output.Download(filename,"重命名.xlsx")
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}
//start ,end string,id int,price float64
func (this *AdminController)SysBufa(){
	/*start := this.GetString("start")
	start1 := this.GetString("start1")
	end := this.GetString("end")
	end1 := this.GetString("end1")
	id,_ := this.GetInt("id")*/
	//price,_ := this.GetFloat("price")
	//nodeTask.PowBufa(start,end,id,price)
	//nodeTask.NodeBufa(start,start1,end,end1,id)

	//****************************
	/*f := 17081.5056
	var ndb model.VNodeDataBill
	ndb.KtoPrice = 12.01
	ndb.MacAddrAmount = f
	ttx:=[]string{}
	var userErr []model.UserErrTx
	list,ueList := nodeTask.NodeRelease(f, ndb, ttx,userErr)
	fmt.Println(len(list),"===",len(ueList))
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))*/
	//************************************
	sw := new(model.SWallet)
	sysId := 36930000000000
	dbUtil.Engine.Id(sysId).Get(sw)
	key := []byte(strconv.FormatInt(sw.Id, 16) + voteEnums.AES_CODE)
	p, _ := base64.StdEncoding.DecodeString(sw.KtoPowPri)
	result, _ := nodeUtil.AesDecrypt(p, key)
	priKtoPow :=string(result)
	var tl []models.WWalletTransfer
	//txList:=make([]string,11)
	txList1:=make([]string,5)
	//var succList []string
	//this.x.Table("w_wallet_transfer").Where("status=? and from_addr=? and create_time>?","FAIL","Kto4bDHCt85cjb9xC1KExvfx79rKfSgpwMXf8YDeVfqN1S8","2020-07-29 00:00:00").Find(&tl)

	sql := `
SELECT * FROM w_wallet_transfer WHERE from_addr = 'Kto4bDHCt85cjb9xC1KExvfx79rKfSgpwMXf8YDeVfqN1S8' AND STATUS='FAIL'
AND create_time>'2020-07-29 00:00:00' AND to_addr NOT IN (SELECT to_addr FROM w_wallet_transfer WHERE from_addr = 'Kto4bDHCt85cjb9xC1KExvfx79rKfSgpwMXf8YDeVfqN1S8' AND STATUS='SUCCESS'
AND create_time>'2020-07-29 00:00:00' )`
	dbUtil.Engine.SQL(sql).Find(&tl)
	var nonce uint64 = 0
	for k, v := range tl {
		if k%6 == 5{
			var i =0
			for _, t := range txList1 {
			Loop22:
				if !dbUtil.ExistHash(t, true) && i < 10 {
					time.Sleep(time.Second)
					i++
					fmt.Println("i=",i)
					goto Loop22
				}
			}
			txList1 = []string{}
			nonce = 0
		}
		m := ktoApi.ToWei(v.Amount, 11)
		tx,non, err := ktoApi.SendTradeKto(sw.KtoPowAddr,v.ToAddr, priKtoPow,m.Uint64(),nonce)
		if err != nil {
			logs.Info("l=", k, "用户地址=", v.ToAddr, "数量=",v.Amount," tx=", tx)
			continue
		}
		dbUtil.SetHash(tx, false)
		nonce = non
		//list = append(list,tx)
		txList1 = append(txList1,tx)
	}
	fmt.Println("完成=======")
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)Jiedo(){
	addr := this.GetString("addr")
	amount,_ := this.GetFloat("amount")
	t, _ := this.GetInt("t")
	if t==1 {
		b := ktoApi.UnlockBalanceOther(addr, 0)
		fmt.Println("其他=",b)
	}else if t==0 {
		s := ktoApi.GetFrozeAllAmount(addr)
		//m,_ :=strconv.ParseFloat(s,10)
		b := ktoApi.UnlockBalance(addr,amount)
		fmt.Println("其他=",b,s)
	}else if t>30 {
		b := ktoApi.UnlockOneBalanceOther(addr, amount,t)
		fmt.Println("b=",b)
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func (this *AdminController)JiedoAllAmount(){
	addr := this.GetString("addr")
	s := ktoApi.GetFrozeAllAmount(addr)

	b := ktoApi.UnlockBalanceAllAmount(addr,s )
	fmt.Println("其他=",b,s)
	TxResponse(this.s, this.Controller,"完成")
	return
}

func ExportExcel() (filename string, err error) {

	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell

	var node []model.VNodeInfo
	dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj).Find(&node)

	file = xlsx.NewFile()
	for _, v := range node {
		var user []model.VUser
		dbUtil.Engine.Where("sup_node_id=?",v.Id).Find(&user)
		sheet, _ = file.AddSheet(v.Name)
		row = sheet.AddRow()
		cell = row.AddCell()
		cell.Value = "手机号"
		cell = row.AddCell()
		cell.Value = "邮箱"
		cell = row.AddCell()
		cell.Value = "其他"
		cell = row.AddCell()
		cell.Value = "投票数量"
		cell = row.AddCell()
		cell.Value = "原力算力"
		cell = row.AddCell()
		cell.Value = "投票当月收益"
		cell = row.AddCell()
		cell.Value = "历史总收益"
		cell = row.AddCell()
		cell.Value = "地址"
		cell = row.AddCell()
		cell.Value = "是否矿主"
		cell = row.AddCell()
		cell.Value = "到期日期"
		cell = row.AddCell()
		cell.Value = "总票数:="+func()string{
			return fmt.Sprint(v.TotalAmount)
		}()
		for _, u := range user {
			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = u.UserPhone
			cell = row.AddCell()
			cell.Value = u.UserEmail
			cell = row.AddCell()
			cell.Value = u.UserOther
			cell = row.AddCell()
			cell.Value = func()string{
				return fmt.Sprint(u.SupVoteNum)
			}()
			cell = row.AddCell()
			cell.Value = decimal.NewFromFloat(u.PowPrice+u.LockPrice).Round(4).String()
			cell = row.AddCell()
			cell.Value = func()string{
				return fmt.Sprint(u.SupProfit)
			}()
			cell = row.AddCell()
			cell.Value = func()string{
				return fmt.Sprint(u.TotalProfit)
			}()
			cell = row.AddCell()
			cell.Value = u.VoteAddr
			cell = row.AddCell()
			cell.Value = func()string{
				if u.IsMac == 1{
					return "是"
				}else{
					return "否"
				}

			}()
			cell = row.AddCell()
			cell.Value = v.EndTime
		}

		if !utils.FileExists("logs") {
			os.MkdirAll("logs", os.ModePerm)
		}

	}
	filename = "logs/" + cast.ToString(time.Now().Unix()) + ".xlsx"
	err = file.Save(filename)
	return filename, err
}

func (this *AdminController)ExportSql(){
	file := "D:/mysqlBf/"+time.Now().Format("2006-01-02 15-04-05")+".sql"
	err := this.x.DumpAllToFile(file)
	fmt.Println("err=",err)
	fileZip := "D:/mysqlBf/"+time.Now().Format("2006-01-02 15-04-05")+".zip"
	zipfile, err1 :=os.Create(fileZip)
	fmt.Println("err1=",err1)
	archive := zip.NewWriter(zipfile)
	f, _ := os.Open(file)
	info, e1 := f.Stat()
	if e1 != nil {
		fmt.Println("e1=",e1)
	}
	header, e2 := zip.FileInfoHeader(info)
	//header.Name = header.Name
	header.Method = zip.Deflate
	if e2 != nil {
		fmt.Println("e2=",e2)
	}
	writer, err3 := archive.CreateHeader(header)
	if err3 != nil {
		fmt.Println("err3=",err3)
	}
	_, err4 := io.Copy(writer, f)
	fmt.Println("err4=",err4)
	f.Close()
	archive.Close()
	zipfile.Close()
	m := gomail.NewMessage()
	t := "787293518@qq.com"
	ff,p :="wallet@ktoken.ws","Ktoken88888"
	m.SetHeader("From", ff)
	m.SetHeader("To", t)
	m.SetHeader("Subject", "验证码")
	m.SetBody("text/html", "数据备份")
	m.Attach(fileZip)
	d := gomail.NewDialer("smtp.qiye.aliyun.com", 465, ff, p)
	if err := d.DialAndSend(m); err != nil {
		fmt.Println("邮箱错误=",err)
	}

	TxResponse(this.s, this.Controller,"完成")
	return
}

