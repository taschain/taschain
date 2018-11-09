import account


class ContractGame():
    def __init__(self):
        self.a = 10
        self.mystr = "myabc"
        self.mybool = True
        self.mynone = None

    def deploy(self):
        print("deploy")

    def ready(self):
        print("game is ready.")

    @register.public()
    def contract_int(self):
        print("contract is called")
        try:
            count = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_int", "[100]")
            self.a = self.a + count
            print("count from another contract. count = " + str(self.a))
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_str(self):
        print("contract is called")
        try:
            count = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_str", "[]")
            self.mystr += count
            print("count from another contract. count = " + self.mystr)
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_bool(self):
        print("contract is called")
        try:
            self.mybool = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_bool", "[True]")
            print("count from another contract. count = " + str(self.mybool))
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_none(self):
        print("contract is called")
        try:
            self.mynone = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_none", "[]")
            print("count from another contract. count = " + str(self.mynone))
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_deep(self):
        print("===deep start==")
        try:
            count = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "count_deep", "[]")
            abc = count + 1
            print("count from another contract. count 1 ")
            print("count from another contract. count = " + str(abc))
        except Exception as e:
            print(e.args)
        print("===deep end==")
