import account


class Tns():

    def __init__(self):
        self.account_owner = TasCollectionStorage()
        self.account_address = TasCollectionStorage()
        self.address_account = TasCollectionStorage()
        self.admin = msg.sender

    # 检查权限
    def check_owner(self, account):
        # if account in self.account_owner:
        if self.account_owner[account] is None:
            raise Exception("account暂未注册")

        if self.account_owner[account] != msg.sender:
            raise Exception("没有owner权限")

    # 检查命名规则
    def check_account(self, account):
        if len(account) == 10:
            for ch in account:
                if not ("a" <= ch <= "z" or ord("0") <= ord(ch) <= ord("9")):
                    raise Exception("account只能使用a~z 0~9")
        else:
            raise Exception("account长度必须等于10")

    # 检查admin权限
    def check_admin(self):
        if self.admin != msg.sender:
            raise Exception("没有admin权限")

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
        self.address_account[address] = account

    # 账户名权限转让
    @register.public(str, str)
    def set_account_owner(self, account, new_owner_address):
        self.check_owner(account)
        self.account_owner[account] = new_owner_address

    # 注册并设置账户名对应的地址
    @register.public(str, str)
    def register_and_set_account_address(self, account, address):
        self.register_account(account)
        self.set_account_address(account, address)

    # 代理注册账户并设置账户名对应的地址
    @register.public(str, str)
    def proxy_register_and_set_account_address(self, account, address):
        self.register_account(account)
        self.set_account_address(account, address)
        self.set_account_owner(account, address)

    # 获取account绑定的地址
    @register.public(str)
    def get_address(self, account):
        # if account in self.account_address:
        if self.account_address[account] is not None:
            return self.account_address[account]
        else:
            Exception("account暂未注册")

    # 获取地址对应的account
    @register.public(str)
    def get_account(self, address):
        if self.address_account[address] is not None:
            return self.address_account[address]
        else:
            Exception("account暂未注册")

    # 设置短账户地址
    @register.public(str, str)
    def set_short_account_address(self, account, address):
        for ch in account:
            if not ("a" <= ch <= "z" or
                    ord("0") <= ord(ch) <= ord("9") or
                    ch == "."):
                raise Exception("account只能使用a~z 0~9 \".\"")

        if self.account_owner[account] is None:
            if self.admin == msg.sender:
                self.account_owner[account] = msg.sender
            else:
                raise Exception("没有admin权限")
        else:
            if self.account_owner[account] != msg.sender:
                raise Exception("没有owner权限")

        self.account_address[account] = address
        self.address_account[address] = account

    # 转让admin权限
    @register.public(str)
    def set_admin(self, new_admin_address):
        self.check_admin()
        self.admin = new_admin_address


