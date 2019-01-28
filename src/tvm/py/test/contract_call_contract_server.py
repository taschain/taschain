

class Server():
    def __init__(self):
        print("deploy Server")

    @register.public()
    def hello(self):
        print("hello")
        return "hello"

    @register.public(int, int)
    def plus(self, a, b):
        return a + b
