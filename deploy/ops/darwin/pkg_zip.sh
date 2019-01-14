#!/bin/bash

version=0.5.2

cd ../daily
sh build.sh
cp -f gtas ../ops/gtas_mac
cd ../ops
zip -r gtas_mac_v${version}.zip gtas_mac
