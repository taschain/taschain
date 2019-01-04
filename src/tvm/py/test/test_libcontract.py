
print("test_libcontract start:")
dir()

import lib_helloworld

class Libcontract(object):

    def __init__(self):
        pass

    def test(self):
        lib_helloworld.helloworld()
        lib_helloworld.Test().test_prt()

# if __name__ == '__main__':
#     Libcontract().test()