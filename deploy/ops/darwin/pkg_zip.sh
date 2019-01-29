#!/bin/bash

version=0.6.0

cd ../../daily
sh build.sh
cp -f gtas ../ops/darwin/gtas_mac
cd ../ops/darwin
zip -r gtas_mac_v${version}.zip gtas_mac
