package common

import "testing"

func runParseCoin(s string, expect uint64, t *testing.T) {
	v, err := ParseCoin(s)
	if err != nil {
		t.Fatal(err)
	}
	if v != expect {
		t.Errorf("parse coin error,input %v, expect %v", s, expect)
	}
}

func TestParseCoin_Correct(t *testing.T) {
	runParseCoin("232RA", 232, t)
	runParseCoin("232ra", 232, t)
	runParseCoin("232kra", 232000, t)
	runParseCoin("232mra", 232000000, t)
	runParseCoin("232tas", 232000000000, t)
}

func runParseCoinWrong(s string, t *testing.T) {
	_, err := ParseCoin(s)
	if err == nil {
		t.Fatalf("parse error string error: %v", s)
	}
}
func TestParseCoin_Wrong(t *testing.T) {
	runParseCoinWrong("232R", t)
	runParseCoinWrong("232a", t)
	runParseCoinWrong("", t)
	runParseCoinWrong("232", t)
	runParseCoinWrong("232 TAS", t)
}
