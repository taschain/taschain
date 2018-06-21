#!/bin/bash

for file in ./pid_tas*
do
    kill -9 `cat $file`
    rm -f $file
done
