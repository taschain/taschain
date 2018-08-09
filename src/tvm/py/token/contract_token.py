
from lib import Address

class Storage(object):
    data = {}

    @staticmethod
    def get(key):
        return Storage.data[key]

    @staticmethod
    def put(key, value):
        Storage.data[key] = value

    @staticmethod
    def delete(key):
        del Storage.data[key]

    @staticmethod
    def load(obj):
        for k in Storage.data:
            setattr(obj, k, Storage.data[k])
        # print("Load:", Storage.data)

    @staticmethod
    def save(obj):
        for k in obj.__dict__:
            Storage.put(k, obj.__dict__[k])
        print("Save:", Storage.data)


def require(b):
    if not b:
        raise Exception("")


this = Address("")

owner = Address("")


class Msg(object):
    sender = Address("")
    value = 0

    # @staticmethod
    # def sender():
    #     return Msg._sender
    #
    # @staticmethod
    # def set_sender(address):
    #     Msg._sender = address
    #
    # @staticmethod
    # def value():
    #     return Msg._value
    #
    # @staticmethod
    # def set_value(value):
    #     Msg._value = value


#调用者是否为合约创建者
def check_owner():
    if owner == Msg.sender:
        return True
    else:
        raise Exception("只有合约owner可以操作")


class Event(object):
    @staticmethod
    def emit(event_name, *param):
        print("Event: ", event_name, param)


# def tokenRecipient(_sender, _value, _tokenContract, _extraData):
#     require(_tokenContract == tokenContract);
#     require(tokenContract.transferFrom(_sender, address(this), 1));
#     uint256 payloadSize;
#     uint256 payload;
#     assembly
#     {
#     payloadSize: = mload(_extraData)
#     payload: = mload(add(_extraData, 0x20))
#     }
#     payload = payload >> 8 * (32 - payloadSize);
#     info[sender] = payload;

#
#


class BalanceDict():
    def __init__(self):
        self.data = {}

    def __getitem__(self, item):
        if item not in self.data:
            self.data[item] = 0
        return self.data[item]

    def __setitem__(self, key, value):
        self.data[key] = value

    def __delitem__(self, key):
        del self.data[key]


class TokenERC20(object):
    def __init__(self):
        self.name = ""
        self.symbol = ""
        self.totalSupply = 0

        self.balanceOf = {}
        self.allowance = {}

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
        # if _from not in self.balanceOf:
        #     self.balanceOf[_from] = 0
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
        self.balanceOf[Msg.sender] = self.totalSupply

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
        if _from not in self.balanceOf:
            self.balanceOf[_from] = 0
        require(self.balanceOf[_from] >= _value)
        require(_value > 0)
        # require((_from not in self.frozenAccount) or (not self.frozenAccount[_from]))
        # require((_to not in self.frozenAccount) or (not self.frozenAccount[_to]))
        self.balanceOf[_from] -= _value
        if _to not in self.balanceOf:
            self.balanceOf[_to] = 0
        self.balanceOf[_to] += _value
        Event.emit("Transfer", _from, _to, _value)

    def mint_token(self, target, minted_amount):
        check_owner()
        if target not in self.balanceOf:
            self.balanceOf[target] = 0
        self.balanceOf[target] += minted_amount
        self.totalSupply += minted_amount
        Event.emit("Transfer", 0, this, minted_amount)
        Event.emit("Transfer", this, target, minted_amount)

    def freeze_account(self, target, freeze):
        check_owner()
        self.frozenAccount[target] = freeze
        Event.emit("FrozenFunds", target, freeze)

    def set_prices(self, new_sell_price, new_buy_price):
        check_owner()
        self.sell_price = new_sell_price
        self.buy_price = new_buy_price

    def buy(self):
        amount = Msg.value / self.buy_price
        self._transfer(this, Msg.sender, amount)
        # TODO 扣钱

    def sell(self, amount):
        require(this.balance() >= amount * self.sell_price)
        self._transfer(Msg.sender, this, amount)
        Msg.sender.transfer(amount * self.sell_price)



#
#
#










