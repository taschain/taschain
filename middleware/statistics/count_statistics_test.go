package statistics

import "testing"

func TestCount(t *testing.T) {
	AddCount("a", 1,1)
	AddCount("a", 1,1)
	printAndRefresh()
	AddCount("a", 2,1)
	printAndRefresh()
}
