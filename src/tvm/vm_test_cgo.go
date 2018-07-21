package tvm

func VmTest() {
	tvm_init()

	tvm_execute("import tas\ntas.test()")
}

func VmTestContract() {
	tvm_init()

	script := `
class TasAccount():

    address = ""

    def transfer(self, toAddress, amount):
        print("From {0} Send to {1} amout: {2}".format(self.address, toAddress, amount))
`

	tvm_execute(script)

	script = `

def apply():
    myAccount = TasAccount()
    myAccount.address = "myAddress"
    otherAccount = "otherAddress"
    myAccount.transfer(otherAccount, 50)

apply()
`
	tvm_execute(script)
}

func VmTestClass() {
	tvm_init()

	script := `

from tas import *

test()

tasa = tasaccount()

print(tasa)

#print(tasa.hello())

print("start")

print(tasa.desc)

tasa.desc = "asdfsadf"

print(tasa.desc)

print("end")

`
	tvm_execute(script)
}
