import lib




class Contract(object):

    def __init__(self, addr, env):
        self.data = lib.storage.get(addr)
        self.addr = addr
        self.env = env
        exec(self.data.get_code(), self.env)
        exec("tas_{addr} ={class_name}()".format(addr=addr, class_name=self.data.get_class()), self.env)
        self.data.load_data(self.env.get("tas_{addr}".format(addr=addr)))

    def call(self, function_name, *args, **kwargs):
        self.env["tas_{addr}_args".format(addr=self.addr)] = args
        self.env["tas_{addr}_kwargs".format(addr=self.addr)] = kwargs
        self.env["tas_{addr}_function_name".format(addr=self.addr)] = function_name
        exec("tas_{addr}_func = getattr(tas_{addr}, tas_{addr}_function_name, None)".format(addr=self.addr), self.env)
        exec("tas_{addr}_func(*tas_{addr}_args, **tas_{addr}_kwargs)".format(addr=self.addr), self.env)
        self.data.dump_data(self.env.get("tas_{addr}".format(addr=self.addr)))


