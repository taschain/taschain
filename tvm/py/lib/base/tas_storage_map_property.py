import account
class TasCollectionStorage:
    tasJson = TasJson()

    def __init__(self,nestin =  1):
        self.readData = {}  # only get,not flush to db
        self.writeData = {}  # write to db
        self.nestIn = nestin  #max nestin map

    def __setitem__(self, key, value):
        TasJson.checkMapKey(key)
        if value is None:
            self.removeData(key)
        else:
            self.checkValue(value)
            self.readData[key] = value
            self.writeData[key] = value

    def checkValueCanDel(self,value):
        if type(value) == type(self):
            raise LibException("can not remove a map!",5)


    def checkRemoveData(self,key):
        inReadData = False
        inWriteData = False
        inDb = False
        if key in self.readData:
            value = self.readData[key]
            self.checkValueCanDel(value)
            inReadData = True

        if key in self.writeData:
            value = self.writeData[key]
            self.checkValueCanDel(value)
            inWriteData = True

        dbKey = TasJson.getDbKey() + "@" + key
        tp, dbValue = self.getDataFromDB(dbKey)
        if tp == -1:  # db is null,
            pass
        elif tp == 0:  # this is map!cannot del
            raise LibException("can not remove a map!",4)
        else:
            inDb = True
        return inReadData,inWriteData,inDb


    def removeData(self,key):
        inReadData,inWriteData,inDb = self.checkRemoveData(key)
        if inReadData:
            del self.readData[key]
        if inWriteData:
            del self.writeData[key]
        if inDb:
            dbKey = TasJson.getDbKey() + "@" + key
            account.remove_data(dbKey)

    def __delitem__(self, key):
       self.removeData(key)

    def __iter__(self):
        return None

    def __getitem__(self, key):
        TasJson.checkMapKey(key)
        TasJson.setVisitMapKey(key)
        return self.getValue(key)

    def getDataFromDB(self,key):
        value = account.get_data(key)
        if value is None or value == "":
            return -1,None
        tp, value = TasCollectionStorage.tasJson.decodeValue(value)
        return tp,value

    def getValue(self,key):
        if key in self.readData:
            return self.readData[key]
        else:#get value from db
            dbKey = TasJson.getDbKey()
            tp, value = self.getDataFromDB(dbKey)
            if tp == -1:
                return None
            elif tp == 0:#put db data into memory(this is map)
                value = TasCollectionStorage()
                self.writeData[key]=value
            self.readData[key] = value
            return value

    def checkValue(self,value):
        if type(value) == type(self):
            if self.nestIn + 1> 5:
                raise LibException("map can not be more than nested 5",3)
            self.nestIn += 1
            value.nestIn = self.nestIn
            pass
        else:
            TasJson.checkBaseValue(value,1)


    def flushData(self,fieldName):
        for k in self.writeData:
            newKey=fieldName+"@" + k
            toWriteData = self.writeData[k]
            if type(toWriteData) == type(self):
                account.set_data(newKey, TasCollectionStorage.tasJson.encodeValue(0, "0"))
                toWriteData.flushData(newKey)
            else:
                account.set_data(newKey, TasCollectionStorage.tasJson.encodeValue(1,self.writeData[k]))


# class SysNormalIter:
#     def __init__(self,father):
#         self.iter = account.get_iterator(TasJson.getDbKey())
#         self.iterFromMem(father,TasJson.getDbKey())
#         self.relaceStr = TasJson.getDbKey()+"@"
#         self.mem = {}
#
#     def iterFromMem(self,father,ks):
#         for k in father.writeData:
#             newKey = ks+ "@" + k
#             toWriteData = father.writeData[k]
#             if type(toWriteData) == type(father):
#                 self.iterFromMem(toWriteData,newKey)
#             else:
#                 self.mem[newKey] = toWriteData
#
#     def getNextKV(self):
#         vl = account.get_iterator_next(self.iter)
#         jsondata = TasCollectionStorage.tasJson.decodeNormal(vl)
#         hasValue = jsondata['hasValue']#1normalvalue 0:null data 2:map node
#         if hasValue == 0 :
#             if  len(self.mem) == 0:#if memory and db all null then return
#                 raise StopIteration
#             memValue = None
#             memKey = ""
#             for key,value in self.mem.items():#if db is null,then get data from memory
#                 memValue = value
#                 memKey = key
#                 break
#             del self.mem[memKey]
#             newKey = memKey.replace(self.relaceStr, "", 1)
#             return newKey,memValue
#         elif hasValue == 2:#this is map node
#             return None, None
#         value = jsondata['value']
#         key = jsondata['key']
#         if value == "":  # this is root node
#             return None,None
#         if key in self.mem:#check from memory if thie key exists in memory
#             memValue = self.mem[key]
#             del self.mem[key]
#             return key,memValue
#         newKey = key.replace(self.relaceStr,"",1)
#         return newKey,value
#
#     def __next__(self):
#         key,vl = self.getNextKV()
#         while vl is None:
#             key,vl = self.getNextKV()
#         return key,vl



