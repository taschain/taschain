import account


class ContractGame():
    def __init__(self):
        self.a = 10
        self.mystr = "myabc"
        self.mybool = True
        self.mynone = 1
        print("deploy")

    def ready(self):
        print("game is ready.")

    @register.public()
    def contract_int(self):
        print("contract is called")
        try:
            count = 111
            self.a = self.a + count
            print("count from another contract. count = " + str(self.a))
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_str(self):
        print("contract is called")
        try:
            count = account.contractCall("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "count_str", "[]")
            self.mystr += count
            print("count from another contract. count = " + self.mystr)
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_bool(self):
        print("contract is called")
        try:
            self.mybool = account.contractCall("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "count_bool", "[True]")
            print("count from another contract. count = " + str(self.mybool))
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_none(self):
        print("contract is called")
        try:
            mynone = account.contractCall("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "count_none", "[]")
            print("count from another contract. count = " + str(mynone))
            if mynone is None:
                self.mynone = 2
        except Exception as e:
            print(e.args)

    @register.public()
    def contract_deep(self):
        print("===deep start==")
        try:
            count = account.contractCall("0x2c34ce1df23b838c5abf2a7f6437cca3d3067ed509ff25f11df6b11b582b51eb", "count_deep", "[]")
            abc = count + 1
            print("count from another contract. count 1 ")
            print("count from another contract. count = " + str(abc))
        except Exception as e:
            print(e.args)
        print("===deep end==")
