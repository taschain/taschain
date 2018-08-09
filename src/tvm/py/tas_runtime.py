from lib import Msg
from lib.address import Address
import lib


with open("test.py", "r", encoding="utf-8") as f:
    code = f.read()
caller = lib.storage.new_account()
lib.msg = Msg(data="", sender=Address("0x0"), value=100)
contract_addr = lib.storage.new_contract(code, "MyAdvancedToken")
print(contract_addr)
contract = Address(contract_addr)
contract.call("deploy")
contract.call("buy")