

from clib.tas_runtime import glovar
from clib.tas_runtime.address import Address
from clib.tas_runtime.msgxx import Msg


# 创建账户
caller = glovar.storage.new_account()

# 创建充值合约
with open("recharge/recharge.py", "r", encoding="utf-8") as f:
    code = f.read()
recharge_contract_addr = glovar.storage.new_contract(code, "Recharge")
recharge_contract = Address(recharge_contract_addr)

# 创建token合约
with open("token/contract_token.py", "r", encoding="utf-8") as f:
    code = f.read()
token_contract_addr = glovar.storage.new_contract(code, "MyAdvancedToken")
token_contract = Address(token_contract_addr)

# 部署合约
glovar.msg = Msg(data="", sender=Address(caller), value=100)
token_contract.call("deploy")

# 调用token合约，购买
token_contract.call("buy")

# 调用token合约，进行充值
token_contract.call("approveAndCall", recharge_contract_addr, 50, "13968999999")