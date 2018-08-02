package tvm

import (
	"io/ioutil"
	"fmt"
)

func Read0(filename string)  (string){
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("read fail", err)
	}
	return string(f)
}

func VmTest() {

	tvm := NewTvm(nil)
	tvm.Execute(Read0("py/token/contract_token_test.py"))
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

func VmTestABI() {
	tvm := NewTvm(nil)


	tvm.Execute(`
def Test(a, b, c, d):
    print(a)
    print(b)
    print(c)
    print(d)
`)

	str := `{"FuncName": "Test", "Args": [10.123, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`
	tvm.ExecuteABIJson(str)
}

func VmTestException() {
	tvm := NewTvm(nil)

	tvm.Execute(`
i am error
`)
}

func VmTestToken(){
	tvm := NewTvm(nil)

	tvm.Execute(`
import account
class Storage(object):
    data = {}
    @staticmethod
    def get(key):
        return data[key]

    @staticmethod
    def put(key, value):
        data[key] = value

    @staticmethod
    def delete(key):
        del data[key]


def require(b):
    if not b:
        raise (Exception, "")


class Address(object):
    def __str__(self):
        return self.value

    def __set__(self, instance, value):
        self.value = value

    def __init__(self, value):
        self.value = value

    def __eq__(self, other):
        return self.value == other.value

    def invalid(self):
        # TODO 检查是否合法地址
        return True

    def balance(self):
        #获取地址里的余额
		leftSum = account.getBalance(self.value)
        return leftSum

    def transfer(self, _value):
        #转账到合约
		account.transfer(self.value, this, _value )
        pass


# 当前合约的地址
this = account.contractAddr(Msg.sender);


class Msg(object):
    sender = Address("")
    value = 0


#调用者是否为合约创建者
def owner():
    b = account.isContractCreater(Msg.sender)
    if b:
        return True
    else:
        raise (Exception, "只有合约owner可以操作")


class Event(object):
    @staticmethod
    def emit(event_name, *param):
        print(event_name, param)
        #for item in param:
            #print(item)

#
#


class TokenERC20(object):
    def __init__(self):
        self.name = ""
        self.symbol = ""
        self.totalSupply = 0

        self.balanceOf = {}
        self.allowance = {}
    '''
    // This generates a public event on the blockchain that will notify clients
    event Transfer(address indexed from, address indexed to, uint256 value);

    // This generates a public event on the blockchain that will notify clients
    event Approval(address indexed _owner, address indexed _spender, uint256 _value);

    // This notifies clients about the amount burnt
    event Burn(address indexed from, uint256 value);
    '''

    def _transfer(self, _from, _to, _value):
        if _to not in self.balanceOf:
            self.balanceOf[_to] = 0
        if _from not in self.balanceOf:
            self.balanceOf[_from] = 0
        # 接收账户地址是否合法
        require(_to.invalid())
        # 账户余额是否满足转账金额
        require(self.balanceOf[_from] >= _value)
        # 检查转账金额是否合法
        require(_value > 0)
        # 转账
        self.balanceOf[_from] -= _value
        self.balanceOf[_to] += _value

    def transfer(self, _to, _value):
        self._transfer(Msg.sender, _to, _value)

    def transfer_from(self, _from, _to, _value):
        require(_value <= self.allowance[_from][Msg.sender])
        self.allowance[_from][Msg.sender] -= _value
        self._transfer(_from, _to, _value)
        return True

    def approve(self, _spender, _value):
        self.allowance[Msg.sender][_spender] = _value
        Event.emit("Approval", Msg.sender, _spender, _value)
        return True

    def approveAndCall(self, _spender, _value, _extraData):
        #tokenRecipient spender = tokenRecipient(_spender)
        if self.approve(_spender, _value):
            #spender.receiveApproval(Msg.sender, _value, this, _extraData);
            return True
        else:
            return False

    def burn(self, _value):
        #检查账户余额
        require(self.balanceOf[Msg.sender] >= _value)
        self.balanceOf[Msg.sender] -= _value
        self.totalSupply -= _value
        Event.emit("Burn", Msg.sender, _value)
        return True

    def burnFrom(self, _from, _value):
        if _from not in self.balanceOf:
            self.balanceOf[_from] = 0
        #检查账户余额
        require(self.balanceOf[_from] >= _value)
        require(_value <= self.allowance[_from][Msg.sender])
        self.balanceOf[_from] -= _value
        self.allowance[_from][Msg.sender] -= _value
        self.totalSupply -= _value
        Event.emit("Burn", _from, _value)
        return True


class MyAdvancedToken(TokenERC20):
    def __init__(self):
        super(MyAdvancedToken, self).__init__()

        self.sell_price = 0
        #Storage.register("sell_price")
        self.buy_price = 0

        self.frozenAccount = {}

    def apply(self):
        pass

    def deploy(self):
        self.sell_price = 100
        self.buy_price = 100
        self.name = "TAS"
        self.symbol = "%"
        self.totalSupply = 1000000

    #/* This generates a public event on the blockchain that will notify clients */
    #event FrozenFunds(address target, bool frozen);

    # @property
    # def sell_price(self):
    #     self._sell_price = Storage.get("sell_price")
    #     return self._sell_price
    #
    # @sell_price.setter
    # def sell_price(self, value):
    #     self._sell_price = value
    #     Storage.put("sell_price", self._sell_price)
    #
    # @sell_price.deleter
    # def sell_price(self):
    #     Storage.delete("sell_price")
    #     self._sell_price = 0

    def _transfer(self, _from, _to, _value):
        require(_to.invalid)
        require(self.balanceOf[_from] >= _value)
        require(_value > 0)
        require(not self.frozenAccount[_from])
        require(not self.frozenAccount[_to])
        self.balanceOf[_from] -= _value
        self.balanceOf[_to] += _value
        Event.emit("Transfer", _from, _to, _value)

    def mint_token(self, target, minted_amount):
        owner()
        self.balanceOf[target] += minted_amount
        self.totalSupply += minted_amount
        Event.emit("Transfer", 0, this, minted_amount)
        Event.emit("Transfer", this, target, minted_amount)

    def freeze_account(self, target, freeze):
        owner()
        self.frozenAccount[target] = freeze
        Event.emit("FrozenFunds", target, freeze)

    def set_prices(self, new_sell_price, new_buy_price):
        owner()
        self.sell_price = new_sell_price
        self.buy_price = new_buy_price

    def buy(self):
        amount = Msg.value / self.buy_price
        self._transfer(this, Msg.sender, amount)
        #扣钱
		account.subBlance(Msg.sender, amount)

    def sell(self, amount):
        require(this.balance() >= amount * self.sell_price)
        self._transfer(Msg.sender, this, amount)
        Msg.sender.transfer(amount * self.sell_price)



#
#
#
`)
}