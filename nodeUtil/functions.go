package nodeUtil

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/shopspring/decimal"
	"strconv"
	"time"

	"strings"
)


func Md5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str) )
	s := hex.EncodeToString(hash.Sum(nil))
	return s
}

func Fload64Split(v float64)string{
	sm := decimal.NewFromFloat(v).Round(8).String()
	x := int64(v)
	y := float64(x)
	if v!=y {
		if !strings.Contains(sm, ".") {
			return fmt.Sprint(sm)
		}
		if len(strings.Split(sm,".")[1])<7{
			return sm
		}else{
			i2 := sm[:len(sm)-2]
			return i2
		}
	}else{
		return sm
	}
}

func TimeSection()bool{
	time2:= time.Now().Format("15:04:05")
	all := strings.ReplaceAll(time2, ":", "")
	n, _ := strconv.Atoi(all)

	end:= "00:30:00"
	et := strings.ReplaceAll(end, ":", "")
	e, _ := strconv.Atoi(et)
	if n>e {
		return false
	}
	return true
}

func Contains(l []string, value string) bool {
	for _, v := range l {
		if strings.EqualFold(v,value) {
			return true
		}
	}
	return false
}

func ContainsInt64(l []int64, value int64) bool {
	for _, v := range l {
		if v==value{
			return true
		}
	}
	return false
}
