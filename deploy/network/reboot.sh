#!/bin/bash

stdout_log='logs/nohup_out_'$instance_index'.log'
nohup ./gtas miner --config $config_file --rpc --rpcport $rpc_port  --instance $instance_index --pprof $pprof_port --nat $nat_server --apply $apply --keystore keystore$instance_index > $stdout_log 2>&1 & echo $! > √