


class Recharge(object):

    def receive_approval(self, _from, _value, _token, _extraData):
        # 获取需要充值的电话号码
        phone_number = int(_extraData)
        # 收取代币

        # 充值
        print("充值成功: ", phone_number, _value)

    def test(self):
        print("充值成功: test")

    def deploy(self):
        pass

