import account
class ContractStorage():
    def __init__(self):
        print("==================init----------------")

    def expectValue(self,expectValue,trueValue):
        if expectValue != trueValue:
            raise Exception("error value")

    def deploy(self):
        pass

    def setBaseNeedSuccess1(self):
        self.a=1
        self.b="iamstr"
        self.c=True
        self.d=[1,2,3]
        self.e={"1":"2","3":"4"}
        self.expectValue(1,self.a)
        self.expectValue("iamstr", self.b)
        self.expectValue(True, self.c)
        self.expectValue([1,2,3], self.d)
        self.expectValue({"1":"2","3":"4"}, self.e)

    def getBaseNeedSuccess1(self):
        self.expectValue(1, self.a)
        self.expectValue("iamstr", self.b)
        self.expectValue(True, self.c)
        self.expectValue([1, 2, 3], self.d)
        self.expectValue({"1": "2", "3": "4"}, self.e)

    def setBaseNeedSuccess2(self):
        self.a=111111111111111111111111111111111111111111111
        self.expectValue(111111111111111111111111111111111111111111111, self.a)
        self.a=[1,2,3,4,5,6,1,"2","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]]
        self.expectValue([1,2,3,4,5,6,1,"2","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]], self.a)

        longStr = ""
        longDict={}
        longList=[]
        for num in range(1, 10000):
            longStr = longStr+str(num)
            longDict[str(num)]=num
            longList.append(num)
        self.x=longStr
        self.y = longDict
        self.z = longList
        self.expectValue(longStr,self.x)
        self.expectValue(longDict, self.y)
        self.expectValue(longList, self.z)


    def getBaseNeedSuccess2(self):
        self.a=22222222222222222222222222222222222222222222
        self.expectValue(22222222222222222222222222222222222222222222, self.a)
        longStr = ""
        for num in range(1, 10000):
            longStr = longStr + str(num)
        self.expectValue(longStr, self.x)
        longStr = ""
        longDict = {}
        longList = []
        for num in range(1, 10000):
            longStr = longStr + str(num)
            longDict[str(num)] = num
            longList.append(num)
        self.expectValue(longStr, self.x)
        self.expectValue(longDict, self.y)
        self.expectValue(longList, self.z)





