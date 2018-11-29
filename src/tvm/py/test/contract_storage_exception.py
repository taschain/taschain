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
            #self.s["1"]["1"]["1"]["1"]["1"] = TasCollectionStorage()
        except Exception as e:
            print(e.msg)
