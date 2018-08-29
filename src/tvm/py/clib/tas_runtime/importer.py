import sys
import importlib.abc

# Debugging
import logging
# logging.basicConfig(level=logging.DEBUG)
log = logging.getLogger(__name__)


class UrlMetaFinder(importlib.abc.MetaPathFinder):
    def __init__(self):
        self.data = {}
        self.moduleLoader = {}

    def add_module(self, module_name, code):
        log.debug('add_module: module_name=%r, code=%r', module_name, code)
        self.data[module_name] = code
        self.moduleLoader[module_name] = UrlModuleLoader(module_name, code)

    def find_module(self, fullname, path=None):
        log.debug('find_module: fullname=%r, path=%r', fullname, path)

        if fullname in self.data.keys():
            log.debug('find_module: module %r found', fullname)
            return self.moduleLoader[fullname]
        else:
            log.debug('find_module: module %r not found', fullname)
            return None

    def invalidate_caches(self):
        self.data.clear()
        self.moduleLoader.clear()
        log.debug('invalidating link cache')


# Module Loader for a URL
class UrlModuleLoader(importlib.abc.SourceLoader):
    def __init__(self, module_name, code):
        self.module_name = module_name
        self.code = code

    def module_repr(self, module):
        return '<urlmodule %r from %r>' % (module.__name__, module.__file__)

    # Optional extensions
    def get_code(self, fullname):
        src = self.get_source(fullname)
        return compile(src, self.module_name, 'exec')

    def get_filename(self, fullname):
        return self.module_name

    def get_data(self, path):
        pass

    def get_source(self, fullname):
        log.debug('loader: reading %r', self.module_name)
        return self.code
        # try:
        #     with open(self.file_path, "r", encoding="utf-8") as f:
        #         source = f.read()
        #     log.debug('loader: %r loaded', self.module_name)
        #     return source
        # except FileExistsError as e:
        #     log.debug('loader: %r failed. %s', self.module_name, e)
        #     raise ImportError("Can't load %s" % self.module_name)

    def is_package(self, fullname):
        return False


def install_meta():
    finder = UrlMetaFinder()
    sys.meta_path.append(finder)
    log.debug('%r installed on sys.meta_path', finder)
    return finder


def remove_meta(finder):
    sys.meta_path.remove(finder)
    log.debug('%r removed from sys.meta_path', finder)


# meta_finder = install_meta()
# meta_finder.add_module("test_lib_helloworld", './testbb/test_lib_helloworld.py')
#
#
# import test_lib_helloworld
# test_lib_helloworld.Test().test_prt()