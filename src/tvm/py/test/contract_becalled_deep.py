import account

class ContractBeCalledDeep():
    def __init__(self):
        pass

    def deploy(self):
        print("deploy")


    def ready(self):
        print("ContractBeCalled is ready.")

    def deepest(self):
        print("===deeptest start==")
        account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_deep", "[]")
        print("===deeptest end==")
        return 100