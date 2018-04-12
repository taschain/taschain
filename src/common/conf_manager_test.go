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
	//cm.SetBool("test_1", "bool_1", true)
	//cm.SetDouble("test_2", "double_1", 10.33)
	//cm.SetString("test_2", "str1", "abc好的")

	t.Log(cm.GetBool("test_1", "bool_1"))
	t.Log(cm.GetDouble("test_2", "double_1"))
	t.Log(cm.GetString("test_2", "str1"))
	t.Log(cm.GetString("test_2", "str2"))

}