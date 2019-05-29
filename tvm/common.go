package tvm

import (
	"bufio"
	"errors"
	"strings"
)

func ContractInfo(code string) (map[string]string, error) {

	rtv := make(map[string]string)
	var err error
	tokenMap := map[string]string{
		"version": "#tvm_version",
		"type":    "#tvm_type",
	}

	sr := strings.NewReader(code)
	buf := bufio.NewReader(sr)
	//buf := bufio.NewReaderSize(sr, 0)
	var codeLine []byte
	for {
		line, isPrefix, err := buf.ReadLine()
		codeLine = append(codeLine, line...)
		if nil != err {
			break
		}
		if !isPrefix {
			//fmt.Printf("%q\n", codeLine)
			for k, v := range tokenMap {
				if strings.HasPrefix(string(codeLine), v) {
					//fmt.Printf("%q\n", codeLine)
					rtv[k] = strings.TrimPrefix(string(codeLine), v)
					rtv[k] = strings.TrimSpace(rtv[k])
				}
			}
			codeLine = nil
		}
	}
	if err == nil {
		if CheckContractInfo(rtv) {
			err = errors.New("合约信息格式错误")
		}
	}
	return rtv, err
}

func CheckContractInfo(contractInfo map[string]string) bool {
	//TODO:
	return true
}
