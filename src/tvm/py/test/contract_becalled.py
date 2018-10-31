class ContractBeCalled():
    def __init__(self):
        pass

    def deploy(self):
        print("deploy")


    def ready(self):
        print("ContractBeCalled is ready.")

    def count(self):
        print("ContractBeCalled is called")
        return 100