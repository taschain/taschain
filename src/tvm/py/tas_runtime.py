
import pickle
import signal
import threading
import sys
import traceback


from clib.tas_runtime import glovar
from clib.tas_runtime.address import Address
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.block import Block
from clib.tas_runtime.storage import Storage


def print_dict(_dict):
    if type(_dict) is dict:
        for key in _dict:
            print("key: ", key, " value:", _dict[key])
            print_dict(_dict[key])
    elif hasattr(_dict, '__dict__'):
        for key in _dict.__dict__:
            print("key: ", key, " value:", _dict.__dict__[key])
            print_dict(_dict.__dict__[key])

# 初始化运行环境
try:
    with open("data.pkl", "rb") as f:
        data = f.read()
        if data == b"":
            glovar.storage = Storage()
        else:
            glovar.storage = pickle.loads(data)
except FileNotFoundError:
    glovar.storage = Storage()
# block = Block()
# #t = threading.Thread(target=block.run)
# block.alive = True
# t.start()
#
#
# def _exit(signum, frame):
#     print("_exit")
#     print(glovar.storage.data["0x0"].data)
#     print(glovar.storage.data["0x1"].data)
#     print(glovar.storage.data["0x2"].data)
#     block.alive = True
#     with open("data.pkl", "wb") as f:
#         pickle.dump(glovar.storage, f)
#
#
# signal.signal(signal.SIGINT, _exit)
# signal.signal(signal.SIGTERM, _exit)


# 创建账户
caller = glovar.storage.new_account()
print("caller: ", caller)


# 创建充值合约
with open("recharge/recharge.py", "r", encoding="utf-8") as f:
    code = f.read()
recharge_contract_addr = glovar.storage.new_contract(code, "Recharge")
print("recharge_contract_addr: ", recharge_contract_addr)
recharge_contract = Address(recharge_contract_addr)


# 创建TestStorage合约
with open("test/test_storage.py", "r", encoding="utf-8") as f:
    code = f.read()
test_storage_contract_addr = glovar.storage.new_contract(code, "Test")
print("test_storage_contract_addr: ", test_storage_contract_addr)
test_storage_contract = Address(test_storage_contract_addr)


# 创建token合约
with open("token/contract_token.py", "r", encoding="utf-8") as f:
    code = f.read()
token_contract_addr = glovar.storage.new_contract(code, "MyAdvancedToken")
token_contract = Address(token_contract_addr)

# 部署合约
glovar.msg = Msg(data="", sender=Address(caller), value=100)

# 调用token合约，购买
token_contract.call("buy")

# 调用token合约，进行充值
# token_contract.call("approveAndCall", recharge_contract_addr, 50, "13968999999")

# 调用TestStorage合约
token_contract.call("approveAndCall", test_storage_contract_addr, 0, "")


print("_exit")
# print_dict(glovar.storage)
# block.alive = True
with open("data.pkl", "wb") as f:
    pickle.dump(glovar.storage, f)



