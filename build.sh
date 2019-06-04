#!/usr/bin/env bash
basepath=$(cd `dirname $0`; pwd)
output_dir=${basepath}/bin/
function buildtvm() {
    sh $basepath/tvm/ctvm/buildlib.sh &&
    cp $basepath/tvm/ctvm/examples/embedding/libtvm.a $basepath/tvm/ &&
    cp $basepath/tvm/ctvm/py/tvm.h $basepath/tvm/
    if [ $? -ne 0 ];then
        exit 1
    fi
}

function buildp2p() {
    if [[ `uname -s` = "Darwin" ]]; then
        cd network/p2p/platform/darwin &&
        make &&
        mv ${basepath}/network/p2p/bin/libp2pcore.a $basepath/network/ &&
        cp ${basepath}/network/p2p/p2p_api.h $basepath/network/
    else
        cd network/p2p/platform/linux &&#
        make &&
        mv ${basepath}/network/p2p/bin/libp2pcore.a $basepath/network/ &&
        cp ${basepath}/network/p2p/p2p_api.h $basepath/network/
    fi
    if [ $? -ne 0 ];then
        exit 1
    fi
}
#git submodule update --init
if [[ $1x = "gtas"x ]]; then
    echo building gtas ...
    #buildtvm
    #buildp2p

    go build -o ${output_dir}/gtas $basepath/cmd/gtas &&
    echo build gtas successfully...

    elif [[ $1x = "tvmcli"x ]]; then
    go build $basepath/cmd/tvmcli &&
    echo build tvmcli successfully...
    elif [[ $1x = "clean"x ]]; then
    rm $basepath/tvm/tvm.h $basepath/tvm/libtvm.a
    rm $basepath/network/p2p_api.h $basepath/network/libp2pcore.a
    echo cleaned
fi