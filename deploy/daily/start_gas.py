# -*- coding: UTF-8 -*-
import sys,os
import threading
import time

def start1(id):
    if (id == 1):    
        cmd='gtas.exe miner --config {config_file} --rpc --rpcport {rpc_port} --super --instance {id} --pprof {pprof_port}  --test --seed 127.0.0.1 --apply heavy --keystore keystore{id} > {stdout_log} 2>&1'.format( 
                config_file=config_file, rpc_port=rpc_port, id=id, pprof_port=pprof_port, stdout_log=stdout_log) 
        print(cmd) 
        os.system(cmd)
    else:
        cmd='gtas.exe miner --config {config_file} --rpc --rpcport {rpc_port} --instance {id} --pprof {pprof_port} --test --seed 127.0.0.1 --apply heavy --keystore keystore{id} > {stdout_log} 2>&1'.format( 
                config_file=config_file, rpc_port=rpc_port, id=id, pprof_port=pprof_port, stdout_log=stdout_log) 
        os.system(cmd) 

if __name__ == '__main__':
    if len(sys.argv) != 2:
        exit(0)

    if not os.path.exists("logs"):
        os.makedirs("logs")

    if not os.path.exists("pid"):
        os.makedirs("pid")    

    # os.mkdir( "logs", 0755 );
    # os.mkdir( "pid", 0755 );

    id_start = 1
    id_count = int(sys.argv[1])
    id_end = id_count + id_start
    ths=[]

    for id in range(id_start, id_end):
        # print "index:", id 
        rpc_port    = 8100 + id
        pprof_port  = 9000 + id
        id_str      = str(id)
        config_file = "tas" + id_str + ".ini"
        stdout_log  = "logs/stdout_" + id_str + ".log"
        pid_file    = "pid/pid_tas" + id_str + ".txt"
		
        # kill exist program
        # if os.path.exists(pid_file):

        th = threading.Thread(target=start1,args=(id,))
        th.setDaemon(True)
        th.start()
time.sleep(2000)	





