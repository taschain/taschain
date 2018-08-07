#!/bin/bash

for file in ./pid/pid_tas*
do
    kill -9 `cat $file`
    rm -f $file
done
