

class Server():
    def __init__(self):
        print("deploy Server")

    @register.public()
    def hello(self):
        print("hello")
        return "hello"
