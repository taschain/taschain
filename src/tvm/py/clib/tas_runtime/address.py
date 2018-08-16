
from clib.tas_runtime.contract import Contract
from clib.tas_runtime import glovar


class Address(object):
    this = ""

    def __init__(self, address):
        self.data = glovar.storage.get(address)
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

    # def __str__(self):
    #     return self.value
    #
    # def __repr__(self):
    #     return self.value

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
        if not self.data.is_contract():
            raise Exception()
            glovar.storage.snapshot()
        if getattr(self, "contract", None) is None:
            env = {}
            Address.this = self.value
            env["this"] = Address(self.value)
            env["Address"] = Address
            env["msg"] = glovar.msg
            self.contract = Contract(self.value, env)
        try:
            self.contract.call(function_name, *args, **kwargs)
        except Exception as e:
            print(repr(e))
            print("error of calling {f}!".format(f=function_name))
            glovar.storage.revert_to_snapshot()
            raise e

