package tvm

import "fmt"

func PycodeStoreContractData(contractName string) string {
	return fmt.Sprintf(`
TasBaseStorage.flushData()
`)
}

func PycodeCreateContractInstance(code string, contractName string) string {
	newCode:= fmt.Sprintf(`
%s
%s
tas_%s = %s()`, PycodeGetTrueUserCode(code), PycodeContractAddHooks(contractName),contractName, contractName)
	return newCode
}

func PycodeContractImports()string{
	return  "from lib.base.tas_storage_base_property import TasBaseStorage\nfrom lib.base.tas_storage_map_property import TasCollectionStorage"
}

func PycodeContractAddHooks(contractName string)string{
	initHook:=fmt.Sprintf("%s.__init__ = TasBaseStorage.initHook",contractName)
	setAttributeHook := fmt.Sprintf("%s.__setattr__= TasBaseStorage.setAttrHook",contractName)
	getAttributeHook := fmt.Sprintf("%s.__getattr__= TasBaseStorage.getAttrHook",contractName)
	return fmt.Sprintf(`
%s
%s
%s
	`,initHook,getAttributeHook,setAttributeHook)
}

func PycodeGetTrueUserCode(code string)string{
	usercode:=fmt.Sprintf(`
%s
%s
	`,PycodeContractImports(),code)
	return usercode
}


func PycodeContractDeploy(code string, contractName string) string {
	invokeDeploy:=fmt.Sprintf(`
tas_%s = %s()
tas_%s.deploy()
	`,contractName, contractName, contractName)

	allContractCode:= fmt.Sprintf(`
%s
%s
%s
`, PycodeGetTrueUserCode(code),PycodeContractAddHooks(contractName),invokeDeploy)
	return allContractCode

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
                raise Exception('index should be string')
            if type(data) != type({'val':1}):
                raise Exception('data should be dict')
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

func GetInterfaceType(value interface{}) string{
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
    	str += fmt.Sprintf("\n" + "__ABIParaTypes.append(type(%s))",GetInterfaceType(abi.Args[i]))
	}

	str += fmt.Sprintf(`
if "%s" in register.funcinfo:
    if len(register.funcinfo["%s"][1]) == len(__ABIParaTypes):
        for i in range(len(__ABIParaTypes)):
            #print(__ABIParaTypes[i])
            #print(register.funcinfo["%s"][1][i])
            if __ABIParaTypes[i] != register.funcinfo["%s"][1][i]:
                raise Exception('function %s para wrong')
    else:
        raise Exception("function %s para wrong!")
else:
    raise Exception("cannot call this function: %s")
`, abi.FuncName, abi.FuncName,abi.FuncName,abi.FuncName,abi.FuncName,abi.FuncName,abi.FuncName)


	return str
}




