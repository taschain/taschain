
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
from clib.tas_runtime.importer import install_meta


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


glovar.importer = install_meta()


def setup1():
    # 创建账户 0x0
    caller = glovar.storage.new_account()
    print("caller: ", caller)

    # 创建充值合约 0x1
    with open("recharge/recharge.py", "r", encoding="utf-8") as f:
        code = f.read()
    recharge_contract_addr = glovar.storage.new_contract(code, "Recharge", caller)
    print("recharge_contract_addr: ", recharge_contract_addr)

    # 创建TestStorage合约 0x2
    with open("test/test_storage.py", "r", encoding="utf-8") as f:
        code = f.read()
    test_storage_contract_addr = glovar.storage.new_contract(code, "Test", caller)
    print("test_storage_contract_addr: ", test_storage_contract_addr)

    # 创建token合约 0x3
    with open("token/contract_token.py", "r", encoding="utf-8") as f:
        code = f.read()
    token_contract_addr = glovar.storage.new_contract(code, "MyAdvancedToken", caller)

    # 创建libcontract合约 0x5
    with open("test/test_libcontract.py", "r", encoding="utf-8") as f:
        code = f.read()
    depends_code = []
    with open("test/test_lib_helloworld.py", "r", encoding="utf-8") as f:
        depends_code.append(f.read())
    depends_addr = []
    for item in depends_code:
        depends_addr.append(glovar.storage.new_lib_contract(item, "lib_helloworld", caller))
    libcontract_contract_addr = glovar.storage.new_contract(code, "Libcontract", caller, depends_addr)



def setup2():
    caller = "0x0"
    recharge_contract_addr = "0x1"
    # recharge_contract = Address(recharge_contract_addr)
    test_storage_contract_addr = "0x2"
    # test_storage_contract = Address(test_storage_contract_addr)
    token_contract = Address("0x3")
    libcontract_contract = Address("0x5")
    glovar.msg = Msg(data="", sender=Address(caller), value=100)

    # 调用token合约，购买
    token_contract.call("buy")

    # 调用token合约，进行充值
    token_contract.call("approveAndCall", recharge_contract_addr, 50, "13968999999")

    # 调用TestStorage合约
    token_contract.call("approveAndCall", test_storage_contract_addr, 0, "")

    # 调用libcontract合约
    libcontract_contract.call("test")

setup1()
setup2()


print("_exit")
# print_dict(glovar.storage)
# block.alive = True
with open("data.pkl", "wb") as f:
    pickle.dump(glovar.storage, f)



