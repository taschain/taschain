class ContractStorageException():
    def __init__(self):
        pass

    @register.public()
    def callExcption1(self):
        try:
            self.s = TasCollectionStorage()
            self.s["1"]= TasCollectionStorage()
            self.s["1"]["1"] = TasCollectionStorage()
            self.s["1"]["1"]["1"] = TasCollectionStorage()
            self.s["1"]["1"]["1"]["1"] = TasCollectionStorage()
            self.s["1"]["1"]["1"]["1"]["1"] = TasCollectionStorage()
        except Exception:
            self.callExcption1 = "success"

    @register.public()
    def callExcption2(self):
        try:
            self.ssssssssssssssssssssssssssssssssssssssssssss = "sas"
        except Exception as e:
            self.callExcption2 = "success"

    @register.public()
    def callExcption3(self):
        try:
            self.sas = {"1":22122}
        except LibException as e:
            self.callExcption3 = "success"
