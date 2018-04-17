package common

import (
	"github.com/glacjay/goini"
	"sync"
	"os"
	"strings"
)

/*
**  Creator: pxf
**  Date: 2018/4/10 下午2:13
**  Description: 
*/

type ConfManager interface {
	//read basic conf from tas.conf file
	//返回section组下的key的值, 若未配置, 则返回默认值defv
	GetString(section string, key string, defaultValue string) (string)
	GetBool(section string, key string, defaultValue bool) (bool)
	GetDouble(section string, key string, defaultValue float64) (float64)
	GetInt(section string, key string, defaultValue int) (int)

	//set basic conf to tas.conf file
	SetString(section string, key string, value string)
	SetBool(section string, key string, value bool)
	SetDouble(section string, key string, value float64)
	SetInt(section string, key string, value int)

	//delete basic conf
	Del(section string, key string)
}

type ConfFileManager struct {
	path string
	dict ini.Dict
	lock sync.RWMutex
}


func NewConfINIManager(path string) ConfManager {
	cs := &ConfFileManager{
		path: path,
	}

	_, err := os.Stat(path)

	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(path)
		if err != nil {
			//TODO: 记日志
			panic(err)
		}
	} else if err != nil {
		//TODO: 记日志
		panic(err)
	}
	cs.dict = ini.MustLoad(path)

	return cs
}

func (cs *ConfFileManager) GetString(section string, key string, defaultValue string) (string) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetString(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetBool(section string, key string, defaultValue bool) (bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetBool(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetDouble(section string, key string, defaultValue float64) (float64) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetDouble(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetInt(section string, key string, defaultValue int) (int) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetInt(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) SetString(section string, key string, value string) {
	cs.update(func() {
		cs.dict.SetString(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetBool(section string, key string, value bool) {
	cs.update(func() {
		cs.dict.SetBool(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetDouble(section string, key string, value float64) {
	cs.update(func() {
		cs.dict.SetDouble(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetInt(section string, key string, value int) {
	cs.update(func() {
		cs.dict.SetInt(strings.ToLower(section), strings.ToLower(key), value)
	})
}


func (cs *ConfFileManager) Del(section string, key string) {
	cs.update(func() {
		cs.dict.Delete(strings.ToLower(section), strings.ToLower(key))
	})
}

func (cs *ConfFileManager) update(updator func()) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

    updator()
    cs.store()
}

func (cs *ConfFileManager) store() {
    err := ini.Write(cs.path, &cs.dict)
	if err != nil {

	}
}





