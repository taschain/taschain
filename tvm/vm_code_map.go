package tvm

import (
	"fmt"
	"strings"
)

func PycodeStoreContractData() string {
	return fmt.Sprintf(`
try:
    TasBaseStorage.flushData()
except Exception as e:
    pass
`)
}

func PycodeCreateContractInstance(code string, contractName string) (string, int) {
	trueCode, libLine := PycodeGetTrueUserCode(code)
	newCode := fmt.Sprintf(`%s%s
try:
    tas_%s = %s()
except Exception:
    raise ABICheckException("ABI input contract name error,input contract name is %s")`, trueCode, PycodeContractAddHooks(contractName), contractName, contractName, contractName)
	return newCode, libLine
}

func PycodeContractImports() string {
	newCode := fmt.Sprintf(`
%s
%s
%s`, TasJson(), TasCollectionStorageCode(), TasBaseStorageCode())
	return newCode
}

func PycodeContractAddHooks(contractName string) string {
	return fmt.Sprintf(`
try:
    %s.__init__ = TasBaseStorage.initHook
    %s.__setattr__= TasBaseStorage.setAttrHook
    %s.__getattr__= TasBaseStorage.getAttrHook
except Exception:
    raise ABICheckException("ABI input contract name error,input contract name is %s")
`, contractName, contractName, contractName, contractName)
}

func PycodeContractDeployHooks(contractName string) string {
	return fmt.Sprintf(`
try:
    %s.__setattr__= TasBaseStorage.setAttrHook
    %s.__getattr__= TasBaseStorage.getAttrHook
except Exception:
    raise ABICheckException("ABI input contract name error,input contract name is %s")
`, contractName, contractName, contractName)

}

func PycodeGetTrueUserCode(code string) (string, int) {
	codeLen := calCodeLines(PycodeContractImports())
	usercode := fmt.Sprintf(`%s%s`, PycodeContractImports(), code)
	return usercode, codeLen
}

func PycodeContractDeploy(code string, contractName string) (string, int) {
	trueCode, libLine := PycodeGetTrueUserCode(code)
	invokeDeploy := fmt.Sprintf(`
try:
    tas_%s = %s()
except Exception:
    raise ABICheckException("ABI input contract name error,input contract name is %s")
`, contractName, contractName, contractName)

	allContractCode := fmt.Sprintf(`%s%s%s`, trueCode, PycodeContractDeployHooks(contractName), invokeDeploy)
	return allContractCode, libLine

}

func PycodeLoadMsg(sender string, value uint64, contractAddr string) string {
	return fmt.Sprintf(`
import ujson
import account
class TEvent(object):
    dict = {}
    def __init__(self):
        pass

TEvents = TEvent()

class DefEvent(object):
    class Node(object):
        def __init__(self,name):
            self.name = name
            
        def __call__(self, index,data):
            if type(index) != type('a'):
                raise LibException('index should be string',2)
            if type(data) != type({'val':1}):
                raise LibException('data should be dict',2)
            account.eventCall(self.name,index,ujson.dumps(data))
            #print("name :", self.name)
            #print("index:",index)
            #print("data :",ujson.dumps(data))

    def __init__(self,name):
        #print(name)
        #def ev_fun(self,index,data):
        #    print(index)
        #    print(data)
        setattr(TEvent,name,DefEvent.Node(name))


class Msg(object):
    def __init__(self, data, value, sender):
        self.data = data
        self.value = value
        self.sender = sender

    def __repr__(self):
        return "data: " + str(self.data) + " value: " + str(self.value) + " sender: " + str(self.sender)

class Register(object):
    def __init__(self):
        self.funcinfo = {}

    def public(self , *dargs):
        def wrapper(func):
            paranametuple = func.__para__
            paraname = list(paranametuple)
            paraname.remove("self")
            paratype = []
            for i in range(len(paraname)):
                paratype.append(dargs[i])
            self.funcinfo[func.__name__] = [paraname,paratype]
            
            def _wrapper(*args , **kargs):
                return func(*args, **kargs)
            return _wrapper
        return wrapper

import builtins
builtins.register = Register()
builtins.msg = Msg(data=bytes(), sender="%s", value=%d)
builtins.this = "%s"`, sender, value, contractAddr)
}

func GetInterfaceType(value interface{}) string {
	switch value.(type) {
	case float64:
		return "1"
	case bool:
		return "True"
	case string:
		return "\"str\""
	case []interface{}:
		return "[list]"
	case map[string]interface{}:
		return "{\"dict\":\"test\"}"
	default:
		fmt.Println(value)
		return "unknow"
		//panic("")
	}
	return ""
}

func PycodeCheckAbi(abi ABI) string {

	var str string
	str = `
__ABIParaTypes=[]`
	for i := 0; i < len(abi.Args); i++ {
		str += fmt.Sprintf("\n"+"__ABIParaTypes.append(type(%s))", GetInterfaceType(abi.Args[i]))
	}

	str += fmt.Sprintf(`
if "%s" in register.funcinfo:
    if len(register.funcinfo["%s"][1]) == len(__ABIParaTypes):
        for i in range(len(__ABIParaTypes)):
            #print(__ABIParaTypes[i])
            #print(register.funcinfo["%s"][1][i])
            if __ABIParaTypes[i] != register.funcinfo["%s"][1][i]:
                raise NoLineLibException('function %s para wrong')
    else:
        raise NoLineLibException("function %s para count wrong!")
else:
    raise NoLineLibException("cannot call this function: %s")
`, abi.FuncName, abi.FuncName, abi.FuncName, abi.FuncName, abi.FuncName, abi.FuncName, abi.FuncName)

	return str
}
func TasJson() string {
	code := `
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
    supportType = [TypeInt, TypeBool, TypeStr, TypeNone]

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
            raise LibException("map can not be more than nested 5",3)
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
            raise LibException("value must be int,bool,string. type is " + str(valueType),5)

    @staticmethod
    def checkKey(key):
        if type(key) != TasJson.TypeStr:
            raise LibException("key must be string",3)
        x = bytes(key, "utf-8")
        #if len(x) > 66:
        #    raise LibException("the length of key cannot more than 66!",3)

    @staticmethod
    def checkMapKey(key):
        if type(key) != TasJson.TypeStr:
            raise LibException("key must be string",3)
        x = bytes(key, "utf-8")
        #if len(x) > 66:
        #    raise LibException("the length of key cannot more than 66!",3)
`
	return code
}

func TasBaseStorageCode() string {
	code := `
import account
class TasBaseStorage:
    readData = {} #only get,not flush to db
    writeData={}  #write to db
    tasJson=TasJson()
    currentViterKey=""
    TypeTasMap=type(TasCollectionStorage())
    tasMapFieldList = {}

    def initHook(self):
        pass

    @staticmethod
    def checkValueCanDel(value):
        if type(value) == TasBaseStorage.TypeTasMap:
            raise LibException("can not remove a map!",5)

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
            raise LibException("can not remove a map!",4)
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
            raise LibException("can not remove a map!",4)
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
                    mapInstance = TasCollectionStorage()
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

`
	return code
}

func TasCollectionStorageCode() string {
	code := `
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
`
	return code
}
func calCodeLines(code string) int {
	return strings.Count(code, "\n") + 1
}
