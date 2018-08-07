# -*- coding: utf-8 -*-
import atexit
import json


@atexit.register
def exit():
    print("exit")
    with open("data.txt", "wb") as f:
        f.write(json.dumps(Storage.data).encode("utf-8"))

class Storage(object):
    data = {}

    @staticmethod
    def get(addr, key):
        return Storage.data[addr][key]

    @staticmethod
    def put(addr, key, value):
        if Storage.data.get(addr) is None:
            Storage.data[addr] = {}
        Storage.data[addr][key] = value

    @staticmethod
    def delete(addr, key):
        del Storage.data[addr][key]

    @staticmethod
    def load(addr, obj):
        for k in Storage.data.get(addr, []):
            setattr(obj, k, Storage.data[addr][k])
        # print("Load:", Storage.data)

    @staticmethod
    def save(addr, obj):
        for k in obj.__dict__:
            Storage.put(addr, k, obj.__dict__[k])
        print("Save:", Storage.data)


contract_dict = {}
with open("data.txt", "r") as f:
    Storage.data = json.loads(f.read())


def register_contract(addr, filename, class_name):
    env = {}
    with open(filename, "r", encoding="utf-8") as f:
        code = f.read()
    exec(code, env)
    exec("obj = " + class_name + "()", env)
    contract_dict[addr] = {"class": class_name, "code": code}


class Contract(object):

    def __init__(self, addr):
        self.contract = contract_dict.get(addr)
        self.addr = addr
        self.env = {}
        exec(self.contract["code"], self.env)
        exec("tas_obj ={class_name}()".format(class_name=self.contract["class"]), self.env)
        Storage.load(self.addr, self.env.get("tas_obj"))


    def call(self, function_name, *args, **kwargs):
        self.env["tas_args"] = args
        self.env["tas_kwargs"] = kwargs
        self.env["tas_function_name"] = function_name
        exec("tas_func = getattr(tas_obj, tas_function_name, None)", self.env)
        exec("tas_func(*tas_args, **tas_kwargs)", self.env)


    def __del__(self):
        Storage.save(self.addr, self.env.get("tas_obj"))


if __name__ == '__main__':
    register_contract("0x1", "test.py", "MyAdvancedToken")
    con = Contract("0x1")
    #con.call("set_prices", 100, 100)
    con.call("test", "hehe")
    del con

