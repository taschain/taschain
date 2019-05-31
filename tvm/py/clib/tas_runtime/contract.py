import sys
import traceback
# from inspect import currentframe, getframeinfo

from clib.tas_runtime import glovar

class InterpreterError(Exception): pass

class Contract(object):

    def __init__(self, addr, env, depends):
        self.data = glovar.storage.get(addr)
        self.addr = addr
        self.env = env
        self.env["tas_filename"] = addr
        self.import_depends(depends)
        # TODO 删除加载的module
        self.my_exec(cmd=self.data.get_code(), globals=self.env)
        self.my_exec(cmd="tas_{addr} ={class_name}()".format(addr=addr, class_name=self.data.get_class()), globals=self.env)
        self.data.load_data(self.env.get("tas_{addr}".format(addr=addr)))

    def import_depends(self, depends):
        for code_addr in depends:
            code = glovar.storage.get(code_addr).get_code()
            class_name = glovar.storage.get(code_addr).get_class()
            glovar.importer.add_module(class_name, code)

    def call(self, function_name, *args, **kwargs):
        self.env["tas_{addr}_args".format(addr=self.addr)] = args
        self.env["tas_{addr}_kwargs".format(addr=self.addr)] = kwargs
        self.env["tas_{addr}_function_name".format(addr=self.addr)] = function_name
        self.my_exec(cmd="tas_{addr}_func = getattr(tas_{addr}, tas_{addr}_function_name, None)".format(addr=self.addr), globals=self.env)
        self.my_exec(cmd="tas_{addr}_func(*tas_{addr}_args, **tas_{addr}_kwargs)".format(addr=self.addr), globals=self.env)
        # print('self.env.get("tas_{addr}"')
        # print(self.env.get("tas_{addr}".format(addr=self.addr)))
        self.data.dump_data(self.env.get("tas_{addr}".format(addr=self.addr)))


    # def tas_exec(self, script_code, env):
    #     frameinfo = getframeinfo(currentframe())
    #     try:
    #         exec(script_code, env)
    #     except Exception as e:
    #         print(script_code)
    #         print(repr(e))
    #         print(e.lineno)
    #         print("Fails at:" + str(sys.exc_info()[2].tb_next.tb_lineno + frameinfo.lineno))
    #         print("Contract :" + self.env["tas_filename"])
    #         raise Exception("error!")
    #         #print "Inside:", len(func_str.split("\n")) - frameinfo.lineno

    def my_exec(self, cmd, globals=None, locals=None, description='source string'):
        try:
            exec(cmd, globals, locals)
        except SyntaxError as err:
            error_class = err.__class__.__name__
            detail = err.args[0]
            line_number = err.lineno
        except Exception as err:
            error_class = err.__class__.__name__
            detail = err.args[0]
            cl, exc, tb = sys.exc_info()
            line_number = traceback.extract_tb(tb)[-1][1]
        else:
            return
        #raise InterpreterError("%s at line %d of %s: %s" % (error_class, line_number, description, detail))
        raise InterpreterError("%s at line %d of %s: %s" % (error_class, line_number, self.env["tas_filename"], detail))