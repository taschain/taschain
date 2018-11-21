import account
import block

class Fomo3D(object):
    def __init__(self):
        self.key_price = 1  #
        self.round = 0  # 当前轮数
        self.total_key_count = 0  # 卖出的所有key的数量
        self.current_round_key_count = 0
        self.owner = msg.sender
        self.balance = TasCollectionStorage()  # 记录各用户的余额
        self.round_list = TasCollectionStorage()  # 记录各round的购买的key的情况
        self.round_list[str(self.round)] = TasCollectionStorage()
        self.round_list_size = TasCollectionStorage()
        self.round_list_key = TasCollectionStorage()
        self.round_list_key[str(self.round)] = TasCollectionStorage()
        self.jackpot = 0  # 奖池金额
        self.previous_jackpot = 0
        self.last_one = ""  # 最后购买key的地址
        self.last_ranks = 0  # 最后购买者所选队伍
        self.airdrop_jackpot = 0  # 小奖池
        self.multiple = 10000  # 放大倍数
        self.contract_balance = 0
        self.round_time = 3 * 60
        self.time_plus = 5
        self.endtime = block.timestamp() + self.round_time
        self.max_jackpot = 10
        self.history = TasCollectionStorage()
        self.history[str(self.round)] = TasCollectionStorage()

    def _add_balance(self, account, amount):
        if self.balance[account] is None:
            self.balance[account] = 0
        if self.history[str(self.round)][account] is None:
            self.history[str(self.round)][account] = 0
        self.balance[account] += amount
        self.history[str(self.round)][account] += amount

    def _distribution(self, amount, jackpot, to_player, contract_balance, airdrop_jackpot):
        self.jackpot += amount * jackpot / 100
        give_to_player = amount * to_player / 100
        currentRoundAccountSize = self.round_list_size[str(self.round)]
        if currentRoundAccountSize is not None:
            for index in range(currentRoundAccountSize):
                account = self.round_list_key[str(self.round)][str(index)]
                self._add_balance(account,give_to_player * self.round_list[str(self.round)][account] / self.current_round_key_count)
        else:
            self.contract_balance += give_to_player

        self.contract_balance += amount * contract_balance / 100
        self.airdrop_jackpot += amount * airdrop_jackpot / 100


    def _snake_distribution(self, amount):
        self._distribution(amount, 30, 38, 2, 30)

    def _cow_distribution(self, amount):
        self._distribution(amount, 43, 25, 2, 30)

    def _whale_distribution(self, amount):
        self._distribution(amount, 56, 12, 2, 30)

    def _bear_distribution(self, amount):
        self._distribution(amount, 43, 25, 2, 30)

    def _airdrop(self):
        if self.airdrop_jackpot > self.max_jackpot * self.multiple:
            lucky_number = block.timestamp()%self.current_round_key_count

            currentRoundAccountSize = self.round_list_size[str(self.round)]
            if currentRoundAccountSize is not None:
                for index in range(currentRoundAccountSize):
                    account = self.round_list_key[str(self.round)][str(index)]
                    value = self.round_list[self.round][account]
                    lucky_number -= value
                    if lucky_number <= 0:
                        self._add_balance(k, self.airdrop_jackpot)
                        self.airdrop_jackpot = 0
                        return


    def _distribute(self, amount, _type):
        if _type == 0:
            self._snake_distribution(amount)
        elif _type == 2:
            self._cow_distribution(amount)
        elif _type == 1:
            self._whale_distribution(amount)
        elif _type == 3:
            self._bear_distribution(amount)
        else:
            raise Exception("invalid ranks type")

    def _who_win(self, last_one_earn, to_player, to_contract, to_jackpot):
        if self.last_one is None:
            pass
        else:
            self._add_balance(self.last_one, self.jackpot * last_one_earn / 100)
            give_to_player = (self.jackpot + self.previous_jackpot) * to_player / 100

            currentRoundAccountSize = self.round_list_size[str(self.round)]
            if currentRoundAccountSize is not None:
                for index in range(currentRoundAccountSize):
                    account = self.round_list_key[str(self.round)][str(index)]
                    self._add_balance(account, give_to_player * self.round_list[str(self.round)][account] / self.current_round_key_count)

        self.contract_balance += self.jackpot * to_contract / 100
        self.round += 1
        self.round_list[str(self.round)] = TasCollectionStorage()
        self.round_list_key[str(self.round)] = TasCollectionStorage()
        self.history[str(self.round)] = TasCollectionStorage()
        self.previous_jackpot = self.jackpot * to_jackpot / 100
        self.jackpot = 0
        self.endtime = block.timestamp() + self.round_time
        self.last_one = None
        self.last_ranks = -1
        self.current_round_key_count = 0


    def _snake_win(self):
        self._who_win(48, 40, 2, 10)

    def _cow_win(self):
        self._who_win(48, 40, 2, 10)

    def _whale_win(self):
        self._who_win(48, 25, 2, 25)

    def _bear_win(self):
        self._who_win(48, 25, 2, 25)

    def _check_end(self):
        if block.timestamp() > self.endtime:
            if self.last_ranks == 0:
                self._snake_win()
            elif self.last_ranks == 2:
                self._cow_win()
            elif self.last_ranks == 1:
                self._whale_win()
            elif self.last_ranks == 3:
                self._bear_win()
            else:
                raise Exception("invalid ranks type")

    def _endtime_rise(self, key_count):
        if self.endtime + self.time_plus * key_count - block.timestamp() > self.round_time:
            self.endtime = block.timestamp() + self.round_time
        else:
            self.endtime = self.endtime + key_count * self.time_plus

    def _update_history(self):
        # for addr, count in self.ba
        pass

    # 购买key
    def _buy_key(self, key_count, ranks_type):
        # 初始化账户
        if self.balance[msg.sender] is None:
            self.balance[msg.sender] = 0
        amount = self.multiple * key_count
        self._check_end()
        self._distribute(amount, ranks_type)
        self._endtime_rise(key_count)
        has_key = self.round_list[str(self.round)][msg.sender]
        if has_key is None:
            self.round_list[str(self.round)][msg.sender] = key_count
            currentRoundAccountSize = self.round_list_size[str(self.round)]
            if currentRoundAccountSize is not None:
                self.round_list_key[str(self.round)][str(currentRoundAccountSize)] = msg.sender
                self.round_list_size[str(self.round)] = currentRoundAccountSize+1
            else:
                self.round_list_key[str(self.round)]["0"] = msg.sender
                self.round_list_size[str(self.round)] = 1
        else:
            self.round_list[str(self.round)][msg.sender] += key_count
        self.current_round_key_count += key_count
        self.last_one = msg.sender
        self.last_ranks = ranks_type

    # payment_type=0: tas购买； payment_type=1： 余额购买
    @register.public(int,int,int)
    def buy(self, key_count, ranks_type, payment_type):
        if payment_type == 0:
            assert key_count * self.key_price <= msg.value
            for i in range(key_count):
                self._buy_key(1, ranks_type)
        elif payment_type == 1:
            value = self.balance[msg.sender]
            if value is None:
                value = 0
            assert key_count * self.key_price * self.multiple <= value
            self.balance[msg.sender] -= key_count * self.key_price * self.multiple
            for i in range(key_count):
                self._buy_key(1, ranks_type)
        else:
            raise Exception("invalid payment_type")
        self._airdrop()

    @register.public()
    def withdraw(self):
        value = self.balance[msg.sender] / self.multiple
        if value == 0:
            return
        self.balance[msg.sender] -= value * self.multiple
        account.transfer(msg.sender, value)

    @register.public()
    def owner_withdraw(self):
        assert msg.sender == self.owner
        value = self.contract_balance / self.multiple
        if value == 0:
            return
        self.contract_balance -= value * self.multiple
        account.transfer(msg.sender, value)
