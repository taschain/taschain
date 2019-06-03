#!/bin/bash

version=0.9.0

cd ../../../
sh build.sh gtas
cp -f bin/gtas deploy/ops/darwin/gtas_mac
cd deploy/ops/darwin
# copy tvm lib
mkdir gtas_mac/py
cp ../../../tvm/py/time.py ./gtas_mac/py
cp ../../../tvm/py/coin.py ./gtas_mac/py
#
zip -r gtas_mac_v${version}.zip gtas_mac
