#!/bin/python3
import platform
import sys

file_list = ["bridge_wrap_windows.go", "bridge_wrap_linux.go", "bridge_wrap_darwin.go"]
path_list = ["windows", "linux", "darwin_amd64"]

def UsePlatform():
    sysstr = platform.system()
    if sysstr =="Windows":
        return "bridge_wrap_windows.go", "windows"
    elif sysstr == "Linux":
        return "bridge_wrap_linux.go", "linux"
    elif sysstr == "Darwin":
        return "bridge_wrap_darwin.go", "darwin_amd64"
    else:
        print ("Unknown System")


if __name__ == '__main__':
    file_name, path = UsePlatform()
    if file_name is None:
        exit(0)
    file_list.remove(file_name)
    path_list.remove(path)
    if sys.version_info<(3, 0):
        YorN = raw_input("{f0} -> {f1}, {f2} ? Y/N ".format(f0=file_name, f1 = file_list[0], f2=file_list[1]))
    else:
        YorN = input("{f0} -> {f1}, {f2} ? Y/N ".format(f0=file_name, f1 = file_list[0], f2=file_list[1]))
    if YorN == "Y" or YorN == "y":
        with open(file_name, "r",encoding='UTF-8') as f:
            content = f.read()
        for i in range(2):
            with open(file_list[i], "w") as f:
                f.write(content.replace(path, path_list[i], 1))
        print("exit")
    else:
        pass