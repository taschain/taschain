#!/bin/bash
instance_index=$1
instance_count=$2
instance_end=$instance_index+$instance_count
nat_server=$3
build_number=$4

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
	stdout_log='logs/stdout_'$instance_index'.log'
	pid_file='pid/pid_tas'$instance_index'.txt'

	if [ -e $pid_file ];then
		kill -9 `cat $pid_file`
	fi

	#echo -e 'nohup ./gtas miner --config' $config_file '--rpc --rpcport' $rpc_port '--super --instance' $instance_index '--pprof' $pprof_port '>' $stdout_log '2>&1 & echo $! >' $pid_file

	if [ $instance_index -eq 1 ];then
		nohup ./gtas miner --config $config_file --rpc --rpcport $rpc_port --super --instance $instance_index --prefix aly_flow --nat $nat_server --build_id $build_number --pprof $pprof_port > $stdout_log 2>&1 & echo $! > $pid_file
	else
		nohup ./gtas miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --prefix aly_flow --nat $nat_server --build_id $build_number --pprof $pprof_port > $stdout_log 2>&1 & echo $! > $pid_file
	fi
	sleep 1
done
