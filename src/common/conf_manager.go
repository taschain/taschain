package common

import (
	"github.com/glacjay/goini"
	"sync"
	"os"
)

/*
**  Creator: pxf
**  Date: 2018/4/10 下午2:13
**  Description: 
*/

type ConfManager interface {
	//read basic conf from tas.conf file
	GetString(section string, key string) (string, bool)
	GetBool(section string, key string) (bool, bool)
	GetDouble(section string, key string) (float64, bool)
	GetInt(section string, key string) (int, bool)

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
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	cs.dict = ini.MustLoad(path)

	return cs
}

func (cs *ConfFileManager) GetString(section string, key string) (string, bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.dict.GetString(section, key)
}

func (cs *ConfFileManager) GetBool(section string, key string) (bool, bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.dict.GetBool(section, key)
}

func (cs *ConfFileManager) GetDouble(section string, key string) (float64, bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.dict.GetDouble(section, key)
}

func (cs *ConfFileManager) GetInt(section string, key string) (int, bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.dict.GetInt(section, key)
}

func (cs *ConfFileManager) SetString(section string, key string, value string) {
	cs.update(func() {
		cs.dict.SetString(section, key, value)
	})
}

func (cs *ConfFileManager) SetBool(section string, key string, value bool) {
	cs.update(func() {
		cs.dict.SetBool(section, key, value)
	})
}

func (cs *ConfFileManager) SetDouble(section string, key string, value float64) {
	cs.update(func() {
		cs.dict.SetDouble(section, key, value)
	})
}

func (cs *ConfFileManager) SetInt(section string, key string, value int) {
	cs.update(func() {
		cs.dict.SetInt(section, key, value)
	})
}


func (cs *ConfFileManager) Del(section string, key string) {
	cs.update(func() {
		cs.dict.Delete(section, key)
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





