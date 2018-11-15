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
        self.basea1=1
        self.baseb1="iamstr"
        self.basec1=True
        #self.d=[1,2,3]
        #self.e={"1":"2","3":"4"}
        self.expectValue(1,self.basea1)
        self.expectValue("iamstr", self.baseb1)
        self.expectValue(True, self.basec1)
        #self.expectValue([1,2,3], self.d)
        #self.expectValue({"1":"2","3":"4"}, self.e)
        self.basenoneVl=None
        self.setBaseNeedSuccess1 = "success"
        return 1

    @register.public()
    def getBaseNeedSuccess1(self):
        self.expectValue(1, self.basea1)
        self.expectValue("iamstr", self.baseb1)
        self.expectValue(True, self.basec1)
        #self.expectValue([1, 2, 3], self.d)
        #self.expectValue({"1": "2", "3": "4"}, self.e)
        self.expectValue(None, self.basenoneVl)
        self.getBaseNeedSuccess1 = "success"
        return 1

    @register.public()
    def setBaseNeedSuccess2(self):
        self.basea2=111111111111111111111111111111111111111111111
        self.expectValue(111111111111111111111111111111111111111111111, self.basea2)
        #self.a=[1,2,3,4,5,6,1,"244444444444444444444444444444444444444444444444444444444","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]]
        #self.expectValue([1,2,3,4,5,6,1,"244444444444444444444444444444444444444444444444444444444","3","4",{"1":2},{"3":4},[1,2,3,{"4":5}]], self.a)

        longStr = ""
        longDict={}
        longList=[]
        for num in range(1, 100):
            longStr = longStr+str(num)
            longDict[str(num)]=num
            longList.append(num)
        self.basex2=longStr
        #self.y = longDict
        #self.z = longList
        self.expectValue(longStr,self.basex2)
        #self.expectValue(longDict, self.y)
        #self.expectValue(longList, self.z)
        self.setBaseNeedSuccess2 = "success"
        return 1

    @register.public()
    def getBaseNeedSuccess2(self):
        self.basea2=22222222222222222222222222222222222222222222
        self.expectValue(22222222222222222222222222222222222222222222, self.basea2)
        longStr = ""
        for num in range(1, 100):
            longStr = longStr + str(num)
        self.expectValue(longStr, self.basex2)
        #longStr = ""
        #longDict = {}
        #longList = []
        # for num in range(1, 100):
        #     longStr = longStr + str(num)
            #longDict[str(num)] = num
            #longList.append(num)
        # self.expectValue(longStr, self.x)
        #self.expectValue(longDict, self.y)
        #self.expectValue(longList, self.z)
        self.getBaseNeedSuccess2 = "success"
        return 1

    @register.public()
    def setBaseNeedSuccess3(self):
        self.basea3 = "32343434"
        self.baseb3 = "444444"
        self.baset3 = 55555555555
        self.basey5 = 100
        #self.x = [1, 2, 3, 4, 5, 6, 1, "2", "3", "4", {"1": 2}, {"3": 4}, [1, 2, 3, {"4": 5}]]
        #self.z ={"1":"2","3":"4","3":"11000","2":[1,12,3]}
        #self.zz = {"1": "2", "3": "4"}
        #self.ll = {"1": "2", "3": "4"}
        self.basea3 = self.baseb3
        self.baset3 = self.basey5

        #self.b = self.x

        self.basey5-=10

        self.expectValue(90, self.basey5)

        self.expectValue("444444", self.basea3)

        #self.expectValue([1, 2, 3, 4, 5, 6, 1, "2", "3", "4", {"1": 2}, {"3": 4}, [1, 2, 3, {"4": 5}]], self.b)
        self.setBaseNeedSuccess3 = "success"
        return 1

    @register.public()
    def getBaseNeedSuccess3(self):
        self.expectValue("444444", self.basea3)
        self.basey5 += 10
        self.expectValue(100, self.basey5)
        #self.expectValue({"1":"2","3":"11000","2":[1,12,3]}, self.z)
        #self.zz["5555"]=6666
        #self.expectValue({"1": "2", "3": "4","5555":6666},self.zz)
        #del self.ll["1"]
        #del self.ll["3"]
        #self.expectValue({},self.ll)
        self.getBaseNeedSuccess3 = "success"
        return 1

    @register.public()
    def setBaseNeedSuccess4(self):
        self.base4nil = "32343434"
        self.base4true = "false"
        self.base4this = "this"
        self.base4self = "self"
        self.base4o = "[{{{}}}}]]]]]]]][[[[[[[[[].&%^%)()()@@@!*())()(&&^%#$#$#@#&*SHSS(*SSHDS(SDD&&*&*88721*&GS}}S?>MLKLSJB*^S&S^S%S$S%$SS#AISHS(SS****S"
        self.base4f="."
        self.base4y=""
        #self.x1=[]
        #self.y1={}
        self.setBaseNeedSuccess4 = "success"
        return 1

    @register.public()
    def getBaseNeedSuccess4(self):
        self.expectValue("32343434", self.base4nil)
        self.expectValue("false", self.base4true)
        self.expectValue("this", self.base4this)
        self.expectValue("self", self.base4self)
        self.expectValue("[{{{}}}}]]]]]]]][[[[[[[[[].&%^%)()()@@@!*())()(&&^%#$#$#@#&*SHSS(*SSHDS(SDD&&*&*88721*&GS}}S?>MLKLSJB*^S&S^S%S$S%$SS#AISHS(SS****S", self.base4o)
        self.expectValue(".", self.base4f)
        self.expectValue("", self.base4y)
        #self.expectValue([], self.x1)
        #self.expectValue({}, self.y1)
        self.getBaseNeedSuccess4 = "success"
        return 1

    @register.public()
    def setChangeKey(self):
        self.base5x1 = "this is x1"
        a=self.base5x1
        a = ".........."
        #self.x2=[1,2,3,4,5]
        #b=self.x2
        #b.append(6)

        #self.x3={1:2,"2":"3",3:4}
        #c=self.x3
        #del c["2"]
        self.setChangeKey = "success"
        return 1

    @register.public()
    def getChangeKey(self):
        self.expectValue("this is x1", self.base5x1)
        #self.expectValue([1,2,3,4,5,6], self.x2)
        #self.expectValue({1:2,3:4}, self.x3)
        self.getChangeKey = "success"
        return 1

    @register.public()
    def baseErrors(self):
        self.keyTooLongError()
        self.keyNotExistsError()
        self.mapNestInTooMuchError()
        self.listNestInTooMuchError()
        self.notSupportTypeError()
        self.baseErrors = "success"
        return 1


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
        pass
        # try:
        #     self.a=[1,2,334343,"43e334",[1],{"1":222,3232:33,"444":{444:444,"555":"=>","444":{"111":444,"44":[333,[33]]}}}]
        # except Exception:
        #     pass
        # else:
        #     raise Exception("exception exception !")







