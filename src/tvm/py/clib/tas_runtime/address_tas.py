
# from clib.tas_runtime.contract import Contract
from clib.tas_runtime import glovar
import account
import ujson

class Address(object):
    this = ""

    def __init__(self, address):
        # self.data = glovar.storage.get(address)
        self.value = address

    def invalid(self):
        # TODO 检查是否合法地址
        return True

    def balance(self):
        return self.data.get_balance()

    def transfer(self, _value):
        this_data = glovar.storage.get(Address.this)
        if this_data.get_balance() < _value:
            raise Exception("")
        this_data.set_balance(this_data.get_balance() - _value)
        self.data.set_balance(self.data.get_balance() + _value)

    def __str__(self):
        return self.value

    def __repr__(self):
        return self.value

    def __hash__(self):
        return hash(self.value)

    def __eq__(self, other):
        return self.value == other.value

    def __getstate__(self):
        return {"value":self.value, "data":self.data}

    def __setstate__(self, state):
        self.value = state["value"]
        self.data = state["data"]

    def call(self, function_name, *args, **kwargs):
        print("call: ", function_name)
        print("call: ", ujson.dumps(args))
        account.contractCall(self.value, function_name, ujson.dumps(args))



