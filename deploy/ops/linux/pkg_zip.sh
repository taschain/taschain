#!/bin/bash

version=0.8.0

cd ../../daily
sh build.sh
cp -f gtas ../ops/linux/gtas_linux
cd ../ops/linux
# copy tvm lib
mkdir -p gtas_linux/py
cp ../../../src/tvm/py/time.py ./gtas_linux/py
cp ../../../src/tvm/py/coin.py ./gtas_linux/py
#
zip -r gtas_linux_v${version}.zip gtas_linux
