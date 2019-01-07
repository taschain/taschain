class ContractSyntax():
    def __init__(self):
        pass


    @register.public()
    def callExcption1(self):
        try:
            self.s = ddd
        except Exception:
            self.data = "success"


