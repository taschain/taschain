from clib.tas_runtime import glovar
import account
import block
class CrowdFunding():
    def __init__(self):
        self.funding_goal = 0       # 众筹目标
        self.funding = 0            # 已众筹金额
        self.max_block_number = 0   # 众筹最高区块
        self.vote_dict = {}         # 众筹情况数据存储
        self.on_sale = True         # 众筹状态
        self.owner = ""             # 众筹所有者

    def deploy(self):
        self.funding_goal = 10000
        self.max_block_number = block.number() + 1000
        self.owner = msg.sender

    def sale(self):
        if self.max_block_number < block.number():
            self.on_sale = False
        if not self.on_sale:
            raise Exception("not on sale")
        value = msg.value
        sender = msg.sender
        self.funding = self.funding + value
        if self.funding > self.funding_goal:
            value = self.funding_goal-self.funding
            self.on_sale = False
            account.transfer(sender, value)
            return
        balance = self.vote_dict.get(sender, 0)
        self.vote_dict[sender] = balance + value
        print(self.vote_dict)

    def withdraw(self, addr):
        if self.owner != msg.sender:
            return
        if not self.on_sale and self.funding >= self.funding_goal:
            account.transfer(addr, self.funding)

    def failed(self):
        if not self.on_sale and self.funding < self.funding_goal and self.funding >= account.get_balance(this):
            for k, v in self.vote_dict.items():
                account.transfer(k, v)