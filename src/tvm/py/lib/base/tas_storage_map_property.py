import account
from serializable.tas_json_decoder import TasJson
class TasMapStorage:
    TypeInt = type(1)
    TypeBool = type(True)
    TypeStr = type("")
    TypeList = type([])
    TypeDict = type({})
    supportType = [TypeInt, TypeBool, TypeStr, TypeList, TypeDict]
    tasJson = TasJson()
    size=0

    def __init__(self,nestin =  1):
        self.readData = {}  # only get,not flush to db
        self.writeData = {}  # write to db
        self.nestIn = nestin  #max nestin map

    def __setitem__(self, key, value):
        TasJson.checkKey(key)
        self.checkValue(value)
        self.readData[key] = value
        self.writeData[key] = value
        #self.checkSyntax(key,value)

    # def checkSyntax(self,key,value):
    #     if key in self.readData:
    #         currentVl = self.readData[key]
    #         if type(TasMapStorage()) == type(value) == type(currentVl):
    #             raise Exception("Cannot set dict property to another dict")


    def __iter__(self):
        it = SysNormalIter(self)
        return it

    def items(self):
        return self

    def __getitem__(self, key):
        TasJson.checkKey(key)
        TasJson.setVisitMapKey(key)
        return self.getValue(key)

    def getValue(self,key):
        #get value from memory
        if key in self.readData:
            return self.readData[key]
        else:#get value from db
            dbKey = TasJson.getDbKey()
            value = account.get_data(dbKey)
            if value == None or value == "":
                return None
            else:#put db data into memory
                tp,value = TasMapStorage.tasJson.decodeValue(value)
                if tp == 0:
                    value = TasMapStorage()
                    self.writeData[key]=value
                self.readData[key]=value
                return value

    def checkValue(self,value):
        if type(value) == type(self):
            if self.nestIn + 1> 5:
                raise Exception("map can not be more than nested 5")
            self.nestIn += 1
            value.nestIn = self.nestIn
            pass
        else:
            TasJson.checkBaseValue(value,1)


    def flushData(self,fieldName):
        for k in self.writeData:
            newKey=fieldName+"_" + k
            toWriteData = self.writeData[k]
            if type(toWriteData) == type(self):
                account.set_data(newKey, TasMapStorage.tasJson.encodeValue(0, "0"))
                toWriteData.flushData(newKey)
            else:
                #print(TasMapStorage.tasJson.encodeValue(1,self.writeData[k]))
                account.set_data(newKey, TasMapStorage.tasJson.encodeValue(1,self.writeData[k]))


class SysNormalIter:
    def __init__(self,father):
        self.iter = account.get_iterator(TasJson.getDbKey())
        self.iterFromMem(father,TasJson.getDbKey())

    def iterFromMem(self,father,ks):
        self.mem = {}
        for k in father.writeData:
            newKey = ks+ "_" + k
            toWriteData = father.writeData[k]
            if type(toWriteData) == type(father):
                self.iterFromMem(toWriteData,newKey)
            else:
                self.mem[newKey] = toWriteData

    def getNextKV(self):
        vl = account.get_iterator_next(self.iter)
        jsondata = TasMapStorage.tasJson.decodeNormal(vl)
        hasValue = jsondata['hasValue']
        if hasValue == 0 :
            if  len(self.mem) == 0:#if memory and db all null then return
                raise StopIteration
            memValue = None
            memKey = ""
            for key,value in self.mem.items():#if db is null,then get data from memory
                memValue = value
                memKey = key
                break
            del self.mem[memKey]
            return memKey,memValue
        value = jsondata['value']
        key = jsondata['key']
        if value == "":  # this is root node
            return None,None
        if value['tp'] == 0:#if this is 0,this value is map,not leaf node
            return None,None
        if key in self.mem:#check from memory if thie key exists in memory
            memValue = self.mem[key]
            del self.mem[key]
            return key,memValue
        return key,value['vl']

    def __next__(self):
        key,vl = self.getNextKV()
        while vl is None:
            key,vl = self.getNextKV()
        return key,vl



