import account

class ContractBeCalled():
    def __init__(self):
        pass

    def deploy(self):
        print("deploy")


    def ready(self):
        print("ContractBeCalled is ready.")

    def count_int(self, input):
        print("ContractBeCalled is called 1 ")
        return 100+input

    def count_str(self):
        print("ContractBeCalled is called 2")
        return "bcd"

    def count_bool(self, input):
        print("ContractBeCalled is called 3")
        if input:
            return not input

    def count_none(self):
        print("ContractBeCalled is called 4")
        pass

    def count_deep(self):
        print("===deeper start==")
        account.contractCall("0x9a6bf01ba09a5853f898b2e9e6569157a01a7a00", "deepest", "[]")
        print("===deeper end==")
        pass


# contractBeCalled = ContractBeCalled()
# contractBeCalled.count_str("abc")
# print("123")