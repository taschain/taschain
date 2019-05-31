#!/bin/bash

version=0.9.0

cd ../../../
sh build.sh gtas
cp -f bin/gtas deploy/ops/linux/gtas_linux
cd deploy/ops/linux
# copy tvm lib
mkdir -p gtas_linux/py
cp ../../../tvm/py/time.py ./gtas_linux/py
cp ../../../tvm/py/coin.py ./gtas_linux/py
#
zip -r gtas_linux_v${version}.zip gtas_linux
