package common

import "testing"

/*
**  Creator: pxf
**  Date: 2018/4/11 上午9:13
**  Description: 
*/

var (
	PATH = "tas_test.ini"
	cm = NewConfINIManager(PATH)
)

func TestConfFileManager_SetBool(t *testing.T) {

	cm.SetBool("teSt_1", "bool_1", true)
	cm.SetDouble("test_2", "double_1", 10.33)
	cm.SetString("TTT", "STR", "abc好的")

	t.Log(cm.GetBool("test_1", "netId", true))
	t.Log(cm.GetDouble("test_2", "double_1", 100))
	t.Log(cm.GetString("test_2", "str1", "sss"))
	t.Log(cm.GetString("test_2", "str2", "223"))
	t.Log(cm.GetString("ttt", "ID", "dDDD"))

	sm := cm.GetSectionManager("test_2")
	t.Log(sm.GetString("str1", "sss"))
	sm.SetDouble("d1", 2932)
	sm.SetString("abc", "DBU")
	sm.Del("str2")
}