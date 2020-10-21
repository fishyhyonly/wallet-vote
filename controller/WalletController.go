package controller

import (
	"encoding/base64"
	"fmt"
	"strconv"
	. "wallet-vote/ktoApi"
	"wallet-vote/model"
	"wallet-vote/nodeUtil"
	"wallet-vote/voteEnums"
)

type WalletController struct {
	baseController
}

//获取可用余额
func (this *WalletController)GetAbleBalance(){
	user := new(model.VUser)
	this.x.Id(this.User.Id).Get(user)
	balance := GetAbleBalance(user.VoteAddr)
	ObjectResponse(this.s,this.Controller,balance)
	return
}

func (this *WalletController)Testaa(){
	addr := this.GetString("addr")
	//amount,_ := this.GetFloat("amount")
	b:= GetAbleBalance(addr)
	fmt.Println("可用=",b)
	froze := GetFroze(addr)
	froze1, froze2, froze3, froze4 := GetOtherFroze(addr)
	fmt.Println("froze=",froze,"其他=",froze1, froze2, froze3, froze4)
	//b := ktoApi.FrozenBalance(addr, amount)
	//fmt.Println("b=",b)
	ObjectResponse(this.s,this.Controller,froze)
	return
}

func (this *WalletController)TestDongjie(){
	addr := this.GetString("addr")
	amount,_ := this.GetFloat("amount")
	t,_ := this.GetInt("t")
	if t==0 {
		b := FrozenBalance(addr, amount)
		fmt.Println("b=",b)
	}else if t==1 {
		other := FrozenBalanceOther(addr, amount)
		fmt.Println("other=",other)
	}else if t>30 {
		other := FrozenOneBalanceOther(addr, amount,t)
		fmt.Println("other=",other)
	}
	TxResponse(this.s, this.Controller,this.Tr("op_suc"))
	return
}

func getPriAll(id int64, decPri string) (pri string) {
	key := []byte(strconv.FormatInt(int64(id), 16) + voteEnums.AES_CODE)
	p, _ := base64.StdEncoding.DecodeString(decPri)
	result, _ := nodeUtil.AesDecrypt(p, key)
	pri = string(result)
	return
}

