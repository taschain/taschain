import account
class ContractStorage():
    def __init__(self):
        print("==================init----------------")

    def expectValue(self,expectValue,trueValue):
        if expectValue != trueValue:
            raise Exception("error value,expectValue="+str(expectValue)+".getvalue is:" + str(trueValue))

    def deploy(self):
        pass

    @register.public()
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

    @register.public()
    def getBaseNeedSuccess1(self):
        self.expectValue(1, self.a)
        self.expectValue("iamstr", self.b)
        self.expectValue(True, self.c)
        self.expectValue([1, 2, 3], self.d)
        self.expectValue({"1": "2", "3": "4"}, self.e)

    @register.public()
    def setBaseNeedSuccess2(self):
        self.a=111111111111111111111111111111111111111111111
        self.expectValue(111111111111111111111111111111111111111111111, self.a)
        self.a=[1,2,3,4,5,6,1,"244444444444444444444444444444444444444444444444444444444","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]]
        self.expectValue([1,2,3,4,5,6,1,"244444444444444444444444444444444444444444444444444444444","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]], self.a)

        longStr = ""
        longDict={}
        longList=[]
        for num in range(1, 100):
            longStr = longStr+str(num)
            longDict[str(num)]=num
            longList.append(num)
        self.x=longStr
        self.y = longDict
        self.z = longList
        self.expectValue(longStr,self.x)
        self.expectValue(longDict, self.y)
        self.expectValue(longList, self.z)

    @register.public()
    def getBaseNeedSuccess2(self):
        self.a=22222222222222222222222222222222222222222222
        self.expectValue(22222222222222222222222222222222222222222222, self.a)
        longStr = ""
        for num in range(1, 100):
            longStr = longStr + str(num)
        self.expectValue(longStr, self.x)
        longStr = ""
        longDict = {}
        longList = []
        for num in range(1, 100):
            longStr = longStr + str(num)
            longDict[str(num)] = num
            longList.append(num)
        self.expectValue(longStr, self.x)
        self.expectValue(longDict, self.y)
        self.expectValue(longList, self.z)

    @register.public()
    def setBaseNeedSuccess3(self):
        self.a = "32343434"
        self.b = "444444"
        self.t = 55555555555
        self.y = 100
        self.x = [1, 2, 3, 4, 5, 6, 1, "2", "3", "4", {"1": 2}, {"3": 4}, [1, 2, 3, {"4": 5}]]
        self.z ={"1":"2","3":"4","3":"11000","2":[1,12,3]}
        self.zz = {"1": "2", "3": "4"}
        self.ll = {"1": "2", "3": "4"}
        self.a = self.b
        self.t = self.y

        self.b = self.x

        self.y-=10

        self.expectValue(90, self.y)

        self.expectValue("444444", self.a)

        self.expectValue([1, 2, 3, 4, 5, 6, 1, "2", "3", "4", {"1": 2}, {"3": 4}, [1, 2, 3, {"4": 5}]], self.b)

    @register.public()
    def getBaseNeedSuccess3(self):
        self.expectValue("444444", self.a)
        self.y += 10
        self.expectValue(100, self.y)
        self.expectValue({"1":"2","3":"11000","2":[1,12,3]}, self.z)
        self.zz["5555"]=6666
        self.expectValue({"1": "2", "3": "4","5555":6666},self.zz)
        del self.ll["1"]
        del self.ll["3"]
        self.expectValue({},self.ll)

    @register.public()
    def setBaseNeedSuccess4(self):
        self.nil = "32343434"
        self.true = "false"
        self.this = "this"
        self.self = "self"
        self.o = "[{{{}}}}]]]]]]]][[[[[[[[[].&%^%)()()@@@!*())()(&&^%#$#$#@#&*SHSS(*SSHDS(SDD&&*&*88721*&GS}}S?>MLKLSJB*^S&S^S%S$S%$SS#AISHS(SS****S"
        self.f="."
        self.y=""
        self.x1=[]
        self.y1={}

    @register.public()
    def getBaseNeedSuccess4(self):
        self.expectValue("32343434", self.nil)
        self.expectValue("false", self.true)
        self.expectValue("this", self.this)
        self.expectValue("self", self.self)
        self.expectValue("[{{{}}}}]]]]]]]][[[[[[[[[].&%^%)()()@@@!*())()(&&^%#$#$#@#&*SHSS(*SSHDS(SDD&&*&*88721*&GS}}S?>MLKLSJB*^S&S^S%S$S%$SS#AISHS(SS****S", self.o)
        self.expectValue(".", self.f)
        self.expectValue("", self.y)
        self.expectValue([], self.x1)
        self.expectValue({}, self.y1)

    @register.public()
    def setChangeKey(self):
        self.x1 = "this is x1"
        a=self.x1
        a = ".........."
        self.x2=[1,2,3,4,5]
        b=self.x2
        b.append(6)

        self.x3={1:2,"2":"3",3:4}
        c=self.x3
        del c["2"]

    @register.public()
    def getChangeKey(self):
        self.expectValue("this is x1", self.x1)
        self.expectValue([1,2,3,4,5,6], self.x2)
        self.expectValue({1:2,3:4}, self.x3)

    @register.public()
    def baseErrors(self):
        self.keyTooLongError()
        self.keyNotExistsError()
        self.mapNestInTooMuchError()
        self.listNestInTooMuchError()
        self.notSupportTypeError()


    def keyTooLongError(self):
        try:
            self.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa = 100
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

    def keyNotExistsError(self):
        try:
            print(self.tttttt + 1)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

    def mapNestInTooMuchError(self):
        try:
            self.a={"xeeeeeee":1,"rerefdff":2,"reree":{"2":3},"pppppp[][][]":[1,12,3,{"11":333,"222":[1,2,3,{"-----":"wqwqwqwq","nnnn":[44,555,"saddf",{"4444":555,"11":{"4444":555}}]}]}]}
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

    def notSupportTypeError(self):
        try:
            self.a1=self
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a2 = ContractStorage
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a3 = ()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a4 = ContractStorage()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")



    def listNestInTooMuchError(self):
        try:
            self.a=[1,2,334343,"43e334",[1],{"1":222,3232:33,"444":{444:444,"555":"=>","444":{"111":444,"44":[333,[33]]}}}]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")







