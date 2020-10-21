package controller

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/shopspring/decimal"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"wallet-token/models"
	"wallet-vote/dbUtil"
	"wallet-vote/ktoApi"
	"wallet-vote/model"
	"wallet-vote/nodeUtil"
	"wallet-vote/voteEnums"
)

type VoteController struct {
	baseController
}

//获取节点
func (this *VoteController)GetNode(){
	stage,err := this.GetInt("stage")
	nodeId,err := this.GetInt64("nodeId")
	if err != nil {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("paraErr"))
		return
	}
	where := this.x.Where("Status=?",voteEnums.VoteStatus_ing)
	whereC := this.x.Where("Status=?",voteEnums.VoteStatus_ing)
	if stage != 0 {
		where.And("Stage=?",stage)
	}
	var votes []model.VNodeInfo
	where.Desc("total_amount").Find(&votes)
	var voteC model.VNodeInfo
	tv, _ := whereC.And("Stage=?", voteEnums.VoteStage_cj).Sum(&voteC, "total_amount")
	type userInfo struct {
		NodeId int64
		SupNodeId int64
		VoteNum decimal.Decimal
		SupVoteNum decimal.Decimal
		SupProfit decimal.Decimal
	}
	type votesInfo struct {
		NowTime int64
		IssueNum int
		EndTime int64
		Tp float64
		TotalVote decimal.Decimal
		Top interface{}
		UserInfo userInfo
		Votes []model.VNodeInfo
	}
	user := new(model.VUser)
	this.x.Id(this.User.Id).Get(user)
	var top interface{} = voteEnums.VALUEDEFAULT
	if user.VoteArea!=voteEnums.VALUEDEFAULT {
		for k, v := range votes {
			if v.Id == user.NodeId {
				top = k+1
				break
			}
		}
	}
	var t,e int64
	var i int
	if votes!=nil {
		tm := time.Now()
		t = tm.Unix()
		if nodeId !=0{
			var nodeInfo model.VNodeInfo
			this.x.Id(nodeId).Get(&nodeInfo)
			endTime, _ := time.Parse("2006-01-02", nodeInfo.EndTime)
			e = endTime.Unix()-28800
			i = nodeInfo.IssueNum
		}else{
			endTime, _ := time.Parse("2006-01-02", votes[0].EndTime)
			e = endTime.Unix()-28800
			i = votes[0].IssueNum
		}
	}else{
		votes = make([]model.VNodeInfo,0)
	}
	format := time.Now().Format("2006-01-02")
	te, _ := time.Parse("2006-01-02", format)
	tps := time.Now().Unix() - te.Unix() + 28800
	sysId := 36930000000000
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	fs := strconv.FormatInt(tps, 10)
	f,_ := strconv.ParseFloat(fs,64)
	tp, _ := decimal.NewFromFloat(f * 0.198 * (1 - sw.MacRatio)).RoundToEnd(4).Float64()
	ui := userInfo{user.NodeId,user.SupNodeId,decimal.NewFromFloat(user.VoteNum).RoundToEnd(2),decimal.NewFromFloat(user.SupVoteNum).RoundToEnd(2),decimal.NewFromFloat(user.SupProfit).RoundToEnd(2)}
	v :=votesInfo{t,i,e,tp,decimal.NewFromFloat(tv).RoundToEnd(2),top,ui,votes}
	TxObjectResponse(this.s, this.Controller,v)
	return
}

//节点投票
func (this *VoteController)NodeVote(){
	if nodeUtil.TimeSection() {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("sysLock"))
		return
	}
	amount,err := this.GetFloat("amount")
	pwd := this.GetString("pwd")
	if err!=nil {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("paraErr"))
		return
	}
	user := &model.VUser{}
	_, err = this.x.Id(this.User.Id).Get(user)
	fmt.Println("err=",err)
	if amount<=0 {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("voteAmountErr"))
		return
	}
	x := int64(amount)
	y := float64(x)
	if amount!=y {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("voteAmountInt"))
		return
	}
	vinto := &model.VNodeInfo{}
	this.x.Id(user.NodeId).Get(vinto)
	if vinto.Status==voteEnums.VoteStatus_ing && vinto.Stage == voteEnums.VoteStage_hx{
		wb := new(models.WWalletBase)
		this.x.Id(user.WalletId).Get(wb)
		if nodeUtil.Md5(pwd)!=wb.Password{
			TxErrMsgResponse(this.s, this.Controller, this.Tr("payPwdErr"))
			return
		}


		balance := ktoApi.GetAbleBalance(user.VoteAddr)
		if b, e := strconv.ParseFloat(balance, 10);e!=nil{
			if b<amount{
				TxErrMsgResponse(this.s, this.Controller, this.Tr("amountLow"))
				return
			}
		}
		user.VoteNum = user.VoteNum+amount
		i, errs := this.s.Where("user_id=?",user.UserId).Update(user)
		if errs!=nil {
			logs.Info("个人更新锁仓失败,err=",errs,user.UserId,amount)
			TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
			return
		}
		if user.UserId == vinto.Id {
			vinto.MacAmount += amount
		}
		vinto.TotalAmount += amount
		ta,er:=this.s.Where("id=?",vinto.Id).Update(vinto)
		if er!=nil {
			logs.Info("节点更新锁仓失败,err=",er,vinto.Id,amount)
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
			NodeName:vinto.Name,
			NodeId:vinto.Id,
			NodeArea:vinto.Area,
			NodeImage:vinto.Image,
			Amount:amount,
			IssueNum:vinto.IssueNum,
			BillType:voteEnums.Bill_TYPE_TP,
			AmountType:voteEnums.Bill_TYPE_TP,
			CreateTime:t,
		}
		n, _ := this.s.Insert(&voteBill)
		if i !=1 || n!=1 || ta!=1{
			TxErrMsgResponse(this.s, this.Controller, this.Tr("voteErr"))
			return
		}
		b := ktoApi.FrozenBalance(user.VoteAddr, amount)
		//bf :=ktoApi.FrozenBalanceOther(user.VoteAddr,amount)
		if !(b) {
			logs.Info("锁仓失败=",b,user.VoteAddr,amount)
			TxErrMsgResponse(this.s, this.Controller, this.Tr("lockBalanceErr"))
			return
		}
	}else{
		TxErrMsgResponse(this.s, this.Controller, this.Tr("nodeVoteErr"))
		return
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

//节点流水
func (this *VoteController)GetBill(){
	billType,err := this.GetInt("billType")
	s,_ :=this.GetInt("size")
	l,_ :=this.GetInt("limit")
	if err!=nil {
		TxErrMsgResponse(this.s, this.Controller, this.Tr("paraErr"))
		return
	}
	var bills []model.VVoteBill
	bill :=new(model.VVoteBill)

	where := this.x.Where("user_id=?", this.User.Id)
	whereC := this.x.Where("user_id=?", this.User.Id)
	whereT := this.x.Where("user_id=?", this.User.Id)
	if billType == voteEnums.Bill_TYPE_YL {
		where.Where("bill_type!=?",voteEnums.Bill_TYPE_TP)
		whereC.Where("bill_type!=?",voteEnums.Bill_TYPE_TP)
		whereT.Where("bill_type!=?",voteEnums.Bill_TYPE_TP)
	}else {
		where.Where("bill_type=?",billType)
		whereC.Where("bill_type=?",billType)
		whereT.Where("bill_type=?",billType)
	}
	where.Limit(l,s*l).Desc("create_time").Find(&bills)

	t, _ := whereT.Sum(bill, "amount")
	count,_ := whereC.Count(bill)
	if bills ==nil {
		bills = make([]model.VVoteBill,0)
	}
	p :=models.Page{Limit:l,Size:s,Total:count,TotalAmount:t,DataList:bills}
	ObjectResponse(this.s,this.Controller,p)
	return
}

func (this *VoteController)SupNodeInfo(){
	type nodeInfo struct {
		Price string
		Tps int
		Total decimal.Decimal
		Hide int64
		NodeId int
	}
	value := dbUtil.GetValue(voteEnums.KTOPRICE,1)
	hide := dbUtil.GetValue(voteEnums.NOW_KTO_BLOCK,2)
	mad := dbUtil.GetValue(voteEnums.NOW_MACHINE_ADDR,3)
	getMachine := dbUtil.GetMachine(mad.(string))
	var mid int
	if getMachine!=nil {
		mid = getMachine.(model.VMachine).Id
	}
	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(300) + 1800
	i :=nodeInfo{}
	i.Price = strings.Split(value.(string),"[")[1]
	i.Tps = r
	i.Hide = hide.(int64)
	f := float64(hide.(int64))
	t, _ := strconv.ParseFloat((decimal.NewFromFloat(0.45).Mul(decimal.NewFromFloat(f))).String(), 10)
	i.Total = decimal.NewFromFloat(t).RoundToEnd(2)
	i.NodeId = mid
	ObjectResponse(this.s,this.Controller,i)
	return
}

