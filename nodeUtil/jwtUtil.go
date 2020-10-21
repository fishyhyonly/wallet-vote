package nodeUtil

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"strconv"
	"time"
)

type UserInfo struct {
	Id int64
	Token string
	Version string
}

func CreateToken(u UserInfo)(tokenss string,err error){
	//自定义claim
	claim := jwt.MapClaims{
		"id":       u.Id,
		"token":     u.Token,
		"version":   u.Version,
		"exp":      time.Now().Add(time.Hour*240).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,claim)
	bytes := []byte(strconv.FormatInt(u.Id, 10))
	tokenss,err  = token.SignedString(bytes)
	return
}

func secret(s string)jwt.Keyfunc{
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(s),nil
	}
}


func ParseToken(s,tokenss string)(user UserInfo,err error){
	user = UserInfo{}
	fmt.Println("s=",s)
	token,err := jwt.Parse(tokenss,secret(s))
	if err != nil{
		return
	}
	claim,ok := token.Claims.(jwt.MapClaims)
	if !ok{
		err = errors.New("cannot convert claim to mapclaim")
		return
	}
	//验证token，如果token被修改过则为false
	if  !token.Valid{
		err = errors.New("token is invalid")
		return
	}
	user.Id =int64(claim["id"].(float64))
	user.Token = claim["token"].(string)
	user.Version = claim["version"].(string)
	return
}
