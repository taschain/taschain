
import account

class Client():
    def __init__(self):
        print("deploy Client")

    @register.public()
    def client_print(self):
        print_str = account.contractCall('0x303e2d65fc7cec255932f6cbbfac69851d47a56d06e3f33dc63c9620e07b1872', 'hello', '[]')
        print_str = print_str + " world"
        print(print_str)
