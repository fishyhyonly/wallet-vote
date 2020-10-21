package dbUtil

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/labstack/gommon/log"
)
var Engine *xorm.Engine
func init(){
	u := beego.AppConfig.String("mysqluser")
	p := beego.AppConfig.String("mysqlpass")
	url := beego.AppConfig.String("mysqlurls")
	dbname := beego.AppConfig.String("mysqldb")
	port := beego.AppConfig.String("mysqlport")
	/*orm.RegisterDataBase("vote", "mysql", u+":"+p+"@tcp("+url+":+"+port+")/"+dbname+"?charset=utf8")
	orm.RegisterModel(new(model.VVoteInfo))*/
	var err error
	Engine, err = xorm.NewEngine("mysql", u+":"+p+"@tcp("+url+":+"+port+")/"+dbname+"?charset=utf8")
	if err!=nil{
		log.Fatal(err)
	}

	//在控制台打印出生成的SQL语句
	Engine.ShowSQL(true)

	//Engine.Sync2(new(model.VUser),new(model.VVoteInfo))
}

func AddNonce(n uint64,name string){
	logs.Info("name=",n)
	sql1 := "update s_wallet set "+name+"=? "
	_, err := Engine.Exec(sql1, n)
	if err != nil {
		logs.Info("sql2执行错误=",err)
	}
}

