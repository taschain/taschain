from clib.tas_runtime import address


class TasDeploy(object):

    def __init__(self):
        self.from_address = address("")
        self.contract_name = ""
        self.main_file = ""
        self.depends_file = ""
        self.lib_contract = False
        main_code_hex = ""
        with open(self.main_file, "r", encoding="UTF-8") as f:
            main_code_hex = f.read().encode("UTF-8").hex()

        depends_code_hex = {}
        for file_path in self.depends_file:
            with open(file_path, "r", encoding="UTF-8") as f:
                depends_code_hex[f.name] = f.read().encode("UTF-8").hex()

        # return address


if __name__ == '__main__':
    tasDeploy = TasDeploy()
    tasDeploy.contract_name = "MyAdvancedToken"
    tasDeploy.main_file = "token/contract_token.py"
    tasDeploy.deploy()

















