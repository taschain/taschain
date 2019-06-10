import requests
import json



# 查询余额
def GTAS_balance():
    parm = {
        "method": "GTAS_balance",
        "params": ["6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"],
        "jsonrpc": "2.0",
        "id": "1"
    }
    r = requests.get('http://127.0.0.1:8101', json=parm)
    response = json.loads(r.content.decode("UTF-8"))
    print(response)

# 转账
def GTAS_tx(_from, _to):
    parm = {
        "method": "GTAS_tx",
        "params": [_from,
                   _to,
                   1,
                   "hello world".encode("UTF-8").hex()],
        "jsonrpc": "2.0",
        "id": "1"
    }
    r = requests.get('http://127.0.0.1:8101', json=parm)
    response = json.loads(r.content.decode("UTF-8"))
    print(response)

# 部署
def GTAS_deploy():

    contract_main = ""
    contract_depends = []

    with open("token/contract_token_test.py", "r", encoding="UTF-8") as f:
        contract_main = f.read().encode("UTF-8").hex()

    with open("token/contract_token.py", "r", encoding="UTF-8") as f:
        contract_depends.append(f.read().encode("UTF-8").hex())

    contract_name = "MyAdvancedToken"

    parm = {
        "method": "GTAS_tx",
        "params": ["6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b",
                   "",
                   0,
                   "hello world".encode("UTF-8").hex()],
        "jsonrpc": "2.0",
        "id": "1"
    }
    r = requests.get('http://127.0.0.1:8101', json=parm)
    response = json.loads(r.content.decode("UTF-8"))
    print(response)

# 部署2
def GTAS_deploy2():

    code = 'print("hello world!")'

    parm = {
        "method": "GTAS_tx",
        "params": ["6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b",
                   "",
                   0,
                   code.encode("UTF-8").hex()],
        "jsonrpc": "2.0",
        "id": "1"
    }
    r = requests.get('http://127.0.0.1:8101', json=parm)
    response = json.loads(r.content.decode("UTF-8"))
    print(response)


if __name__ == '__main__':

    GTAS_deploy2()

    # _from = "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
    # _to = "0xc68ac00a9e2e6bb4ced88a5689fab47152913d89"
    #
    # GTAS_t(_from, _to)


