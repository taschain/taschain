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
        self.map1a = TasCollectionStorage()
        self.map1a["1111"] = True
        self.map1a["retrt"] = 111111111111111111111111111
        self.map1a["r454"] = False
        self.map1a["$$$"] = "SADDSDSDFDFDF@$#$#$#"
        #self.a["^^&"] = [1,2,3,4]
        #self.a[")()!"] = {"1222":454}
        self.map1a["null"] = None
        self.map1a["d1"] = 100
        self.map1a["d2"] = 200

        del self.map1a["d1"]
        self.setMapBaseDataSetNeedSuccess = "success"
        return 1

    @register.public()
    def getMapBaseDataSetNeedSuccess(self):
        self.expectValue(True, self.map1a["1111"])
        self.expectValue(111111111111111111111111111, self.map1a["retrt"])
        self.expectValue(False, self.map1a["r454"])
        self.expectValue("SADDSDSDFDFDF@$#$#$#", self.map1a["$$$"])
        #self.expectValue([1,2,3,4], self.a["^^&"])
        #self.expectValue({"1222":454}, self.a[")()!"])
        self.expectValue(None, self.map1a["d1"])
        self.expectValue(None, self.map1a["null"])
        del self.map1a["d2"]
        self.getMapBaseDataSetNeedSuccess = "success"
        return 1

    @register.public()
    def getMapBaseDataSetNeedSuccess2(self):
        self.expectValue(None, self.map1a["d2"])
        self.getMapBaseDataSetNeedSuccess2 = "success"
        return 1

    @register.public()
    def setMapCoverValue(self):
        self.map2a = TasCollectionStorage()
        self.map2a["1"]=1000
        self.map2a["2"]="[]"
        self.map2a["3"]="{1:2}"
        self.map2a["4"]=True
        self.map2a["4"]=self.map2a["1"]
        self.map2a["3"]=self.map2a["4"]
        self.map2a["2"]=self.map2a["3"]

        self.map2b=TasCollectionStorage()
        key1 = self.map2b
        key1["c1"] = 100
        key1["c2"] = 200
        self.setMapCoverValue = "success"
        return 1

    @register.public()
    def getMapCoverValue(self):
        self.expectValue(1000, self.map2a["1"])
        self.expectValue(1000, self.map2a["2"])
        self.expectValue(1000, self.map2a["3"])
        self.expectValue(1000, self.map2a["4"])

        self.expectValue(100, self.map2b["c1"])
        self.expectValue(200, self.map2b["c2"])
        self.getMapCoverValue = "success"
        return 1

    @register.public()
    def setMapNestIn(self):
        self.map3a = TasCollectionStorage()
        self.map3xxx = TasCollectionStorage()
        self.map3a["x1"] = TasCollectionStorage()
        self.map3a["x1"]["x12"] = "x12"
        self.map3a["x1"]["x13"] = 13
        #self.a["x1"]["x14"] = [1,2,3,4]
        self.map3a["x1"]["x16"] = "del"
        self.map3a["x1"]["x11"] = TasCollectionStorage()
        #self.a["x1"]["x11"]["121"] = {"1":2,"2":[2,3,4]}
        self.map3a["x1"]["x11"]["122"] = 200
        self.map3a["x1"]["x11"]["123"] = 300
        self.map3a["x1"]["x11"]["124"] = 400
        self.map3a["x1"]["x11"]["x111"] = TasCollectionStorage()
        self.map3a["x1"]["x11"]["x111"]["x1111"] = TasCollectionStorage()
        #self.a["x1"]["x11"]["x111"]["x1111"]["x11112"] = [1,1,2,3,{"3:":111}]
        try: #nest in too much
            self.map3a["x1"]["x11"]["x111"]["x1111"]["x11111"] = TasCollectionStorage()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        # try:
        #     self.a["x1"]["x15"] = [1, 2, 3, 4,{"1": 2, "2": [111, 2, 2, 3, {"x1": 2, "x2": [1, 2, 3], "x3": {"xx1": 1}}]}]
        # except Exception:
        #     pass
        # else:
        #     raise Exception("exception exception !")

        try:#can not remove a map
            del self.map3a["x1"]["x11"]["x111"]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")
        self.setMapNestIn = "success"
        return 1


    @register.public()
    def getMapNestIn(self):
        self.expectValue("x12", self.map3a["x1"]["x12"])
        self.expectValue(13, self.map3a["x1"]["x13"])
        #self.expectValue([1,2,3,4], self.a["x1"]["x14"])
        #self.expectValue({"1":2,"2":[2,3,4]}, self.a["x1"]["x11"]["121"])
        self.expectValue(200, self.map3a["x1"]["x11"]["122"])
        self.expectValue(300, self.map3a["x1"]["x11"]["123"])
        self.expectValue(400, self.map3a["x1"]["x11"]["124"])
        #self.expectValue([1,1,2,3,{"3:":111}],self.a["x1"]["x11"]["x111"]["x1111"]["x11112"])

        self.map3a["x1"]["x222"] = 123456
        self.map3a["x1"]["x223"] = 123456
        self.map3a["x1"]["x224"] = 123456

        self.expectValue(123456, self.map3a["x1"]["x222"])

        deldata = self.map3a["x1"]
        self.expectValue("del", deldata["x16"])
        self.expectValue("del", self.map3a["x1"]["x16"])

        try:#can not remove a map
            del self.map3a["x1"]["x11"]["x111"]
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        data = self.map3a["x1"]["x13"]
        data = data + 100
        self.expectValue(113, data)

        deldata2 = self.map3a["x1"]["x11"]
        del deldata2["122"]
        del deldata2["123"]
        del deldata2["124"]
        deldata2["125"] = 500

        if self.map3xxx["1"] == None:
            self.map3xxx["1"] = TasCollectionStorage()
            newKey = self.map3xxx["1"]
            newKey["2"] = 1000
        self.expectValue(1000, self.map3xxx["1"]["2"])
        self.getMapNestIn = "success"
        return 1

    @register.public()
    def getMapNestIn2(self):
        self.expectValue(123456, self.map3a["x1"]["x222"])
        self.expectValue(123456, self.map3a["x1"]["x223"])
        self.expectValue(123456, self.map3a["x1"]["x224"])

        self.expectValue(None, self.map3a["x1"]["x11"]["122"])
        self.expectValue(None, self.map3a["x1"]["x11"]["123"])
        self.expectValue(None, self.map3a["x1"]["x11"]["124"])
        self.expectValue(500, self.map3a["x1"]["x11"]["125"])
        self.expectValue(1000, self.map3xxx["1"]["2"])
        self.getMapNestIn2 = "success"
        return 1

    @register.public()
    def setMapErrors(self):
        self.keyTooLongError()
        self.ValueNotSupportError()
        self.setMapErrors = "success"
        return 1

    def keyTooLongError(self):
        try:
            self.aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa = TasCollectionStorage()
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.map4a = TasCollectionStorage()
            self.map4a["aaaaaaaaaaaaaasdsdsddffffffffffffffffrrgrgrggfgfgfgfgfffffffffffffff"]=10
        except Exception:
            pass
        else:
            raise Exception("exception exception !")


    def ValueNotSupportError(self):
        self.map5a = TasCollectionStorage()
        try:
            self.map5a["key1"] = TasCollectionStorage
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.map5a["1d"] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.map5a[123] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        try:
            self.map5a[False] = (1,2,4)
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

    @register.public()
    def setNull(self):
        self.mapnullm = TasCollectionStorage()
        self.mapnullm2 = ""
        self.mapnullm3 = 100
        self.mapnullm4 = 200
        self.mapnullm5 = 300
        try:
            self.mapnullm = None
        except Exception:
            pass
        else:
            raise Exception("exception exception !")

        self.mapnullm2= None
        self.mapnulln = TasCollectionStorage()
        self.setNull = "success"


    @register.public()
    def getNull1(self):
        self.mapnullm3 = None
        self.expectValue(None, self.mapnullm3)
        self.expectValue(200, self.mapnullm4)
        self.mapnullm4 = None
        try:
            self.mapnulln = None
        except Exception:
            pass
        else:
            raise Exception("exception exception !")
        self.getNull1 = "success"

    @register.public()
    def getNull2(self):
        self.expectValue(None, self.mapnullm3)
        self.expectValue(None, self.mapnullm4)
        self.expectValue(300, self.mapnullm5)
        self.getNull2 = "success"









