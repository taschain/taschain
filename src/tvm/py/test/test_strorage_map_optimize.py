import account
class ContractMapStorage():
    def expectValue(self,expectValue,trueValue):
        if expectValue != trueValue:
            raise Exception("error value,expectValue="+str(expectValue)+".getvalue is:" + str(trueValue))

    def deploy(self):
        pass
        #print("deploy ok")

    @register.public()
    def setMapBaseDataSetNeedSuccess(self):
        self.a = TasMapStorage()
        self.a["1111"] = True
        self.a["retrt"] = 111111111111111111111111111
        self.a["r454"] = False
        self.a["$$$"] = "SADDSDSDFDFDF@$#$#$#"
        self.a["^^&"] = [1,2,3,4]
        self.a[")()!"] = {"1222":454}
        self.a["null"] = None
        self.a["d1"] = 100
        self.a["d2"] = 200

        del self.a["d1"]
        return 1

    @register.public()
    def getMapBaseDataSetNeedSuccess(self):
        self.expectValue(True, self.a["1111"])
        self.expectValue(111111111111111111111111111, self.a["retrt"])
        self.expectValue(False, self.a["r454"])
        self.expectValue("SADDSDSDFDFDF@$#$#$#", self.a["$$$"])
        self.expectValue([1,2,3,4], self.a["^^&"])
        self.expectValue({"1222":454}, self.a[")()!"])
        self.expectValue(None, self.a["d1"])
        self.expectValue(None, self.a["null"])
        del self.a["d2"]
        return 1

    @register.public()
    def getMapBaseDataSetNeedSuccess2(self):
        self.expectValue(None, self.a["d2"])
        return 1

    @register.public()
    def setMapCoverValue(self):
        self.a = TasMapStorage()
        self.a["1"]=1000
        self.a["2"]=[]
        self.a["3"]={1:2}
        self.a["4"]=True
        self.a["4"]=self.a["1"]
        self.a["3"]=self.a["4"]
        self.a["2"]=self.a["3"]

        self.b=TasMapStorage()
        key1 = self.b
        key1["c1"] = 100
        key1["c2"] = 200
        return 1

    @register.public()
    def getMapCoverValue(self):
        self.expectValue(1000, self.a["1"])
        self.expectValue(1000, self.a["2"])
        self.expectValue(1000, self.a["3"])
        self.expectValue(1000, self.a["4"])

        self.expectValue(100, self.b["c1"])
        self.expectValue(200, self.b["c2"])
        return 1

    @register.public()
    def setMapNestIn(self):
        self.a = TasMapStorage()
        self.xxx = TasMapStorage()
        self.a["x1"] = TasMapStorage()
        self.a["x1"]["x12"] = "x12"
        self.a["x1"]["x13"] = 13
        self.a["x1"]["x14"] = [1,2,3,4]
        self.a["x1"]["x16"] = "del"
        self.a["x1"]["x11"] = TasMapStorage()
        self.a["x1"]["x11"]["121"] = {"1":2,"2":[2,3,4]}
        self.a["x1"]["x11"]["122"] = 200
        self.a["x1"]["x11"]["123"] = 300
        self.a["x1"]["x11"]["124"] = 400
        self.a["x1"]["x11"]["x111"] = TasMapStorage()
        self.a["x1"]["x11"]["x111"]["x1111"] = TasMapStorage()
        self.a["x1"]["x11"]["x111"]["x1111"]["x11112"] = [1,1,2,3,{"3:":111}]
        try: #nest in too much
            self.a["x1"]["x11"]["x111"]["x1111"]["x11111"] = TasMapStorage()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a["x1"]["x15"] = [1, 2, 3, 4,{"1": 2, "2": [111, 2, 2, 3, {"x1": 2, "x2": [1, 2, 3], "x3": {"xx1": 1}}]}]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:#can not remove a map
            del self.a["x1"]["x11"]["x111"]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")
        return 1


    @register.public()
    def getMapNestIn(self):
        self.expectValue("x12", self.a["x1"]["x12"])
        self.expectValue(13, self.a["x1"]["x13"])
        self.expectValue([1,2,3,4], self.a["x1"]["x14"])
        self.expectValue({"1":2,"2":[2,3,4]}, self.a["x1"]["x11"]["121"])
        self.expectValue(200, self.a["x1"]["x11"]["122"])
        self.expectValue(300, self.a["x1"]["x11"]["123"])
        self.expectValue(400, self.a["x1"]["x11"]["124"])
        self.expectValue([1,1,2,3,{"3:":111}],self.a["x1"]["x11"]["x111"]["x1111"]["x11112"])

        self.a["x1"]["x222"] = 123456
        self.a["x1"]["x223"] = 123456
        self.a["x1"]["x224"] = 123456

        self.expectValue(123456, self.a["x1"]["x222"])

        deldata = self.a["x1"]
        self.expectValue("del", deldata["x16"])
        self.expectValue("del", self.a["x1"]["x16"])

        try:#can not remove a map
            del self.a["x1"]["x11"]["x111"]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        data = self.a["x1"]["x13"]
        data = data + 100
        self.expectValue(113, data)

        deldata2 = self.a["x1"]["x11"]
        del deldata2["122"]
        del deldata2["123"]
        del deldata2["124"]
        deldata2["125"] = 500

        if self.xxx["1"] == None:
            self.xxx["1"] = TasMapStorage()
            newKey = self.xxx["1"]
            newKey["2"] = 1000
        self.expectValue(1000, self.xxx["1"]["2"])
        return 1

    @register.public()
    def getMapNestIn2(self):
        self.expectValue(123456, self.a["x1"]["x222"])
        self.expectValue(123456, self.a["x1"]["x223"])
        self.expectValue(123456, self.a["x1"]["x224"])

        self.expectValue(None, self.a["x1"]["x11"]["122"])
        self.expectValue(None, self.a["x1"]["x11"]["123"])
        self.expectValue(None, self.a["x1"]["x11"]["124"])
        self.expectValue(500, self.a["x1"]["x11"]["125"])
        self.expectValue(1000, self.xxx["1"]["2"])
        return 1

    @register.public()
    def setMapErrors(self):
        self.keyTooLongError()
        self.ValueNotSupportError()
        return 1

    def keyTooLongError(self):
        try:
            self.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa = TasMapStorage()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a = TasMapStorage()
            self.a["aaaaaaaaaaaaaasdsdsddffffffffffffffffrrgrgrggfgfgfgfgfffffffffffffff"]=10
        except Exception:
            pass
        else:
            raise Exception("exception exception !")


    def ValueNotSupportError(self):
        self.a = TasMapStorage()
        try:
            self.a["key1"] = TasMapStorage
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a["1d"] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a[123] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.a[False] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")










