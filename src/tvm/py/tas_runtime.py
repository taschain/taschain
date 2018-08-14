

from clib.tas_runtime import glovar
from clib.tas_runtime.address import Address
from clib.tas_runtime.msgxx import Msg

# 充值合约
with open("recharge/recharge.py", "r", encoding="utf-8") as f:
    code = f.read()
recharge_contract_addr = glovar.storage.new_contract(code, "Recharge")
print(recharge_contract_addr)
recharge_contract = Address(recharge_contract_addr)

# token合约
with open("test.py", "r", encoding="utf-8") as f:
    code = f.read()
caller = glovar.storage.new_account()
glovar.msg = Msg(data="", sender=Address("0x0"), value=100)
print(glovar.msg)
token_contract_addr = glovar.storage.new_contract(code, "MyAdvancedToken")
print(token_contract_addr)
token_contract = Address(token_contract_addr)
token_contract.call("deploy")
token_contract.call("buy")

# 调用token合约进行充值
token_contract.call("approveAndCall", recharge_contract_addr, 50, "13968999999")