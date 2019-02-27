#!/bin/bash

version=0.7.5

cd ../../daily
sh build.sh
cp -f gtas ../ops/darwin/gtas_mac
cd ../ops/darwin
# copy tvm lib
mkdir gtas_mac/py
cp ../../../src/tvm/py/time.py ./gtas_mac/py
cp ../../../src/tvm/py/coin.py ./gtas_mac/py
#
zip -r gtas_mac_v${version}.zip gtas_mac
