import hashlib
import time

class Storage(object):
    def __init__(self):
        self.data = {}
        self.count = 0

    def new_contract(self, code: str, class_name: str):
        address = AddressStorage(is_contract=True)
        address.set_code(code)
        address.set_class(class_name)
        key = "0x{count}".format(count=self.count)
        self.count += 1
        self.data[key] = address
        return key

    def new_account(self, balance=10000):
        address = AddressStorage()
        key = "0x{count}".format(count=self.count)
        self.count += 1
        address.set_balance(balance)
        self.data[key] = address
        return key

    def get(self, addr):
        return self.data.get(addr, AddressStorage())


class AddressStorage():
    def __init__(self, is_contract=False):
        address_type = "normal"
        if is_contract:
            address_type = "contract"
        self.data = {
            "type": address_type,
            "code": "",
            "class": "",
            "balance": 0,
            "data": {}
        }

    def is_contract(self):
        return self.data["type"] == "contract"

    def get_balance(self):
        return self.data.get("balance")

    def set_balance(self, value: int):
        self.data["balance"] = value

    def set_code(self, code: str):
        self.data["code"] = code

    def get_code(self):
        return self.data["code"]

    def set_class(self, class_name: str):
        self.data["class"] = class_name

    def get_class(self):
        return self.data["class"]

    def dump_data(self, obj):
        if not self.is_contract():
            raise Exception("")
        for k in obj.__dict__:
            self.data["data"][k] = obj.__dict__[k]

    def load_data(self, obj):
        for k in self.data["data"]:
            setattr(obj, k, self.data["data"][k])



