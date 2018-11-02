package tvm

import (
	"testing"
)

func TestVmStringParse(t *testing.T){

	// 测试用例1
	parse := vmStringParse("1|2|dd")
	if len(parse)!=3{
		t.Fatal("len is not 3")
	}
	if parse[0] != "1"{
		t.Fatal("wrong value")
	}
	if parse[1] != "2"{
		t.Fatal("wrong value")
	}
	if parse[2] != "dd"{
		t.Fatal("wrong value")
	}

	// 测试用例2
	parse = vmStringParse("1|33|3|3|")
	if len(parse)!=3{
		t.Fatal("len is not 3")
	}
	if parse[0] != "1"{
		t.Fatal("wrong value")
	}
	if parse[1] != "33"{
		t.Fatal("wrong value")
	}
	if parse[2] != "3|3|"{
		t.Fatal("wrong value")
	}

}

func TestExecutedVmSucceed(t *testing.T){

	if !ExecutedVmSucceed("1|2|dd"){
		t.Fatal("should Succeed")
	}

	if ExecutedVmSucceed("4|2|dd"){
		t.Fatal("should Succeed")
	}


}

