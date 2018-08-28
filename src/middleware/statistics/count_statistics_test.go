package statistics

import "testing"

func TestCount(t *testing.T)  {
	AddCount("a",1)
	AddCount("a",1)
	printAndRefresh()
	AddCount("a",2)
	printAndRefresh()
}
