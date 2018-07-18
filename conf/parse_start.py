# -*- coding: utf-8 -*-
import json
import time
import sys
import re

rpcport = 8100
pprofport = 9000

def load(jf):
    with open(jf) as json_file:
        data = json.load(json_file)
        return data

def write(fn, content):
    f = open(fn, "w")
    f.write(content)
    f.flush()
    f.close()

def command(conf, isSuper):
    seq = int(conf.split(".")[0][3:])
    s = ""
    if isSuper:
        s = "--super"
    content = "if [ -e pid_tas%d.txt ];then\n\tkill -9 `cat pid_tas%d.txt`\nfi" % (seq, seq)
    content = "%s\nnohup ./gtas miner --config %s --rpc --rpcport %d %s --pprof %d > logs/stdout_%s.log 2>&1 & echo $! > pid_tas%d.txt" % (content, conf, rpcport+seq, s, pprofport+seq, seq, seq)

    slp = 1
    if isSuper:
        slp = 5

    content = "%s\nsleep %d" % (content, slp)

    return (content, rpcport+seq)

def generateFiles(data):
    hostports=[]
    for c in data:
        include_list = "gtas\n" + "stop.sh\n"
        content = "#/bin/bash\n"
        content += "if [ ! -d 'logs' ]; then\n\tmkdir logs\nfi\n"
        host = c["host"]
        for inst in c["instants"]:
            include_list += inst["config"] + "\n"
            (ct, p) = command(inst["config"], inst["super"])
            content += ct + "\n\n"
            hostports.append("%s:%d" %(host, p))
        fn = startShName(host)
        write(fn, content)
        include_list += fn + "\n"
        include_list += "p2p_core.so\n"
        write("include_list_" + host, include_list)
    write("host_port", ",".join(hostports))

def getHosts(data):
    arr = []
    for c in data:
        arr.append(c["host"])
    print ",".join(arr)

def startShName(host):
    return "start_%s.sh" % host.replace(".", "_")

def checkip(ip):
    p = re.compile('^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$')
    if p.match(ip):
        return True
    else:
        return False

# def getCmdByHost(data, host):
#     cmds = []
#     for c in data:
#         if c["host"] == host:
#             for inst in c["instants"]:
#                 cmds.append(command(inst["config"], inst["super"]))
#     print ",".join(cmds)



if __name__ == "__main__":
    p = sys.argv[1]

    if checkip(p):
        print startShName(p)
    else:
        data = load(sys.argv[1])
        generateFiles(data)
        getHosts(data)



    # if t == 0:
    #     getHosts(data)
    # elif t == 1:
    #     getCmdByHost(data, sys.argv[3])
    # else:
    #     parse(data)
