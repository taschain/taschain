


class Test(object):

    def __init__(self):
        self.count = 0

    def receive_approval(self, _from, _value, _token, _extraData):
        self._count()
        print("Test: ", self.count)

    def _count(self):
        self.count += 1


