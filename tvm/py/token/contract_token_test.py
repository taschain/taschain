
from contract_token import *
import contract_token


def main():
    Msg.sender = Address("0xabcdefghijk")
    Msg.value = 0

    # 部署合约
    myAdvancedToken = MyAdvancedToken()
    myAdvancedToken.deploy()
    Storage.save(myAdvancedToken)

    # 执行合约
    # [1] 初始化测试环境
    #global this
    contract_token.this = Address("0x123456789")
    #global owner
    contract_token.owner = Address("0xabcdefghijk")
    # [2] 执行合约

    # Test 1 设置价格
    myAdvancedToken = MyAdvancedToken()
    Storage.load(myAdvancedToken)
    myAdvancedToken.set_prices(1, 1)
    Storage.save(myAdvancedToken)
    # Test 2 转账
    myAdvancedToken = MyAdvancedToken()
    Storage.load(myAdvancedToken)
    myAdvancedToken.transfer(Address("0xbcbcbcbcbc"), 50)
    Storage.save(myAdvancedToken)
    # Test 3 减发
    myAdvancedToken = MyAdvancedToken()
    Storage.load(myAdvancedToken)
    myAdvancedToken.burn(100000)
    Storage.save(myAdvancedToken)
    # Test 4 增发
    myAdvancedToken = MyAdvancedToken()
    Storage.load(myAdvancedToken)
    myAdvancedToken.mint_token(contract_token.this, 200000)
    Storage.save(myAdvancedToken)


if __name__ == '__main__':
    main()
