from clib.tas_runtime.address import Address


class Storage(object):
    def __init__(self):
        self._data = {}
        self.count = 0
        self.copy = {}

    def new_lib_contract(self, code: str, owner: str):
        address = AddressStorage(is_contract=True)
        address.address_type = "lib"
        address.set_owner(owner)
        address.set_code(code)
        key = "0x{count}".format(count=self.count)
        self.count += 1
        self._data[key] = address
        return key

    def new_contract(self, code: str, class_name: str, owner: str, depends=[]):
        address = AddressStorage(is_contract=True)
        address.set_owner(owner)
        address.set_code(code)
        address.set_class(class_name)
        address.set_depends(depends)
        key = "0x{count}".format(count=self.count)
        self.count += 1
        self._data[key] = address
        # deploy
        contract = Address(key)
        contract.call("deploy")
        return key

    def new_account(self, balance=10000):
        address = AddressStorage()
        key = "0x{count}".format(count=self.count)
        self.count += 1
        address.set_balance(balance)
        self._data[key] = address
        return key

    def get(self, addr):
        return self._data.get(addr, AddressStorage())

    def snapshot(self):
        pass
        #self.copy = copy.deepcopy(self._data)

    def revert_to_snapshot(self):
        pass
        #self._data = copy.deepcopy(self.copy)

    @property
    def data(self):
        return self._data

    @data.setter
    def data(self, data):
        self._data = data

    # def __setattr__(self, key, value):
    #     print("Storage setattr: ", key, "value: ", value)
    #     super().__setattr__(key, value)



def check_base_type(obj):
    return obj
    base_types = [int, float, str, bool, Address]
    container_types = [dict, list, tuple]
    if type(obj) in base_types:
        return obj
    elif type(obj) in container_types:
        for item in obj:
            if type(obj) is dict:
                value = obj[item]
                if type(item) not in base_types:
                    return None
            else:
                value = item
            if check_base_type(value) is None:
                return None
        return obj
    else:
        return None


class AddressStorage(object):
    def __init__(self, is_contract=False):
        address_type = "normal"
        if is_contract:
            address_type = "contract"
        self.data = {
            "type": address_type,
            "_code": "",
            "_class": "",
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
        self.data["_code"] = code

    def get_code(self):
        return self.data["_code"]

    def set_class(self, class_name: str):
        self.data["_class"] = class_name

    def get_class(self):
        return self.data["_class"]

    def set_owner(self, owner: str):
        self.data["owner"] = owner

    def get_owner(self):
        return self.data["owner"]

    def set_depends(self, depends):
        self.data["_depends"] = depends

    def get_depends(self):
        return self.data.get("_depends", [])

    def dump_data(self, obj):
        if not self.is_contract():
            raise Exception("")
        for k in obj.__dict__:
            # print("dump_data: start")
            # print(k)
            # print(obj.__dict__[k])
            if check_base_type(obj.__dict__[k]) is not None:
                self.data["data"][k] = obj.__dict__[k]
            else:
                raise Exception("不能存储非基础类型: key: ", k, " value: ", obj.__dict__[k])
            # print("dump_data: end")

        print("dump_data:")
        print_data = self.data.copy()
        print_data.pop("_code")
        print(print_data)

    def load_data(self, obj):
        for k in self.data["data"]:
            setattr(obj, k, self.data["data"][k])

        print("load_data")
        print_data = self.data.copy()
        print_data.pop("_code")
        print(print_data)

