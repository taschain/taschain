

from lib.base.msg import Msg

from lib.base.address import Address

import base


with open("test.py", "r", encoding="utf-8") as f:
    code = f.read()
caller = base.storage.new_account()
base.msg = Msg(data="", sender=Address("0x0"), value=100)
contract_addr = base.storage.new_contract(code, "MyAdvancedToken")
print(contract_addr)
contract = Address(contract_addr)
contract.call("deploy")
contract.call("buy")