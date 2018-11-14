from lib.base.tas_storage_map_property import TasMapStorage
from serializable.tas_json_decoder import TasJson
import account
class TasBaseStoage:
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
        if type(value) == TasBaseStoage.TypeTasMap:
            raise Exception("can not remove a map!")

    @staticmethod
    def getDataFromDB(key):
        value = account.get_data(key)
        if value is None or value == "":
            return -1,None
        tp, value = TasBaseStoage.tasJson.decodeValue(value)
        return tp,value

    @staticmethod
    def checkRemoveData(key):
        if key in TasBaseStoage.tasMapFieldList:
            raise Exception("can not remove a map!")
        inReadData = False
        inWriteData = False
        inDb = False
        if key in TasBaseStoage.readData:
            value = TasBaseStoage.readData[key]
            TasBaseStoage.checkValueCanDel(value)
            inReadData = True

        if key in TasBaseStoage.writeData:
            value = TasBaseStoage.writeData[key]
            TasBaseStoage.checkValueCanDel(value)
            inWriteData = True


        tp, dbValue = TasBaseStoage.getDataFromDB(key)
        if tp == -1:  # db is null,
            pass
        elif tp == 0:  # this is map!cannot del
            raise Exception("can not remove a map!")
        else:
            inDb = True
        return inReadData,inWriteData,inDb

    @staticmethod
    def removeData(key):
        inReadData,inWriteData,inDb = TasBaseStoage.checkRemoveData(key)
        if inReadData:
            del TasBaseStoage.readData[key]
        if inWriteData:
            del TasBaseStoage.writeData[key]
        if inDb:
            account.remove_data(key)

    def getAttrHook(self, key):
        if key in TasBaseStoage.tasMapFieldList:
            TasJson.setVisitMapField(key)
            return TasBaseStoage.tasMapFieldList[key]
        else:
            return TasBaseStoage.getValue(key)

    def setAttrHook(self, key, value):
        TasJson.checkKey(key)
        if value is None:
            TasBaseStoage.removeData(key)
        else:
            if TasBaseStoage.TypeTasMap == type(value):
                TasBaseStoage.tasMapFieldList[key] = value
            else:
                TasBaseStoage.checkValue(value)
                if key in TasBaseStoage.tasMapFieldList:
                    del TasBaseStoage.tasMapFieldList[key]
                TasBaseStoage.readData[key]=value
                TasBaseStoage.writeData[key] = value

    @staticmethod
    def checkValue(value):
        TasJson.checkBaseValue(value,1)


    @staticmethod
    def getValue(key):
        #get value from memory
        if key in TasBaseStoage.readData:
            return TasBaseStoage.readData[key]
        else:#get value from db
            value = account.get_data(key)
            if value is None or value == "":
                return None
            else:#put db data into memory
                tp,value = TasBaseStoage.tasJson.decodeValue(value)
                if tp == 0:
                    TasJson.setVisitMapField(key)
                    mapInstance = TasMapStorage()
                    TasBaseStoage.tasMapFieldList[key] = mapInstance
                    return mapInstance
                TasBaseStoage.readData[key]=value
                return value


    #after call will call this function
    @staticmethod
    def flushData():
       for k in TasBaseStoage.writeData:
           #print(TasBaseStoage.tasJson.encodeValue(1,TasBaseStoage.writeData[k]))
           account.set_data(k,TasBaseStoage.tasJson.encodeValue(1,TasBaseStoage.writeData[k]))
       for k in TasBaseStoage.tasMapFieldList:
           account.set_data(k, TasBaseStoage.tasJson.encodeValue(0, "0"))
           TasBaseStoage.tasMapFieldList[k].flushData(k)
