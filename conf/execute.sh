#!/bin/bash

export GOPATH=/home/gopath:${WORKSPACE}
rm -f ./gtas
echo 'building gtas...'
go build -o ./gtas ./src/gtas/main.go
echo 'gtas build finished.'

run='run'
remote_home="/root/tas"
user="root"
simple_server_home=/root/simple_server

html=summary_$BUILD_ID.html
cp src/gtas/fronted/summary.html $simple_server_home
mv $simple_server_home/summary.html $simple_server_home/$html

cp -r conf/* ../$run
mv gtas ../$run
cd ../$run

#生成各个实例的启停脚本并获取部署的机器ip
hosts=`python parse_start.py $start_config`


set -x
OLD_BUILD_ID=$BUILD_ID
echo $OLD_BUILD_ID
BUILD_ID=dontKillMe

host_arr=(${hosts//,/ })
echo 'host arr', $host_arr

for host in ${host_arr[@]}
do
  echo 'start rsync to host ', $host
  rsync -rvatz --progress ../$run $user@${host}:$remote_home
  echo 'rsync finished. start ssh to host', $host

  start_sh=`python parse_start.py $host`
  ssh $user@${host} "cd $remote_home/$run; bash -x $start_sh; exit"
  echo 'host start instants finished', $host
done

#改回原来的BUILD_ID值
BUILD_ID=$OLD_BUILD_ID
echo $BUILD_ID

hp=`cat host_port`
sed -i "s/__HOSTS__/$hp/g" $simple_server_home/$html

echo http://10.0.0.12:8000/${html}
