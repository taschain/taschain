import account


class Tns():

    def __init__(self):
        self.account_owner = TasCollectionStorage()
        self.account_address = TasCollectionStorage()
        print("tns deploy")

    # 检查权限
    def check_owner(self, account):
        # if account in self.account_owner:
        if self.account_owner[account] is not None:
            if self.account_owner[account] == msg.sender:
                return True
            else:
                raise Exception("没有owner权限")
        else:
            raise Exception("account暂未注册")

    # 检查命名规则
    def check_account(self, account):
        if len(account) == 10:
            for ch in account:
                if "a" <= ch <= "z" or "A" <= ch <= "Z" or ord("0") <= ord(ch) <= ord("9"):
                    return True
                else:
                    raise Exception("account只能使用a~z A~Z 0~9")
        else:
            raise Exception("account长度必须等于10")

    # 注册账户名
    @register.public(str)
    def register_account(self, account):
        self.check_account(account)
        # if account not in self.account_owner:
        if self.account_owner[account] is None:
            self.account_owner[account] = msg.sender
        else:
            raise Exception("account已被注册")

    # 设置账户名对应的地址
    @register.public(str, str)
    def set_account_address(self, account, address):
        self.check_owner(account)
        self.account_address[account] = address

    # 账户名权限转让
    @register.public(str, str)
    def set_account_owner(self, account, new_owner):
        self.check_owner(account)
        self.account_owner[account] = new_owner

    # 获取account绑定的地址
    @register.public(str)
    def get_address(self, account):
        # if account in self.account_address:
        if self.account_address[account] is not None:
            return self.account_address[account]
        else:
            Exception("account暂未注册")
