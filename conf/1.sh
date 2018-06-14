#!/bin/bash

nohup ./gtas miner --config ./tas1.ini --rpc --rpcport 8101 --super --pprof 9001 >> ../logs/stdout1.log 2>&1 &
