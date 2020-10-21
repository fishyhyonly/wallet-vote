package ktoApi

import (
	"context"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"math"
	"math/big"
	"os"
	"strconv"
	"time"
	pb "wallet-token/message"
	"wallet-vote/nodeUtil"
)

var kc *grpc.ClientConn
var ktoClient pb.GreeterClient
var ktoClient1 pb.GreeterClient
var ktoClient2 pb.GreeterClient
var ktoClient3 pb.GreeterClient
var ktoClient4 pb.GreeterClient
var ktoRpc = beego.AppConfig.String("ktoRpc")
func init(){
	var err error
	kc, err = grpc.Dial(ktoRpc,grpc.WithInsecure(),grpc.WithBlock())
	if err!=nil {
		logs.Info("初始化连接ktoRpc:",err)
		os.Exit(1)
	}
	ktoClient = pb.NewGreeterClient(kc)
}

func init(){
	time.Sleep(time.Second * 5)
	go func() {
		for {
		number := new(pb.ReqMaxBlockNumber)
		_,err := ktoClient.GetMaxBlockNumber(context.Background(), number)
		if err!=nil {
			logs.Info("断开重连2",err)
			newktoClient()
			continue
		}
		time.Sleep(time.Second * 1)
	}}()
}

func createClient(i int){
	switch i {
	case 1:
		kr:=beego.AppConfig.String("ktoRpc1")
		kc, _ := grpc.Dial(kr,grpc.WithInsecure(),grpc.WithBlock())
		ktoClient1 = pb.NewGreeterClient(kc)
		break
	case 2:
		kr:=beego.AppConfig.String("ktoRpc2")
		kc, _ := grpc.Dial(kr,grpc.WithInsecure(),grpc.WithBlock())
		ktoClient2 = pb.NewGreeterClient(kc)
		break
	case 3:
		kr:=beego.AppConfig.String("ktoRpc3")
		kc, _ := grpc.Dial(kr,grpc.WithInsecure(),grpc.WithBlock())
		ktoClient3 = pb.NewGreeterClient(kc)
		break
	case 4:
		kr:=beego.AppConfig.String("ktoRpc4")
		kc, _ := grpc.Dial(kr,grpc.WithInsecure(),grpc.WithBlock())
		ktoClient4 = pb.NewGreeterClient(kc)
		break
	}
}

func newktoClient(){
	kc.Close()
	logs.Info("进来重连")
	var err error
	kc, err = grpc.Dial(ktoRpc,grpc.WithInsecure(),grpc.WithBlock())
	if err!=nil {
		logs.Info(err)
		os.Exit(1)
	}else{
		logs.Info("重连成功")
	}
	ktoClient = pb.NewGreeterClient(kc)
}

//获取可用余额
func GetAbleBalance(addr string)string{
	decode :=nodeUtil.Decode(addr[3:])
	if len(decode) != 32 {
		return "0"
	}
	repBalance := new(pb.ReqBalance)
	froBalance := new(pb.ReqFrozenAssets)
	repBalance.Address = addr
	froBalance.Addr = addr
	b, err := ktoClient.GetBalance(context.Background(), repBalance)
	if err!=nil {
		logs.Info("err=",err)
		return "0"
	}
	f, err := ktoClient.GetFrozenAssets(context.Background(), froBalance)
	if err != nil {
		return "0"
	}
	//fmt.Println("Balnce=",b.Balnce)
	c :=b.Balnce-f.FrozenAssets
	fbalance := new(big.Float)
	fbalance.SetString(strconv.FormatUint(c,10))
	ktoValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(11)))

	sum := ktoValue.String()
	v2, err := strconv.ParseFloat(sum, 64)
	s := nodeUtil.Fload64Split(v2)
	return s
	/*if v2==0 {
		return "0"
	}else{
		round := decimal.NewFromFloat(v2).Round(8).String()
		m := round[:len(round)-2]
		return m
	}*/
	//num := fmt.Sprintf("%0.6f", v2) //float取后六位
	//return num
}

//锁仓
func FrozenBalance(addr string,amount float64)bool{
	lock :=new(pb.ReqLockBalance)
	lock.Address = addr
	lock.Amount = ToWei(amount, 11).Uint64()
	b,_:= ktoClient.SetLockBalance(context.Background(), lock)
	return b.Status
}

//其他锁仓
func FrozenBalanceOther(addr string,amount float64)bool{
	lock :=new(pb.ReqLockBalance)
	lock.Address = addr
	lock.Amount = ToWei(amount, 11).Uint64()
	var b1,b2,b3,b4 bool
	for a:=0;a<5 ;a++  {
		if ktoClient1==nil {
			createClient(1)
			rl1,_:= ktoClient1.SetLockBalance(context.Background(), lock)
			b1 = rl1.Status
		}else{
			rl1,_:= ktoClient1.SetLockBalance(context.Background(), lock)
			b1 = rl1.Status
		}
		if b1 {
			break
		}
	}

	for a:=0;a<5 ;a++  {
		if ktoClient2==nil {
			createClient(2)
			rl2,_:= ktoClient2.SetLockBalance(context.Background(), lock)
			b2 = rl2.Status
		}else{
			rl2,_:= ktoClient2.SetLockBalance(context.Background(), lock)
			b2 = rl2.Status
		}
		if b2 {
			break
		}
	}

	for a:=0;a<5 ;a++  {
		if ktoClient3==nil {
			createClient(3)
			rl3,_:= ktoClient3.SetLockBalance(context.Background(), lock)
			b3 = rl3.Status
		}else{
			rl3,_:= ktoClient3.SetLockBalance(context.Background(), lock)
			b3 = rl3.Status
		}
		if b3 {
			break
		}
	}

	for a:=0;a<5 ;a++  {
		if ktoClient4==nil {
			createClient(4)
			rl4,_:= ktoClient4.SetLockBalance(context.Background(), lock)
			b4 = rl4.Status
		}else{
			rl4,_:= ktoClient4.SetLockBalance(context.Background(), lock)
			b4 = rl4.Status
		}
		if b4 {
			break
		}
	}

	sb :=b1&&b2&&b3&&b4
	return sb
}

//其他单个锁仓
func FrozenOneBalanceOther(addr string,amount float64,i int)bool{
	lock :=new(pb.ReqLockBalance)
	lock.Address = addr
	lock.Amount = ToWei(amount, 11).Uint64()
	var b1 bool
	switch i {
	case 31:
		if ktoClient1==nil {
			createClient(1)
			rl1,_:= ktoClient1.SetLockBalance(context.Background(), lock)
			b1 = rl1.Status
		}else{
			rl1,_:= ktoClient1.SetLockBalance(context.Background(), lock)
			b1 = rl1.Status
		}
		break
	case 32:
		if ktoClient2==nil {
			createClient(2)
			rl2,_:= ktoClient2.SetLockBalance(context.Background(), lock)
			b1 = rl2.Status
		}else{
			rl2,_:= ktoClient2.SetLockBalance(context.Background(), lock)
			b1 = rl2.Status
		}
		break
	case 33:
		if ktoClient3==nil {
			createClient(3)
			rl3,_:= ktoClient3.SetLockBalance(context.Background(), lock)
			b1 = rl3.Status
		}else{
			rl3,_:= ktoClient3.SetLockBalance(context.Background(), lock)
			b1 = rl3.Status
		}
		break
	case 34:
		if ktoClient4==nil {
			createClient(4)
			rl4,_:= ktoClient4.SetLockBalance(context.Background(), lock)
			b1 = rl4.Status
		}else{
			rl4,_:= ktoClient4.SetLockBalance(context.Background(), lock)
			b1 = rl4.Status
		}
		break
	}
	return b1
}

//获取冻结金额
func GetFroze(addr string)string{
	froBalance := new(pb.ReqFrozenAssets)
	froBalance.Addr = addr
	f, err := ktoClient.GetFrozenAssets(context.Background(), froBalance)
	if err != nil {
		fmt.Println("err=",err)
		return "0"
	}
	c :=f.FrozenAssets
	fbalance := new(big.Float)
	fbalance.SetString(strconv.FormatUint(c,10))
	ktoValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(11)))

	sum := ktoValue.String()
	v2, err := strconv.ParseFloat(sum, 64)
	s := nodeUtil.Fload64Split(v2)
	return s
	/*if v2==0 {
		return "0"
	}else{
		round := decimal.NewFromFloat(v2).Round(8).String()
		if len(strings.Split(round,".")[1])<3{
			return round
		}else{
			m := round[:len(round)-2]
			return m
		}
	}
	num := fmt.Sprintf("%0.6f", v2) //float取后六位
	return num*/
}
func GetFrozeAllAmount(addr string)uint64{
	froBalance := new(pb.ReqFrozenAssets)
	froBalance.Addr = addr
	f, err := ktoClient.GetFrozenAssets(context.Background(), froBalance)
	if err != nil {
		fmt.Println("err=",err)
		return 0
	}
	return f.FrozenAssets
}
//获取其他服务器冻结金额
func GetOtherFroze(addr string)(uint64,uint64,uint64,uint64){
	froBalance := new(pb.ReqFrozenAssets)
	froBalance.Addr = addr
	var f1,f2,f3,f4 uint64
	if ktoClient1==nil {
		createClient(1)
		f, err := ktoClient1.GetFrozenAssets(context.Background(), froBalance)
		fmt.Println("err=",err)
		f1 = f.FrozenAssets
	}else{
		f, _ := ktoClient1.GetFrozenAssets(context.Background(), froBalance)
		f1 = f.FrozenAssets
	}
	if ktoClient2==nil {
		createClient(2)
		f, _ := ktoClient2.GetFrozenAssets(context.Background(), froBalance)
		f2 = f.FrozenAssets
	}else{
		f, _ := ktoClient2.GetFrozenAssets(context.Background(), froBalance)
		f2 = f.FrozenAssets
	}
	if ktoClient3==nil {
		createClient(3)
		f, _ := ktoClient3.GetFrozenAssets(context.Background(), froBalance)
		f3 = f.FrozenAssets
	}else{
		f, _ := ktoClient3.GetFrozenAssets(context.Background(), froBalance)
		f3 = f.FrozenAssets
	}
	if ktoClient4==nil {
		createClient(4)
		f, _ := ktoClient4.GetFrozenAssets(context.Background(), froBalance)
		f4 = f.FrozenAssets
	}else{
		f, _ := ktoClient4.GetFrozenAssets(context.Background(), froBalance)
		f4 = f.FrozenAssets
	}
	//fbalance := new(big.Float)
	var num1,num2,num3,num4 uint64
	for i:=1; i<=4;i++  {
		var c uint64
		switch i {
		case 1:
			c=f1
			break
		case 2:
			c=f2
			break
		case 3:
			c=f3
			break
		case 4:
			c=f4
			break
		}
		//fbalance.SetString(strconv.FormatUint(c,10))
		//ktoValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(11)))

		//sum := ktoValue.String()
		//v2, _ := strconv.ParseFloat(sum, 64)
		//num := sum
		//num := fmt.Sprintf("%0.6f", v2) //float取后六位
		switch i {
		case 1:
			num1=c
			break
		case 2:
			num2=c
			break
		case 3:
			num3=c
			break
		case 4:
			num4=c
			break
		}
	}
	return num1,num2,num3,num4
}
//解仓
func UnlockBalance(addr string,amount float64)bool{
	lock :=new(pb.ReqUnlockBalance)
	lock.Address = addr
	lock.Amount = ToWei(amount, 11).Uint64()
	b,_:= ktoClient.SetUnlockBalance(context.Background(), lock)
	return b.Status
}

func UnlockBalanceAllAmount(addr string,amount uint64)bool{
	lock :=new(pb.ReqUnlockBalance)
	lock.Address = addr
	lock.Amount =amount
	b,_:= ktoClient.SetUnlockBalance(context.Background(), lock)
	return b.Status
}

//其他解仓
func UnlockBalanceOther(addr string,amount uint64)bool{
	lock :=new(pb.ReqUnlockBalance)
	lock.Address = addr
	//lock.Amount = ToWei(amount, 11).Uint64()
	lock.Amount =amount
	var b1,b2,b3,b4 bool
	if ktoClient1==nil {
		createClient(1)
		rl1,_:= ktoClient1.SetUnlockBalance(context.Background(), lock)
		b1 = rl1.Status
	}else{
		rl1,_:= ktoClient1.SetUnlockBalance(context.Background(), lock)
		b1 = rl1.Status
	}
	if ktoClient2==nil {
		createClient(2)
		rl2,_:= ktoClient2.SetUnlockBalance(context.Background(), lock)
		b2 = rl2.Status
	}else{
		rl2,_:= ktoClient2.SetUnlockBalance(context.Background(), lock)
		b2 = rl2.Status
	}
	if ktoClient3==nil {
		createClient(3)
		rl3,_:= ktoClient3.SetUnlockBalance(context.Background(), lock)
		b3 = rl3.Status
	}else{
		rl3,_:= ktoClient3.SetUnlockBalance(context.Background(), lock)
		b3 = rl3.Status
	}
	if ktoClient4==nil {
		createClient(4)
		rl4,_:= ktoClient4.SetUnlockBalance(context.Background(), lock)
		b4 = rl4.Status
	}else{
		rl4,_:= ktoClient4.SetUnlockBalance(context.Background(), lock)
		b4 = rl4.Status
	}
	//logs.Info(b1,b2,b3,b4)
	sb :=b1&&b2&&b3&&b4
	return sb
}

//其他单个解仓
func UnlockOneBalanceOther(addr string,amount float64,i int )bool{
	lock :=new(pb.ReqUnlockBalance)
	lock.Address = addr
	lock.Amount = ToWei(amount, 11).Uint64()
	var b1 bool
	switch i {
	case 31:
		if ktoClient1==nil {
			createClient(1)
			rl1,_:= ktoClient1.SetUnlockBalance(context.Background(), lock)
			b1 = rl1.Status
		}else{
			rl1,_:= ktoClient1.SetUnlockBalance(context.Background(), lock)
			b1 = rl1.Status
		}
		break
	case 32:
		if ktoClient2==nil {
			createClient(2)
			rl2,_:= ktoClient2.SetUnlockBalance(context.Background(), lock)
			b1 = rl2.Status
		}else{
			rl2,_:= ktoClient2.SetUnlockBalance(context.Background(), lock)
			b1 = rl2.Status
		}
		break
	case 33:
		if ktoClient3==nil {
			createClient(3)
			rl3,_:= ktoClient3.SetUnlockBalance(context.Background(), lock)
			b1 = rl3.Status
		}else{
			rl3,_:= ktoClient3.SetUnlockBalance(context.Background(), lock)
			b1 = rl3.Status
		}
		break
	case 34:
		if ktoClient4==nil {
			createClient(4)
			rl4,_:= ktoClient4.SetUnlockBalance(context.Background(), lock)
			b1 = rl4.Status
		}else{
			rl4,_:= ktoClient4.SetUnlockBalance(context.Background(), lock)
			b1 = rl4.Status
		}
		break
	}
	return b1
}

func ToWei(iamount interface{}, decimals int) *big.Int {
	amount := decimal.NewFromFloat(0)
	switch v := iamount.(type) {
	case string:
		amount, _ = decimal.NewFromString(v)
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei
}
//转账
func SendTradeKto(fromAddr,toAddr,pri string,amount,n uint64)(string,uint64,error){
	if n == 0 {
		n = GetAddrNonce(fromAddr)
	}else{
		n = n+1
	}
	transaction := new(pb.ReqTransaction)
	transaction.From =fromAddr
	transaction.To = toAddr
	transaction.Priv = pri
	transaction.Amount = amount
	transaction.Nonce = n

	resTran, err := ktoClient.SendTransaction(context.Background(), transaction)
	if err!=nil {
		logs.Info(err)
		return "",n,err
	}

	return resTran.Hash,n,nil
}

//获取地址nnce
func GetAddrNonce(fromAddr string)uint64{
	nonce := new(pb.ReqNonce)
	nonce.Address = fromAddr
	resNonce, _ := ktoClient.GetAddressNonceAt(context.Background(), nonce)
	return resNonce.Nonce
}

func QueryTx(hash string)(bool,string){
	req := new(pb.ReqTxByHash)
	req.Hash = hash
	tx, _ := ktoClient.GetTxByHash(context.Background(), req)
	logs.Info("tx=",tx)
	if tx== nil{
		return false,"0"
	}else{
		return true,strconv.FormatUint(tx.BlockNum, 10)
	}
}