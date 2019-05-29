


class Test(object):

    def __init__(self):
        self.count = 0

    def receive_approval(self, _from, _value, _token, _extraData):
        self._count()
        print("Test: ", self.count)
        print("storage: ", storage)
        print("msg: ", msg)
        print("owner: ", owner)
        print("this: ", this)

    def _count(self):
        self.count += 1

