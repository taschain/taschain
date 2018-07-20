package tvm

func VmTest() {
	tvm_init()

	tvm_execute("import tas\ntas.test()")
}

func VmTestContract() {
	tvm_init()

	script := `
from TasAccount import *

def apply():
    myAccount = TasAccount()
    myAccount.address = "myAddress"
    otherAccount = "otherAddress"
    myAccount.transfer(otherAccount, 50)

apply()
`
	tvm_execute(script)
}
