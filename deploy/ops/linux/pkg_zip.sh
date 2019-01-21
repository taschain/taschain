#!/bin/bash

version=0.5.7

cd ../../daily
sh build.sh
cp -f gtas ../ops/linux/gtas_linux
cd ../ops/linux
zip -r gtas_linux_v${version}.zip gtas_linux
