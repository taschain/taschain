import account

class ContractException():
    def __init__(self):
        pass

    def expectValue(self,expectValue,trueValue):
        if expectValue != trueValue:
            raise Exception("error value,expectValue="+str(expectValue)+".getvalue is:" + str(trueValue))

    @register.public()
    def callExcption1(self):
        try:
            xx = account.contractCall("0x2a4e0a5fb3d78a2c725a233b1bccff7560c35610", "be_calledException1", "[]")
        except Exception as e:
            print("====error is " + str(e))
            if str(e).count("ddd")>0:
                self.callExcption1 = "success"

    @register.public()
    def be_calledException1(self):
        self.a=ddd


