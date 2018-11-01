package tvm

import "fmt"

func PycodeStoreContractData(contractName string) string {
	return fmt.Sprintf(`
import account
import ujson
for k in tas_%s.__dict__:
    #print(k)
    #print(type(k))
    #print(tas_%s.__dict__[k])
    #print(type(tas_%s.__dict__[k]))
    value = ujson.dumps(tas_%s.__dict__[k])
    if TAS_PARAMS_DICT.get(k) != value:
        account.set_data(k, value)`, contractName, contractName, contractName, contractName)
}

func PycodeLoadContractData(contractName string) string {
	return fmt.Sprintf(`
import account
import ujson
TAS_PARAMS_DICT = {}
for k in tas_%s.__dict__:
    #	print(k)
    #	print(type(k))
    #	value = ujson.loads(account.get_data("", k))
    #	print(value)
    value = account.get_data(k)
    TAS_PARAMS_DICT[k] = value
    setattr(tas_%s, k, ujson.loads(value))`, contractName, contractName)
}

func PycodeCreateContractInstance(code string, contractName string) string {
	return fmt.Sprintf(`
%s
tas_%s = %s()`, code, contractName, contractName)
}

func PycodeContractDeploy(code string, contractName string) string {
	return fmt.Sprintf(`
%s
TAS_PARAMS_DICT = {}
tas_%s = %s()
tas_%s.deploy()`, code, contractName, contractName, contractName)
}

func PycodeLoadMsg(sender string, value uint64, contractAddr string) string {
	return fmt.Sprintf(`
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.address_tas import Address

class Register(object):
    def __init__(self):
        self.funcinfo = {}

    def public(self , *dargs):
        def wrapper(func):
            paranametuple = func.__para__
            paraname = list(paranametuple)
            paraname.remove("self")
            #print(paraname)
            #print(len(paraname))
            paratype = []
            for i in range(len(paraname)):
                #print(dargs[i])
                paratype.append(dargs[i])
            self.funcinfo[func.__name__] = [paraname,paratype]
            print(self.funcinfo)
            
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
	//case bool:
	//	return "bool"
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
	//return fmt.Sprintf(`if "%s" not in register.funcinfo:
	//raise Exception("cannot call this function: %s")`, abi.FuncName, abi.FuncName)

	var str string //:=`__ABIParaTypes = ["`
	//var types []string

//	if len(abi.Args) == 0 {
//		str = `
//__ABIParaTypes=[]`
//	}else {
//		str = `__ABIParaTypes = ["`
//		for i := 0; i < len(abi.Args); i++ {
//			tmp := GetInterfaceType(abi.Args[i])
//			types = append(types, tmp)
//			//fmt.Println(types[i])
//			str += types[i]
//			if i == len(abi.Args)-1 {
//				str += `"]
//`
//			} else {
//				str += `","`
//			}
//		}
//	}

	str = `
__ABIParaTypes=[]`
    for i := 0; i < len(abi.Args); i++ {
    	str += fmt.Sprintf("\n" + "__ABIParaTypes.append(type(%s))",GetInterfaceType(abi.Args[i]))
	}
	//str += fmt.Sprintf("\nprint(__ABIParaTypes)")
	//fmt.Println(str)

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

	fmt.Println(str)


	return str
}




