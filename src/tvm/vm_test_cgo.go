package tvm

func VmTest() {

	tvm := NewTvm(nil)

	tvm.Execute("import tas\ntas.test()")
}

func VmTestContract() {
	tvm := NewTvm(nil)

	script := `
import tas
class TasAccount():

    address = ""

    def transfer(self, toAddress, amount):
       tas.transfer(self.address, toAddress, amount)
`

	tvm.Execute(script)

	script = `

def apply():
    myAccount = TasAccount()
    myAccount.address = "myAddress"
    otherAccount = "otherAddress"
    myAccount.transfer(otherAccount, 50)

apply()
`
	tvm.Execute(script)
}

func VmTestClass() {
	tvm := NewTvm(nil)

	script := `

from tas import *

test()

tasa = tasaccount()

print(tasa)

#print(tasa.hello())

print("start")

print(tasa.desc)

tasa.desc = 123

print(tasa.desc)

print("end")

`
	tvm.Execute(script)
}
