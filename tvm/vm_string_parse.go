//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package tvm

import "C"

//const split = "|"

// vm返回的字符串进行解析：
// 按照约定：任何情况下，底层必定按照这个格式返，所以不考虑格式非法问题
// 字符串  type|code|data|abi
// type的定义：RETURN_TYPE_INT = "1" RETURN_TYPE_STRING = "2" RETURN_TYPE_NONE = "3" RETURN_TYPE_EXCEPTION = "4" RETURN_TYPE_BOOL = "5"

//func VmStringParse(original string) *[4]string {
//	return vmStringParse(original)
//}
//
//func vmStringParse(original string) *[4]string {
//	result := new([4]string)
//	// 获取type
//	indexOne := strings.Index(original, split)
//
//	if indexOne < 0 {
//		result[0] = original
//		return result
//	}
//	result[0] = original[:indexOne]
//
//	//获取code
//	original = original[indexOne+1:]
//	indexTwo := strings.Index(original, split)
//	if indexTwo < 0 {
//		result[1] = original
//		return result
//	}
//	result[1] = original[:indexTwo]
//
//	//获取data
//	original = original[indexTwo+1:]
//	indexThree := strings.Index(original, split)
//	if indexThree < 0 {
//		result[2] = original
//		return result
//	}
//	result[2] = original[:indexThree]
//
//	//获取abi
//	result[3] = original[indexThree+1:]
//	return result
//}

//func ExecutedVmSucceed(original string, result *ExecuteResult) (int,string) {
//	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
//		fmt.Printf("execute error,code=%d,msg=%s \n", result.ErrorCode, result.Content)
//		return result.ErrorCode, result.Content
//	} else {
//		return types.SUCCESS, ""
//	}
//}
