package routers

import (
	"github.com/astaxie/beego"
	"wallet-vote/controller"
)

func init() {
	beego.Router("/getUser", &controller.UserVoteController{}, "post:GetUser")
	beego.Router("/getAddr", &controller.UserVoteController{}, "post:GetAddr")
	beego.Router("/bindAddr", &controller.UserVoteController{}, "post:BindAddr")
	beego.Router("/bindNode", &controller.UserVoteController{}, "post:BindNode")

	beego.Router("/getNode", &controller.VoteController{}, "post:GetNode")
	beego.Router("/nodeVote", &controller.VoteController{}, "post:NodeVote")
	beego.Router("/getBill", &controller.VoteController{}, "post:GetBill")
	beego.Router("/supNodeInfo", &controller.VoteController{}, "post:SupNodeInfo")

	beego.Router("/getAbleBalance", &controller.WalletController{}, "post:GetAbleBalance")
	beego.Router("/testaa", &controller.WalletController{}, "post:Testaa")
	beego.Router("/testDongjie", &controller.WalletController{}, "post:TestDongjie")

	beego.Router("/login", &controller.AdminController{}, "post:Login")
	beego.Router("/createNode", &controller.AdminController{}, "post:CreateNode")
	beego.Router("/queryUserInfo", &controller.AdminController{}, "post:QueryUserInfo")
	beego.Router("/queryNodeList", &controller.AdminController{}, "post:QueryNodeList")
	beego.Router("/queryNodeUser", &controller.AdminController{}, "post:QueryNodeUser")
	beego.Router("/queryVoteBill", &controller.AdminController{}, "post:QueryVoteBill")
	beego.Router("/queryMachine", &controller.AdminController{}, "post:QueryMachine")
	beego.Router("/changeMachine", &controller.AdminController{}, "post:ChangeMachine")
	beego.Router("/testImage", &controller.AdminController{}, "Get:TestImage")
	beego.Router("/exportSql", &controller.AdminController{}, "Get:ExportSql")
	beego.Router("/sysBufa", &controller.AdminController{}, "post:SysBufa")
	beego.Router("/setNode", &controller.AdminController{}, "post:SetNode")
	beego.Router("/setNodeUser", &controller.AdminController{}, "post:SetNodeUser")
	beego.Router("/supNodeLose", &controller.AdminController{}, "post:SupNodeLose")
	beego.Router("/setNodeAgain", &controller.AdminController{}, "post:SetNodeAgain")
	beego.Router("/jiedo", &controller.AdminController{}, "post:Jiedo")
	beego.Router("/jiedoAllAmount", &controller.AdminController{}, "post:JiedoAllAmount")
	beego.Router("/chekUserFrooz", &controller.AdminController{}, "post:ChekUserFrooz")
	beego.Router("/chekAddrRpcFrooz", &controller.AdminController{}, "post:ChekAddrRpcFrooz")

}



