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

import builtins
builtins.msg = Msg(data=bytes(), sender="%s", value=%d)
builtins.this = "%s"`, sender, value, contractAddr)
}






