# import account

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


    # def deep(self):
    #     count = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count", "[10]")
# contractBeCalled = ContractBeCalled()
# contractBeCalled.count_str("abc")
# print("123")