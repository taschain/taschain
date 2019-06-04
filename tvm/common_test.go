package tvm

import "testing"

func TestContractInfo(t *testing.T) {
	code := `
#tvm_version 0.0.2
#tvm_type lib

from lib.base.utils_tas import *
from lib.base.event import Event
from lib.erc20.token_erc20_tas import TokenERC20

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

class MyAdvancedToken(TokenERC20):

    def __init__(self):
        super(MyAdvancedToken, self).__init__()

        self.sell_price = 100
        #Storage.register("sell_price")
        self.buy_price = 100

        self.frozenAccount = {}
        self.owner = ""
        self.sell_price = 100
        self.buy_price = 100
        self.name = "TAS"
        self.symbol = "%"
        self.totalSupply = 1000000
        self.balanceOf[msg.sender] = self.totalSupply
        self.owner = msg.sender

    def apply(self):
        pass



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

    # def _transfer(self, _from, _to, _value):
    #     require(_to.invalid)
    #     if _from not in self.balanceOf:
    #         self.balanceOf[_from] = 0
    #     require(self.balanceOf[_from] >= _value)
    #     require(_value > 0)
    #     # require((_from not in self.frozenAccount) or (not self.frozenAccount[_from]))
    #     # require((_to not in self.frozenAccount) or (not self.frozenAccount[_to]))
    #     self.balanceOf[_from] -= _value
    #     if _to not in self.balanceOf:
    #         self.balanceOf[_to] = 0
    #     self.balanceOf[_to] += _value
    #     Event.emit("Transfer", _from, _to, _value)

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
        # 在call前已经完成扣款
        amount = msg.value / self.buy_price
        self._transfer(this, msg.sender, amount)

    def sell(self, amount):
        require(this.balance() >= amount * self.sell_price)
        self._transfer(msg.sender, this, amount)
        msg.sender.transfer(amount * self.sell_price)

    # def test(self):
    #     print("test")



# if __name__ == '__main__':
#     a = compile("a = 1",mode="single", filename="s.py")
#     print(a.co_code.decode("utf-8"))`
}
