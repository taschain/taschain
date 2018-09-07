

from clib.tas_runtime import glovar
from clib.tas_runtime.address_tas import Address
from lib.base.event import Event
from lib.base.utils_tas import *


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
        require(Address(_to).invalid())
        # 账户余额是否满足转账金额
        require(self.balanceOf[_from] >= _value)
        # 检查转账金额是否合法
        require(_value > 0)
        # 转账
        self.balanceOf[_from] -= _value
        self.balanceOf[_to] += _value
        Event.emit("Transfer", _from, _to, _value)

    def transfer(self, _to, _value):
        self._transfer(glovar.msg.sender, _to, _value)

    def transfer_from(self, _from, _to, _value):
        require(_value <= self.allowance[_from][glovar.msg.sender])
        self.allowance[_from][glovar.msg.sender] -= _value
        self._transfer(_from, _to, _value)
        return True

    def approve(self, _spender, _value):
        if glovar.msg.sender not in self.allowance:
            self.allowance[glovar.msg.sender] = {}
        self.allowance[glovar.msg.sender][_spender] = _value
        Event.emit("Approval", glovar.msg.sender, _spender, _value)
        return True

    def approveAndCall(self, _spender, _value, _extraData):
        spender = Address(_spender)
        if self.approve(_spender, _value):
            spender.call("receive_approval", glovar.msg.sender, _value, glovar.this, _extraData)
            return True
        else:
            return False

    def burn(self, _value):
        #检查账户余额
        require(self.balanceOf[glovar.msg.sender] >= _value)
        self.balanceOf[glovar.msg.sender] -= _value
        self.totalSupply -= _value
        Event.emit("Burn", glovar.msg.sender, _value)
        return True

    def burnFrom(self, _from, _value):
        # if _from not in self.balanceOf:
        #     self.balanceOf[_from] = 0
        #检查账户余额
        require(self.balanceOf[_from] >= _value)
        require(_value <= self.allowance[_from][glovar.msg.sender])
        self.balanceOf[_from] -= _value
        self.allowance[_from][glovar.msg.sender] -= _value
        self.totalSupply -= _value
        Event.emit("Burn", _from, _value)
        return True
