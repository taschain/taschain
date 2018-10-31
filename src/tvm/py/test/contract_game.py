import account


class ContractGame():
    def __init__(self):
        self.a = 10

    def deploy(self):
        print("deploy")

    def ready(self):
        print("game is ready.")

    def contract(self):
        print("contract is called")
        try:
            count = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count", "[]")
            abc = count +1
            print ("count from another contract. count 1 ")
            print ("count from another contract. count = " + str(abc))
        except Exception as e:
            print(e.args)
