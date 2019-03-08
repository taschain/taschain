#!/bin/bash

#instance_index1:is_heavy,instance_index2:is_heavy...
params=$1
nat_server='120.78.127.246'
nat_port=3000

if [ ! -d 'logs' ]; then
    mkdir logs
fi

if [ ! -d 'pid' ]; then
    mkdir pid
fi

arr=(${params//,/ })
for inst in ${arr[@]}
do
    cfg=(${inst//:/ })
    instance_index=${cfg[0]}
    apply_type=${cfg[1]}
    apply='light'
    rpc_port=$[8100+$instance_index]
    if [ $apply_type = 1 ];then
        apply='heavy'
        rpc_port=8101
    fi

    pprof_port=$[9000+$instance_index]
    config_file='tas'$instance_index'.ini'
    stdout_log='logs/nohup_out_'$instance_index'.log'
    pid_file='pid/pid_tas'$instance_index'.txt'
    if [ -e $pid_file ];then
        kill -9 `cat $pid_file`
    fi

    if [ $instance_index -eq 1 ];then
        nohup ./gtas miner --config $config_file --monitor --rpc --rpcport $rpc_port --super --instance $instance_index --pprof $pprof_port --nat $nat_server --natport $nat_port --apply $apply --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > $pid_file
    else
        nohup ./gtas miner --config $config_file --monitor --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port --nat $nat_server --natport $nat_port --apply $apply --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > $pid_file
    fi
    sleep 0.1
done
