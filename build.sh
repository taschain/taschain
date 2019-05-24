#!/usr/bin/env bash
basepath=$(cd `dirname $0`; pwd)
if [[ $1x = "gtas"x ]]; then
    echo building gtas ...
    if [ ! -f ${basepath}/tvm/libtvm.a ] || [ ! -f ${basepath}/tvm/tvm.h ] ; then
        git submodule update &&
        cd $basepath/tvm/ctvm &&
        git checkout master && git pull &&
        sh $basepath/tvm/ctvm/buildlib.sh &&
        cp $basepath/tvm/ctvm/examples/embedding/libtvm.a $basepath/tvm/ &&
        cp $basepath/tvm/ctvm/py/tvm.h $basepath/tvm/
    fi
    go build $basepath/cmd/gtas &&
    echo build gtas successfully...

    elif [[ $1x = "tvmcli"x ]]; then
    go build $basepath/cmd/tvmcli
    echo build tvmcli successfully...
    elif [[ $1x = "clean"x ]]; then
    rm $basepath/tvm/tvm.h $basepath/tvm/libtvm.a
    echo cleaned
fi