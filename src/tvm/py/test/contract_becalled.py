import account

class ContractBeCalled():
    def __init__(self):
        pass

    def deploy(self):
        print("deploy")


    def ready(self):
        print("ContractBeCalled is ready.")

    @register.public(int)
    def count_int(self, input):
        print("ContractBeCalled is called 1 ")
        return 100+input

    @register.public(str)
    def count_str(self):
        print("ContractBeCalled is called 2")
        return "bcd"

    @register.public(bool)
    def count_bool(self, input):
        print("ContractBeCalled is called 3")
        if input:
            return not input

    @register.public()
    def count_none(self):
        print("ContractBeCalled is called 4")
        pass

    @register.public()
    def count_deep(self):
        print("===deeper start==")
        account.contractCall("0x9a6bf01ba09a5853f898b2e9e6569157a01a7a00", "deepest", "[]")
        print("===deeper end==")
        pass


# contractBeCalled = ContractBeCalled()
# contractBeCalled.count_str("abc")
# print("123")