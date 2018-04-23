package cli

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"governance/global"
	"reflect"
	"strconv"
	"strings"
)

// VoteConfigList 解析vote命令中的传入的config参数。
type VoteConfigList [][2]string

// Set value from command params
func (vcl *VoteConfigList) Set(value string) error {
	kv := strings.Split(value, "=")
	if len(kv) != 2 {
		return fmt.Errorf("'%s' is not like 'key=value', please confirm your input", value)
	}
	*vcl = append(*vcl, [2]string{kv[0], kv[1]})
	return nil
}

// String
func (vcl *VoteConfigList) String() string {
	return ""
}

// IsCumulative 表示选项可叠加
func (vcl *VoteConfigList) IsCumulative() bool {
	return true
}

// ToVoteConfig vote命令行参数转换为core.VoteConfig
func (vcl *VoteConfigList) ToVoteConfig() (*global.VoteConfig, error) {
	voteConfig := &global.VoteConfig{}
	refVc := reflect.ValueOf(voteConfig).Elem()
	for _, kv := range *vcl {
		fd := refVc.FieldByName(kv[0])
		if !fd.CanSet() {
			// 传入未定义参数，提示用户
			// TODO 提示
		} else {
			switch fd.Kind() {
			case reflect.Interface:
				fd.Set(reflect.ValueOf(kv[1]))
			case reflect.Int:
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return nil, err
				}
				fd.Set(reflect.ValueOf(v))
			case reflect.Uint64:
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return nil, err
				}
				if v < 0 {
					return nil, fmt.Errorf("'%d' is not a uint64 param", v)
				}
				fd.Set(reflect.ValueOf(uint64(v)))
			case reflect.String:
				fd.Set(reflect.ValueOf(kv[1]))
			case reflect.Bool:
				v, err := strconv.ParseBool(kv[1])
				if err != nil {
					return nil, err
				}
				fd.Set(reflect.ValueOf(v))
			}
		}

	}
	return voteConfig, nil
}

// VoteConfigParams 解析返回[][2]string, 0:key, 1:value
func VoteConfigParams(s kingpin.Settings) (target *VoteConfigList) {
	target = (*VoteConfigList)(new([][2]string))
	s.SetValue((*VoteConfigList)(target))
	return target
}
