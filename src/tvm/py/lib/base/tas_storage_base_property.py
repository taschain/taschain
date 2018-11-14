from lib.base.tas_storage_map_property import TasMapStorage
from serializable.tas_json_decoder import TasJson
import account
class TasBaseStorage:
    readData = {} #only get,not flush to db
    writeData={}  #write to db
    tasJson=TasJson()
    currentViterKey=""
    TypeTasMap=type(TasMapStorage())
    tasMapFieldList = {}

    def initHook(self):
        pass

    @staticmethod
    def checkValueCanDel(value):
        if type(value) == TasBaseStorage.TypeTasMap:
            raise Exception("can not remove a map!")

    @staticmethod
    def getDataFromDB(key):
        value = account.get_data(key)
        if value is None or value == "":
            return -1,None
        tp, value = TasBaseStorage.tasJson.decodeValue(value)
        return tp,value

    @staticmethod
    def checkRemoveData(key):
        if key in TasBaseStorage.tasMapFieldList:
            raise Exception("can not remove a map!")
        inReadData = False
        inWriteData = False
        inDb = False
        if key in TasBaseStorage.readData:
            value = TasBaseStorage.readData[key]
            TasBaseStorage.checkValueCanDel(value)
            inReadData = True

        if key in TasBaseStorage.writeData:
            value = TasBaseStorage.writeData[key]
            TasBaseStorage.checkValueCanDel(value)
            inWriteData = True


        tp, dbValue = TasBaseStorage.getDataFromDB(key)
        if tp == -1:  # db is null,
            pass
        elif tp == 0:  # this is map!cannot del
            raise Exception("can not remove a map!")
        else:
            inDb = True
        return inReadData,inWriteData,inDb

    @staticmethod
    def removeData(key):
        inReadData,inWriteData,inDb = TasBaseStorage.checkRemoveData(key)
        if inReadData:
            del TasBaseStorage.readData[key]
        if inWriteData:
            del TasBaseStorage.writeData[key]
        if inDb:
            account.remove_data(key)

    def getAttrHook(self, key):
        if key in TasBaseStorage.tasMapFieldList:
            TasJson.setVisitMapField(key)
            return TasBaseStorage.tasMapFieldList[key]
        else:
            return TasBaseStorage.getValue(key)

    def setAttrHook(self, key, value):
        TasJson.checkKey(key)
        if value is None:
            TasBaseStorage.removeData(key)
        else:
            if TasBaseStorage.TypeTasMap == type(value):
                TasBaseStorage.tasMapFieldList[key] = value
            else:
                TasBaseStorage.checkValue(value)
                if key in TasBaseStorage.tasMapFieldList:
                    del TasBaseStorage.tasMapFieldList[key]
                TasBaseStorage.readData[key]=value
                TasBaseStorage.writeData[key] = value

    @staticmethod
    def checkValue(value):
        TasJson.checkBaseValue(value,1)


    @staticmethod
    def getValue(key):
        #get value from memory
        if key in TasBaseStorage.readData:
            return TasBaseStorage.readData[key]
        else:#get value from db
            value = account.get_data(key)
            if value is None or value == "":
                return None
            else:#put db data into memory
                tp,value = TasBaseStorage.tasJson.decodeValue(value)
                if tp == 0:
                    TasJson.setVisitMapField(key)
                    mapInstance = TasMapStorage()
                    TasBaseStorage.tasMapFieldList[key] = mapInstance
                    return mapInstance
                TasBaseStorage.readData[key]=value
                return value


    #after call will call this function
    @staticmethod
    def flushData():
       for k in TasBaseStorage.writeData:
           #print(TasBaseStorage.tasJson.encodeValue(1,TasBaseStorage.writeData[k]))
           account.set_data(k,TasBaseStorage.tasJson.encodeValue(1,TasBaseStorage.writeData[k]))
       for k in TasBaseStorage.tasMapFieldList:
           account.set_data(k, TasBaseStorage.tasJson.encodeValue(0, "0"))
           TasBaseStorage.tasMapFieldList[k].flushData(k)
