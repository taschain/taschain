class Address(object):
    def __init__(self, address):
        self.value = address

    def invalid(self):
        # TODO 检查是否合法地址
        return True

    def balance(self):
        # TODO 获取地址里的余额
        return 0

    def transfer(self, _value):
        # TODO 转账到合约
        pass

    def __str__(self):
        return self.value

    def __repr__(self):
        return self.value

    def __hash__(self):
        return hash(self.value)

    def __eq__(self, other):
        return self.value == other.value

    def contract_call(self):
        pass