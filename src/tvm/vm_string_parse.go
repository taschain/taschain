package tvm

import (
	"strings"
	"strconv"
	"middleware/types"
)

const split = "|"

// vm返回的字符串进行解析：
// 按照约定：任何情况下，底层必定按照这个格式返，所以不考虑格式非法问题
// 字符串  type|code|data
// type的定义：RETURN_TYPE_INT = "1" RETURN_TYPE_STRING = "2" RETURN_TYPE_NONE = "3" RETURN_TYPE_EXCEPTION = "4" RETURN_TYPE_BOOL = "5"

func vmStringParse(original string) *[3]string {
	result := new([3]string)
	// 获取type
	indexOne := strings.Index(original, split)
	if indexOne < 0 {
		return result
	}
	result[0] = original[:indexOne]

	//获取code
	original = original[indexOne+1:]
	indexTwo := strings.Index(original, split)
	if indexTwo < 0 {
		return result
	}
	result[1] = original[:indexTwo]

	//获取data
	result[2] = original[indexTwo+1:]

	return result
}

func ExecutedVmSucceed(original string) (int,string) {
	parsed := vmStringParse(original)
	if parsed[0] == "4" {
		errorCode,err:=strconv.Atoi(parsed[1])
		if err != nil{
			return types.Sys_Error,err.Error()
		}else{
			return errorCode,parsed[2]
		}
	} else {
		return 0,""
	}

}
