#!/bin/bash

instance_info=`cat instance_info`
./tx_sender -l $instance_info -t 1000000 -i $1