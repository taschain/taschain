import block
import account
from lib.base.utils_tas import *
from clib.tas_runtime import glovar


class miner(object):

    def __init__(self):
        self.register_list = {}
        self.deregister_list = {}

    def register(self, public_key, miner_type, vrf_public_key=""):
        """
        申请成为矿工，转账金额做为质押金
        :param public_key: 公钥
        :param miner_type: 0=轻节点 1=重节点
        :param vrf_public_key: 重节点的vrf公钥，轻节点不需要
        :return:
        """
        # 账户余额检查
        # require(account.get_balance(msg.sender) >= msg.value)

        # 质押金检查
        miner_light_stake = 10
        miner_heavy_stake = 50
        if miner_type == 0:
            require(glovar.msg.value >= miner_light_stake)
        elif miner_type == 1:
            require(glovar.msg.value >= miner_heavy_stake)
        else:
            assert False

        #
        info = {"registerBlockNumber": block.number(),
                "stake": glovar.msg.value,
                "type": miner_type,
                "vrfPk": vrf_public_key,
                "owner": glovar.msg.sender}
        self.register_list[public_key] = info

        # 转账
        # account.sub_balance(glovar.msg.sender, glovar.msg.value)
        # account.add_balance(glovar.this, glovar.msg.value)

    def deregister(self, public_key):
        """
        注销矿工资格
        :param public_key:
        :return:
        """
        if public_key in self.register_list:
            info = self.register_list.pop(public_key)
            info["deregisterBlockNumber"] = block.number()
            infos = []
            if public_key in self.deregister_list:
                infos = self.deregister_list[public_key]
            infos.append(info)
            self.deregister_list[public_key] = infos

    def withdraw(self, public_key):
        """
        提取质押金
        :param public_key:
        :return:
        """
        # 是否有质押金冻结记录
        require(public_key in self.deregister_list)

        lock_stake_block_count = 5
        return_stake_infos = []

        infos = self.deregister_list[public_key]
        for info in infos:
            if (block.number() - info["deregisterBlockNumber"] > lock_stake_block_count and
                    info["owner"] == glovar.msg.sender):
                return_stake_infos.append(info)

        # 是否有已解冻质押金未提取，并且检查owner
        require(len(return_stake_infos) > 0)

        for info in return_stake_infos:
            self.deregister_list[public_key].remove(info)

        return_stake = 0
        for info in return_stake_infos:
            return_stake += info["stake"]

        # 转账
        assert return_stake > 0
        print(glovar.this)
        print(account.get_balance(glovar.this))
        assert account.get_balance(glovar.this) > return_stake
        account.transfer(glovar.msg.sender, return_stake)

    def test_print(self):
        print(self.register_list)
        print(self.deregister_list)

    def deploy(self):
        pass
