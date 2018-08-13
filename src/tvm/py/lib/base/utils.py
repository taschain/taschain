

import lib.base as base


def require(b):
    if not b:
        raise Exception("")


#调用者是否为合约创建者
def check_owner():
    if base.owner == base.msg.sender:
        return True
    else:
        raise Exception("只有合约owner可以操作")