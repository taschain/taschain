#!/bin/bash
instance_index=1
instance_count=$1
instance_end=$instance_index+$instance_count


for((;instance_index<instance_end;instance_index++))

do
	if [ ! -d 'logs' ]; then
		mkdir logs
	fi

	if [ ! -d 'pid' ]; then
		mkdir pid
	fi

	rpc_port=$[8100+$instance_index]
	pprof_port=$[9000+$instance_index]
	config_file='tas'$instance_index'.ini'
	stdout_log='logs/nohup_out_'$instance_index'.log'
	pid_file='pid/pid_tas'$instance_index'.txt'

	if [ -e $pid_file ];then
		kill -9 `cat $pid_file`
	fi

	if [ $instance_index -eq 1 ];then
		nohup ./gtas miner --config $config_file --rpc --rpcport $rpc_port --super --instance $instance_index --pprof $pprof_port --test  --seed 120.77.41.14 > $stdout_log 2>&1 & echo $! > $pid_file
	else
		nohup ./gtas miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port --test  --seed 120.77.41.14 > $stdout_log 2>&1 & echo $! > $pid_file
	fi
	sleep 1
done