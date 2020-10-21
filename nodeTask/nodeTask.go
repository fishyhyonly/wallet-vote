package nodeTask

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/toolbox"
	"github.com/go-gomail/gomail"
	"github.com/shopspring/decimal"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"wallet-token/enums"
	"wallet-token/models"
	"wallet-vote/dbUtil"
	"wallet-vote/ktoApi"
	"wallet-vote/model"
	"wallet-vote/nodeUtil"
	"wallet-vote/voteEnums"
)

const sysId = 36930000000000

func init() {
	fmt.Println("试试aaaa")
	tk1 := toolbox.NewTask("myTask1", "0 0 0 * * 0-6", func()error{
		fmt.Println("任务1")
		f,b,s,ue := PowRelease()
		list,ueList := NodeRelease(f, b, s,ue)
		time.Sleep(time.Second*20)
		checkTxList(list,ueList)
		return nil
	})

	tk2 := toolbox.NewTask("myTask2", "0 0 03 * * 0-6", func()error{
		fmt.Println("任务2")
		exportSql()
		logs.Info("备份完成")
		return nil
	})

	toolbox.AddTask("myTask1", tk1)
	toolbox.AddTask("myTask2", tk2)
	toolbox.StartTask()
	//defer toolbox.StopTask()
}

//原力算力分奖励
func PowRelease() (float64,model.VNodeDataBill,[]string,[]model.UserErrTx){
	fmt.Println("进来了")
	ttx:=[]string{}
	var userErr []model.UserErrTx
	var ndb model.VNodeDataBill
	value := dbUtil.GetValue(voteEnums.KTOPRICE,1)
	price,_ := strconv.ParseFloat(strings.Split(value.(string),"[")[1],10)
	ndb.KtoPrice = price
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	priMac := getPri(sw.Id, sw.KtoMacPri)
	//获取矿机币余额
	balance := ktoApi.GetAbleBalance(sw.KtoMacAddr)
	b, e := strconv.ParseFloat(balance, 10)
	if e != nil{
		ndb.MacAddrAmount = -1
		logs.Info("aa=",e)
		return -1,ndb,ttx,userErr
	}else{
		if  b < 1 {
			ndb.MacAddrAmount = b
			return b-sw.PowAmount,ndb,ttx,userErr
		}
	}
	ndb.MacAddrAmount = b
	logs.Info("矿机币余额=",b)
	if sw.PowAmount == 0 {
		return b,ndb,ttx,userErr
	}
	money := ktoApi.ToWei(sw.PowAmount, 11)
	//矿机币转移原力分发地址
	//mn :=sw.MacNonce+1
	hash, _,err := ktoApi.SendTradeKto(sw.KtoMacAddr, sw.KtoPowAddr, priMac, money.Uint64(),0)
	if err != nil {
		logs.Info("矿机币转移算力分发地址=",e)
		return b-sw.PowAmount,ndb,ttx,userErr
	}
	//dbUtil.AddNonce(mn,"mac_nonce")
	dbUtil.SetHash(hash, false)
	tranBill := createTranBill(sw.KtoMacAddr, sw.KtoPowAddr, hash, sw.PowAmount)
	dbUtil.Engine.Insert(&tranBill)
	bi := 0
	for bi=0;bi<8 ;bi++  {
		b, _ := ktoApi.QueryTx(hash)
		if b {
			break
		}
		time.Sleep(time.Second)
	}
	if bi==8 {
		return b-sw.PowAmount,ndb,ttx,userErr
	}
	ndb.PowAddrAmount = sw.PowAmount
	u := new(model.VUser)

	var user []model.VUser
	fp, e := dbUtil.Engine.Where("pow_price>1 and lock_price>1").Sum(u, "pow_price")
	fl, e := dbUtil.Engine.Where("pow_price>1 and lock_price>1").Sum(u, "lock_price")
	f := fp+fl
	if f==0 {
		return b-sw.PowAmount,ndb,ttx,userErr
	}
	ndb.TotalPl = f
	dbUtil.Engine.Where("lock_price>0 and pow_price>0 and vote_addr!='-'").Find(&user)
	logs.Info("分账用户数量=",len(user))
	//每个人的比例
	sratio := decimal.NewFromFloat(sw.PowAmount / f).Round(9).String()[:8]
	ratio, _ := strconv.ParseFloat( sratio, 10)
	//fmt.Println("ratio=",ratio)
	ndb.PowRatio = ratio
	priKto := getPri(sw.Id, sw.KtoPowPri)
	f1, _ := decimal.NewFromFloat(ratio * f).Round(6).Float64()
	am := ktoApi.ToWei(sw.PowAmount-f1+sw.PowCallAmount, 11)
	if sw.PowAmount-f1+sw.PowCallAmount>0 {
		if sw.KtoCalAddr!="-" {
			amtx, _,err := ktoApi.SendTradeKto(sw.KtoPowAddr, sw.KtoCalAddr, priKto, am.Uint64(),0)
			if err==nil {
				amBill := createTranBill(sw.KtoPowAddr, sw.KtoCalAddr, amtx, sw.PowAmount-f1+sw.PowCallAmount)
				dbUtil.SetHash(amtx, false)
				dbUtil.Engine.Insert(&amBill)
			}
			time.Sleep(time.Second*10)
		}
	}

	txList:=make([]string,5)
	var pn uint64 = 0
	sk :=0
	for k, v := range user {
		p := ratio * v.PowPrice
		l := ratio * v.LockPrice
		i2 := p+l
		m := ktoApi.ToWei(i2, 11)
		if k%6 == 5{
			for _, t := range txList {
				i :=0
			Loop1:
				if i < 10 {
					b, _ := ktoApi.QueryTx(t)
					if b {
						break
					}
					time.Sleep(time.Second)
					i++
					logs.Info("i=",i)
					goto Loop1
				}
			}
			txList = []string{}
			pn = 0
		}
		tx, pno,err := ktoApi.SendTradeKto(sw.KtoPowAddr, v.VoteAddr, priKto, m.Uint64(),pn)
		if err != nil {
			logs.Info("分币出现错误",v.UserId)
			logs.Info("k=", k, "用户=", v.UserId, " tx=", tx,"err=",err)
			ue := model.UserErrTx{v.UserId,v.VoteAddr,i2,1,p,l}
			userErr = append(userErr,ue)
			continue
		}
		pn = pno
		ttx = append(ttx, tx)
		txList =append(txList, tx)
		txBill := createTranBill(sw.KtoPowAddr, v.VoteAddr, tx, i2)
		dbUtil.SetHash(tx, false)
		dbUtil.Engine.Insert(&txBill)
		//减去原力算力
		v.TotalProfit += i2
		v.PowPrice -=price*p*sw.PowRatio
		v.LockPrice -=price*l*sw.MacRatio
		dbUtil.Engine.Id(v.UserId).Update(&v)
		ceateBill(v,model.VNodeInfo{},l,voteEnums.Bill_TYPE_YL,voteEnums.Bill_TYPE_YL)
		ceateBill(v,model.VNodeInfo{},p,voteEnums.Bill_TYPE_SL,voteEnums.Bill_TYPE_YL)
		sk++
	}
	logs.Info("完成分账用户数量=",sk)
	return b-sw.PowAmount,ndb,ttx,userErr
}
//节点分红
func NodeRelease(f float64,bill model.VNodeDataBill,ttx []string,ue []model.UserErrTx)([]string,[]model.UserErrTx){
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	bill.CreateTime = t
	if f<=0 {
		bill.NodeAddrAmount = f
		dbUtil.Engine.Insert(&bill)
		return ttx,ue
	}
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	var node []model.VNodeInfo
	var n model.VNodeInfo

	money := ktoApi.ToWei(f, 11)
	//矿机币转移节点分发地址
	priMac := getPri(sw.Id, sw.KtoMacPri)
	hash,_, err := ktoApi.SendTradeKto(sw.KtoMacAddr, sw.KtoNodeAddr, priMac, money.Uint64(),0)
	if err != nil {
		logs.Info("矿机币转移节点分发地址=",err)
		dbUtil.Engine.Insert(&bill)
		return ttx,ue
	}
	dbUtil.SetHash(hash, false)
	tranBill := createTranBill(sw.KtoMacAddr, sw.KtoNodeAddr, hash, f)
	dbUtil.Engine.Insert(&tranBill)
	i := 0
Loop:
	if !dbUtil.ExistHash(hash, true) && i < 20 {
		time.Sleep(time.Second)
		i++
		fmt.Println("i=",i)
		goto Loop
	}
	priKto := getPri(sw.Id, sw.KtoNodePri)
	if sw.KtoLwAddr!="-" {
		rand.Seed(time.Now().UnixNano())
		money := rand.Intn(10000)
		m :=decimal.New(int64(money),-4).Add(decimal.New(20,0))
		cmt1 := ktoApi.ToWei(m, 11)
		ctx1,_, err := ktoApi.SendTradeKto(sw.KtoNodeAddr, sw.KtoLwAddr, priKto, cmt1.Uint64(),0)
		if err == nil {
			mm,_:=m.Float64()
			ctxBill1 := createTranBill(sw.KtoNodeAddr, sw.KtoLwAddr, ctx1,mm)
			dbUtil.SetHash(ctx1, false)
			dbUtil.Engine.Insert(&ctxBill1)
			f = f-mm
		}
		time.Sleep(time.Second*10)
	}

	bill.NodeAddrAmount = f
	if !dbUtil.ExistHash(hash, true) {
		qb, _ := ktoApi.QueryTx(hash)
		if !qb {
			dbUtil.Engine.Insert(&bill)
			logs.Info("交易未完成退出1")
			return ttx,ue
		}
	}
	//所有超级节点
	dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj).Find(&node)
	//所有节点总票数
	ta, _ := dbUtil.Engine.Where("status=? and stage=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj).Sum(&n, "total_amount")
	fmt.Println("ta=",ta)
	if ta == 0 {
		bill.TotalVote = ta
		dbUtil.Engine.Insert(&bill)
		logs.Info("暂无超级节点")
		return ttx,ue
	}
	bill.TotalVote = ta
	//用户的90%
	userT,_ := decimal.NewFromFloat(f*(1-sw.MacRatio)).Round(6).Float64()
	logs.Info("userT=",userT)
	bill.TotalUserVote = userT
	//用户的比例
	userSR := decimal.NewFromFloat(userT/ta).Round(9).String()[:8]
	userR, err4 := strconv.ParseFloat( userSR, 10)
	//userR,err4 := decimal.NewFromFloat(userT/ta).Round(6).Float64()
	logs.Info("userR=",userR)
	bill.UserNodeRatio = userR
	logs.Info("err4=",err4)
	//矿主的10%
	macT,_ := decimal.NewFromFloat(f*sw.MacRatio).Round(6).Float64()
	logs.Info("macT=",macT)
	//领头人的比例
	macSR:= decimal.NewFromFloat(macT/ta).Round(9).String()[:8]
	macR,_  := strconv.ParseFloat( macSR, 10)
	//macR,_ := decimal.NewFromFloat(macT/ta).Round(6).Float64()
	logs.Info("macR=",macR)
	bill.MacNodeRatio = macR
	_, errs := dbUtil.Engine.Insert(&bill)
	logs.Info("Insert bill errs=",errs)

	ca := f - (macR*ta + userR*ta)
	if ca-0.1>0 {
		if sw.KtoCalAddr!="-" {
			cmt := ktoApi.ToWei(ca-0.1, 11)
			ctx,_, err := ktoApi.SendTradeKto(sw.KtoNodeAddr, sw.KtoCalAddr, priKto, cmt.Uint64(),0)
			if err == nil {
				ctxBill := createTranBill(sw.KtoNodeAddr, sw.KtoCalAddr, ctx, ca-0.1)
				dbUtil.SetHash(ctx, false)
				dbUtil.Engine.Insert(&ctxBill)
			}
			time.Sleep(time.Second*10)
		}
	}
	var nonce uint64 = 0
	txList:=make([]string,5)
	for k, v := range node {
		//用户分红
		var uv []model.VUser
		dbUtil.Engine.Where("sup_node_id=? and sup_vote_num>0",v.Id).Find(&uv)
		for l,e := range uv {
			var macAmount float64 = 0
			if e.IsMac == voteEnums.ISMACY {
				//矿主额外的10%
				macAmount = macR * v.TotalAmount
			}
			ut := userR * e.SupVoteNum + macAmount
			mt := ktoApi.ToWei(ut, 11)
			if k%6 == 5{
				i =0
				for _, t := range txList {
				Loop2:
					if !dbUtil.ExistHash(t, true) && i < 10 {
						time.Sleep(time.Second)
						i++
						fmt.Println("i=",i)
						goto Loop2
					}
				}
				txList = []string{}
				nonce = 0
			}
			tx,non, err := ktoApi.SendTradeKto(sw.KtoNodeAddr, e.VoteAddr, priKto, mt.Uint64(),nonce)
			if err != nil {
				logs.Info("l=", l, "用户=", e.UserId, "数量=",ut," tx=", tx)
				uen := model.UserErrTx{e.UserId,e.VoteAddr,ut,2,0,0}
				ue = append(ue,uen)
				continue
			}
			nonce = non
			ttx = append(ttx,tx)
			txList = append(txList,tx)
			txBill := createTranBill(sw.KtoNodeAddr, e.VoteAddr, tx, ut)
			dbUtil.SetHash(tx, false)
			dbUtil.Engine.Insert(&txBill)
			e.TotalProfit = e.TotalProfit+ut
			e.SupProfit = e.SupProfit+ut
			dbUtil.Engine.Id(e.UserId).Update(&e)
			ceateBill(e,v,ut,voteEnums.Bill_TYPE_SY,voteEnums.Bill_TYPE_SY)
		}
	}
	return ttx,ue
}

//检漏
func checkTxList(list []string,ue []model.UserErrTx){
	logs.Info("====进来检查====")
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	priKtoPow := getPri(sw.Id, sw.KtoPowPri)
	priKtoNode := getPri(sw.Id, sw.KtoNodePri)
	var i int
	var nonce uint64 = 0
	var pn uint64 = 0
	txList:=make([]string,5)
	txList1:=make([]string,5)
	value := dbUtil.GetValue(voteEnums.KTOPRICE,1)
	price,_ := strconv.ParseFloat(strings.Split(value.(string),"[")[1],10)
	for k, l := range ue{
		var vuser model.VUser
		dbUtil.Engine.Id(l.UserId).Get(&vuser)
		m := ktoApi.ToWei(l.Amount, 11)
		if l.ErrType==1 {
			if k%6 == 5{
				i =0
				for _, t := range txList {
				Loop21:
					if !dbUtil.ExistHash(t, true) && i < 10 {
						time.Sleep(time.Second)
						i++
						fmt.Println("i=",i)
						goto Loop21
					}
				}
				txList = []string{}
				pn = 0
			}
			tx, pno,err := ktoApi.SendTradeKto(sw.KtoPowAddr, l.Addr, priKtoPow, m.Uint64(),pn)
			if err != nil {
				logs.Info("捡漏原力错误,用户=",l.UserId, " tx=", tx,"err=",err)
				continue
			}
			pn = pno
			list = append(list, tx)
			txList =append(txList, tx)
			txBill := createTranBill(sw.KtoPowAddr,l.Addr, tx, l.Amount)
			dbUtil.SetHash(tx, false)
			dbUtil.Engine.Insert(&txBill)
			//减去原力算力
			vuser.TotalProfit += l.Amount
			vuser.PowPrice -=price*l.PowNum*sw.PowRatio
			vuser.LockPrice -=price*l.LockNum*sw.PowRatio
			dbUtil.Engine.Id(vuser.UserId).Update(&vuser)
			ceateBill(vuser,model.VNodeInfo{},l.LockNum,voteEnums.Bill_TYPE_YL,voteEnums.Bill_TYPE_YL)
			ceateBill(vuser,model.VNodeInfo{},l.PowNum,voteEnums.Bill_TYPE_SL,voteEnums.Bill_TYPE_YL)
		}else{
			if k%6 == 5{
				i =0
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
			m := ktoApi.ToWei(l.Amount, 11)
			tx,non, err := ktoApi.SendTradeKto(sw.KtoNodeAddr, l.Addr, priKtoNode,m.Uint64(),nonce)
			if err != nil {
				logs.Info("捡漏节点错误,用户=", l, "用户=", l.UserId, "数量=",l.Amount," tx=", tx)
				continue
			}
			nonce = non
			list = append(list,tx)
			txList1 = append(txList1,tx)
			txBill := createTranBill(sw.KtoNodeAddr,l.Addr, tx, l.Amount)
			dbUtil.SetHash(tx, false)
			dbUtil.Engine.Insert(&txBill)
			vuser.TotalProfit = vuser.TotalProfit+l.Amount
			vuser.SupProfit = vuser.SupProfit+l.Amount
			dbUtil.Engine.Id(vuser.UserId).Update(&vuser)
			var nodeInfo model.VNodeInfo
			dbUtil.Engine.Id(vuser.SupNodeId).Get(&nodeInfo)
			ceateBill(vuser,nodeInfo,l.Amount,voteEnums.Bill_TYPE_SY,voteEnums.Bill_TYPE_SY)
		}
	}
	logs.Info("list长度=",len(list),"ue长度=",len(ue))
	time.Sleep(time.Second*20)
	nonce =0
	for _, v := range list {
		if !dbUtil.ExistHash(v, true){
			qb, _ := ktoApi.QueryTx(v)
			if !qb {
				var wt models.WWalletTransfer
				dbUtil.Engine.Where("txhash = ?", v).Get(&wt)
				m := ktoApi.ToWei(wt.Amount, 11)
				var p string
				if wt.FromAddr == sw.KtoPowAddr {
					p = priKtoPow
				}else{
					p = priKtoNode
				}
				tx,_, err := ktoApi.SendTradeKto(wt.FromAddr, wt.ToAddr, p, m.Uint64(),nonce)
				if err != nil {
					logs.Info("补发错误,err=",err,"toAddr=",wt.ToAddr)
					continue
				}
				dbUtil.SetHash(tx, false)
				tranBill := createTranBill(wt.FromAddr, wt.ToAddr, tx, wt.Amount)
				dbUtil.Engine.Insert(&tranBill)
				wwt:=model.WWalletTxhash{wt.Id,wt.FromAddr,wt.ToAddr,wt.Txhash,wt.Amount,wt.CreateTime}
				dbUtil.Engine.Insert(&wwt)
				dwt :=new(models.WWalletTransfer)
				dbUtil.Engine.Id(wt.Id).Delete(dwt)
				i := 0
			Loop31:
				qb, _ := ktoApi.QueryTx(tx)
				if !qb {
					time.Sleep(time.Second)
					i++
					fmt.Println("i=",i)
					goto Loop31
				}
			}
		}
	}
	logs.Info("====检查完成====")
}

//原力算力补发
func PowBufa(start ,end string,id int,price float64){
	sql:=`SELECT * FROM v_user WHERE (lock_price>0 AND pow_price>0 AND vote_addr!='-') AND user_id NOT IN(
		SELECT user_id FROM v_vote_bill WHERE amount_type=3 AND create_time>'`+start+`' AND create_time<'`+end+`' GROUP BY user_id
	)`
	var user []model.VUser
	dbUtil.Engine.SQL(sql).Find(&user)
	fmt.Println(len(user),id,price)
	vdb := new(model.VNodeDataBill)
	dbUtil.Engine.Where("id=?",id).Get(vdb)
	fmt.Println(vdb.PowRatio)
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	priKto := getPri(sw.Id, sw.KtoPowPri)
	var pn uint64 = 0
	var ta float64
	txList:=make([]string,11)
	for k, v := range user {
		p := vdb.PowRatio * v.PowPrice
		l := vdb.PowRatio * v.LockPrice
		i2 := p+l
		logs.Info("地址=",v.VoteAddr,"数量=",i2)
		m := ktoApi.ToWei(i2, 11)
		i :=0
		if k%12 == 11{
			for _, t := range txList {
			Loop1:
				if !dbUtil.ExistHash(t, true) && i < 20 {
					time.Sleep(time.Second)
					i++
					logs.Info("i=",i)
					goto Loop1
				}
			}
			txList = []string{}
		}
		tx,non, err := ktoApi.SendTradeKto(sw.KtoPowAddr, v.VoteAddr, priKto, m.Uint64(),pn)
		if err != nil {
			ceateBill(v,model.VNodeInfo{},i2,-1,-1)
			logs.Info("分币出现错误",v.UserId)
			logs.Info("k=", k, "用户=", v.UserId, " tx=", tx,"err=",err)
			continue
		}
		pn = non
		txBill := createTranBill(sw.KtoPowAddr, v.VoteAddr, tx, i2)
		dbUtil.SetHash(tx, false)
		dbUtil.Engine.Insert(&txBill)
		//减去原力算力
		v.TotalProfit += i2
		v.PowPrice -=price*p
		v.LockPrice -=price*l
		dbUtil.Engine.Id(v.UserId).Update(&v)
		ceateBill(v,model.VNodeInfo{},l,voteEnums.Bill_TYPE_YL,voteEnums.Bill_TYPE_YL)
		ceateBill(v,model.VNodeInfo{},p,voteEnums.Bill_TYPE_SL,voteEnums.Bill_TYPE_YL)
		ta += i2
	}
	logs.Info("总共=",ta)
}

func NodeBufa(start ,start1,end ,end1 string,id int){
	vdb := new(model.VNodeDataBill)
	dbUtil.Engine.Where("id=?",id).Get(vdb)
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	priKto := getPri(sw.Id, sw.KtoNodePri)
	sql :=`SELECT user_id FROM v_vote_bill WHERE bill_type=2 AND create_time>'`+start+`' AND create_time<'`+end+`' AND 
user_id NOT IN (
SELECT user_id FROM v_vote_bill WHERE bill_type=2 AND create_time>'`+start1+`' AND create_time<'`+end1+`'
)`
	var userId []int64
	dbUtil.Engine.SQL(sql).Find(&userId)
	var nonce uint64 = 0
	txList:=make([]string,11)
	var ta float64
	for k, v := range userId {
		user:=model.VUser{}
		fmt.Println(v)
		dbUtil.Engine.Where("user_id=?",v).Get(&user)
		ut := vdb.UserNodeRatio * user.SupVoteNum
		ta+=ut
		mt := ktoApi.ToWei(ut, 11)
		if k%12 == 11{
			i:=0
			for _, t := range txList {
			Loop2:
				if !dbUtil.ExistHash(t, true) && i < 20 {
					time.Sleep(time.Second)
					i++
					fmt.Println("i=",i)
					goto Loop2
				}
			}
			txList = []string{}
		}
		tx,non, err := ktoApi.SendTradeKto(sw.KtoNodeAddr, user.VoteAddr, priKto, mt.Uint64(),nonce)
		if err != nil {
			logs.Info("l=", k, "用户=", user.UserId, "数量=",ut," tx=", tx)
			continue
		}
		nonce = non
		txList = append(txList,tx)
		txBill := createTranBill(sw.KtoNodeAddr, user.VoteAddr, tx, ut)
		dbUtil.SetHash(tx, false)
		dbUtil.Engine.Insert(&txBill)
		user.TotalProfit = user.TotalProfit+ut
		user.SupProfit = user.SupProfit+ut
		_,eu:=dbUtil.Engine.Id(user.UserId).Update(&user)
		if eu!=nil {
			logs.Info("eu=",eu)
		}
		var nodeInfo model.VNodeInfo
		dbUtil.Engine.Id(user.SupNodeId).Get(&nodeInfo)
		ceateBill(user,nodeInfo,ut,voteEnums.Bill_TYPE_SY,voteEnums.Bill_TYPE_SY)

	}
	fmt.Println("总共=",ta)
}
//节点考核
func ExamineNode(){
	sw := new(model.SWallet)
	dbUtil.Engine.Id(sysId).Get(sw)
	tm := time.Now()
	t := tm.Format("2006-01-02")
	var oldSN []model.VNodeInfo
	var newSN []model.VNodeInfo
	var loseNode []model.VNodeInfo
	var csn model.VNodeInfo
	//当前超级节点
	dbUtil.Engine.Where("status=? and stage=? and end_time=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj,t).Desc("total_amount").Find(&oldSN)
	//统计再进行的超级节点
	c, _ := dbUtil.Engine.Where("status=? and stage=? and end_time>?", voteEnums.VoteStatus_ing, voteEnums.VoteStage_hx, t).Count(&csn)
	//前n名的候选节点
	dbUtil.Engine.Where("status=? and stage=? and end_time=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_hx,t).Desc("total_amount").Limit(sw.SupNum-int(c),0).Find(&newSN)

	//落选的候选节点
	dbUtil.Engine.Where("status=? and stage=? and end_time=?",voteEnums.VoteStatus_ing,voteEnums.VoteStage_hx,t).Desc("total_amount").Limit(sw.SupNum-int(c),100).Find(&loseNode)

	//老超级节点重置
	sql1 := "update v_node_info set status=? where status=? and stage=? and end_time=?"
	_, err := dbUtil.Engine.Exec(sql1, voteEnums.VoteStatus_succ,voteEnums.VoteStatus_ing,voteEnums.VoteStage_cj,t)
	if err != nil {
		logs.Info("sql2执行错误=",err)
	}
	//当前超级节点到期处理
	for _, v := range oldSN {
		//b := ktoApi.FrozenBalance(v.ChainAddr, v.LockAmount)
		//todo
		//产生新的候选节点
		node := createHXNode(v)
		//节点下用户全部释放锁仓
		var unLockUser []model.VUser
		dbUtil.Engine.Where("sup_node_id=? and is_mac!=?",v.Id,voteEnums.ISMACY).Find(&unLockUser)
		for _, uu := range unLockUser {
			if uu.SupVoteNum>0 {
				ub := ktoApi.UnlockBalance(uu.VoteAddr, uu.SupVoteNum)
				if !ub {
					logs.Info("用户解仓失败,id=",uu.UserId)
				}
			}
		}
		//更新用户节点
		sql := "update v_user set node_id=?,vote_num=0,sup_node_id=0,sup_vote_num=0,sup_profit=0 where sup_node_id=?"
		_, err := dbUtil.Engine.Exec(sql, node.Id, v.Id)
		if err != nil {
			logs.Info("sql执行错误=",err)
		}
		//矿主重新默认投票
		if v.LockAmount>0 {
			b := ktoApi.UnlockBalance(v.ChainAddr,v.MacAmount-v.LockAmount)
			if !b {
				logs.Info("重新锁仓失败,nodeId=",v.Id)
			}else{

			}
			var macUser model.VUser
			dbUtil.Engine.Id(node.UserId).Get(&macUser)
			macUser.VoteNum = node.LockAmount
			dbUtil.Engine.Id(macUser.UserId).Update(&macUser)
		}else{
			node.LockAmount = 0
		}
		dbUtil.Engine.Insert(&node)
	}
	//候选竞选成功
	for _, vn := range newSN {
		//ktoApi.UnlockBalance(vn.ChainAddr,vn.MacAmount-vn.LockAmount)
		//竞选成功在产生出候选节点
		nodeHx := createHXNode(vn)
		nodeHx.MacAmount = 0
		nodeHx.TotalAmount = 0
		nodeHx.LockAmount = 0
		dbUtil.Engine.Insert(&nodeHx)
		node := createCJNode(vn)
		dbUtil.Engine.Id(node.Id).Update(&node)
		sql := "update v_user set sup_node_id=?,sup_vote_num=vote_num,sup_profit=0,node_id=?,vote_num=0 where node_id=?"
		_, err := dbUtil.Engine.Exec(sql, node.Id, nodeHx.Id,node.Id)
		if err != nil {
			logs.Info("sql执行错误=",err)
		}
	}
	//节点候选失败
	/*for k, v := range loseNode {

	}*/
}

func exportSql(){
	str := time.Now().Format("2006-01-02 15-04-05")
	file := "D:/mysqlBf/"+str+".sql"
	err := dbUtil.Engine.DumpAllToFile(file)
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	fileZip := "D:/mysqlBf/"+str+".zip"
	zipfile, err :=os.Create(fileZip)
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	archive := zip.NewWriter(zipfile)
	f, err := os.Open(file)
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	info, err := f.Stat()
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	header, err := zip.FileInfoHeader(info)

	if err!=nil{
		logs.Info("err=",err)
		return
	}
	header.Method = zip.Deflate
	writer, err := archive.CreateHeader(header)
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	_, err = io.Copy(writer, f)
	if err!=nil{
		logs.Info("err=",err)
		return
	}
	f.Close()
	archive.Close()
	zipfile.Close()
	m := gomail.NewMessage()
	t := beego.AppConfig.String("to_sql")
	ff,p :="wallet@ktoken.ws","Ktoken88888"
	m.SetHeader("From", ff)
	m.SetHeader("To", t)
	m.SetHeader("Subject", "数据备份")
	m.SetBody("text/html", "数据备份")
	m.Attach(fileZip)
	d := gomail.NewDialer("smtp.qiye.aliyun.com", 465, ff, p)
	if err := d.DialAndSend(m); err != nil {
		logs.Info("邮箱发送错误=",err)
	}
}

func getPri(id int64, decPri string) (pri string) {
	key := []byte(strconv.FormatInt(int64(id), 16) + voteEnums.AES_CODE)
	p, _ := base64.StdEncoding.DecodeString(decPri)
	result, _ := nodeUtil.AesDecrypt(p, key)
	pri = string(result)
	return
}

func createTranBill(fromAddr, toAddr, hash string, amount float64) (tranBill models.WWalletTransfer) {
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	s, _ := nodeUtil.NewSnowflake(0)
	tranBill = models.WWalletTransfer{
		Id:         s.Generate(),
		FromAddr:   fromAddr,
		ToAddr:     toAddr,
		CoinCode:   enums.KTO,
		Amount:     amount,
		Fee:        0,
		FeeInfo:    "",
		FeeCoin:    enums.KTO,
		Status:     "WAIT",
		Remark:		"",
		Txhash:     hash,
		CreateTime: t,
	}
	return
}
//创建新的候选节点
func createHXNode(n model.VNodeInfo)model.VNodeInfo{
	s, _ := nodeUtil.NewSnowflake(0)
	nodeId := s.Generate()
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	n.Id = nodeId
	n.EndTime = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	n.TotalAmount = n.LockAmount
	n.MacAmount = n.LockAmount
	n.IssueNum += 1
	n.Stage = voteEnums.VoteStage_hx
	n.CreateTime = t
	return n
}

func createCJNode(n model.VNodeInfo)model.VNodeInfo{
	n.EndTime = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	n.Stage = voteEnums.VoteStage_cj
	return n
}

func ceateBill(user model.VUser,vinto model.VNodeInfo,amount float64,bt,at int){
	tm := time.Now()
	t := tm.Format("2006-01-02 15:04:05")
	s, _ := nodeUtil.NewSnowflake(0)
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
		BillType:bt,//voteEnums.Bill_TYPE_TP,
		AmountType:at,//voteEnums.Bill_TYPE_TP,
		CreateTime:t,
	}
	dbUtil.Engine.Insert(&voteBill)
}