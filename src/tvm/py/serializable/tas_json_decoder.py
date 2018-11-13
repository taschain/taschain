import ujson
class TasJson:
    mapFieldName = ""
    mapKey=""
    TypeInt = type(1)
    TypeBool = type(True)
    TypeStr = type("")
    TypeList = type([])
    TypeDict = type({})
    TypeNone = type(None)
    supportType = [TypeInt, TypeBool, TypeStr, TypeList, TypeDict,TypeNone]

    @staticmethod
    def setVisitMapField(key):
        TasJson.mapFieldName=key
        TasJson.clearMapKey()

    @staticmethod
    def setVisitMapKey(key):
        if TasJson.mapKey != "":
            TasJson.mapKey = TasJson.mapKey + "@" + key
        else:
            TasJson.mapKey = key

    @staticmethod
    def clearMapKey():
        TasJson.mapKey = ""

    @staticmethod
    def getDbKey():
        if TasJson.mapKey != "":
            return TasJson.mapFieldName +"@"+ TasJson.mapKey
        return TasJson.mapFieldName

    def decodeValue(self,value):
        if value.startswith('0'):
            return 0,""
        value = value.replace("1","",1)
        data = ujson.loads(value)
        return 1,data

    def decodeNormal(self,value):
        data = ujson.loads(value)
        return data


    def encodeValue(self,type,value):
        if type == 0: #this is map
            return "0"
        else:
            return "1"+ ujson.dumps(value)

    @staticmethod
    def checkBaseValue(value, currentDeep):
        if currentDeep > 5:
            raise Exception("map can not be more than nested 5")
        valueType = type(value)
        TasJson.checkValueIsInBase(valueType)
        if valueType == TasJson.TypeList:
            TasJson.checkListValue(value, currentDeep)
        elif valueType == TasJson.TypeDict:
            TasJson.checkDictValue(value, currentDeep)

    @staticmethod
    def checkDictValue(value, currentDeep):
        for key,data in value.items():
            TasJson.checkBaseValue(data, currentDeep + 1)

    @staticmethod
    def checkListValue(value, currentDeep):
        for data in value:
            TasJson.checkBaseValue(data, currentDeep + 1)

    @staticmethod
    def checkValueIsInBase(valueType):
        if valueType not in TasJson.supportType:
            raise Exception("value must be int,bool,string,list,dict,type is " + str(valueType))

    @staticmethod
    def checkKey(key):
        if type(key) != TasJson.TypeStr:
            raise Exception("key must be string")
        x = bytes(key, "utf-8")
        if len(x) > 32:
            raise Exception("the length of key cannot more than 32!")

    @staticmethod
    def checkMapKey(key):
        if type(key) != TasJson.TypeStr:
            raise Exception("key must be string")
        x = bytes(key, "utf-8")
        if len(x) > 45:
            raise Exception("the length of key cannot more than 45!")
