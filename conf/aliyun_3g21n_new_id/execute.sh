#!/bin/bash

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/home/go:${WORKSPACE}

OLD_BUILD_ID=$BUILD_ID
echo 'build id is: '$OLD_BUILD_ID


#build gtas in /var/lib/jenkins/workspace/tas_aliyun_3g21n_new_id
rm -f ./gtas
echo 'building gtas...'
go build -o ./gtas ./src/gtas/main.go
echo -e 'gtas build finished.\n\n'


run_dir='run'
remote_home="/home/tas"
user="root"
log_dir=${WORKSPACE}/tas_log

#prepare summary.html
rm -rf $log_dir
mkdir $log_dir
html=summary_$BUILD_ID.html
cp src/gtas/fronted/summary.html $log_dir/
mv $log_dir/summary.html $log_dir/$html
BUILD_ID=dontKillMe

#prepare run dir
rm -rf $run_dir
mkdir $run_dir

mv gtas $run_dir/
cp -r conf/$config_dir/tas*.ini $run_dir/
cp -r lib/linux/p2p_core.so $run_dir/

#prepare tools
mkdir $run_dir/tools
cp -r conf/$config_dir/start.json $run_dir/tools
cp conf/$config_dir/parse_start.py $run_dir/tools
cp conf/$config_dir/stop.sh $run_dir/tools

#生成各个实例的启停脚本并获取部署的机器ip
cd $run_dir/tools
hosts=`python parse_start.py start.json`
echo -e 'romote hosts: '$hosts'\n\n'


host_arr=(${hosts//,/ })
for host in ${host_arr[@]}
do
  echo -e 'stoping host: '$host
  #ssh to remote host,stop previous program and clean logs and database
  if [ $clear_data = true ]; then
  	ssh $user@${host} "mkdir -p $remote_home/$run_dir;cd $remote_home/$run_dir;bash -x stop.sh;rm -rf d*; rm -rf logs/*;exit"
  else
  	ssh $user@${host} "mkdir -p $remote_home/$run_dir;cd $remote_home/$run_dir;bash -x stop.sh;exit"
  fi
  echo -e '\n\n'
done
$stop_only && exit


for host in ${host_arr[@]}
do
  cd ${WORKSPACE}/$run_dir/tools
  #sync config file to host
  start_sh=`python parse_start.py $host`
  echo 'start sync config file to host:'$host
  rsync -rvatz --progress --include-from=include_list_$host --exclude=/* . $user@${host}:$remote_home/$run_dir

  cd ..
  rsync -rvatz --progress --include-from=tools/include_list_$host --exclude=/* . $user@${host}:$remote_home/$run_dir

  echo -e '\n'$host'  booting...\n'
  #ssh to remote host to start to run new program
  ssh $user@${host} "cd $remote_home/$run_dir; bash -x $start_sh;exit"

  echo -e $host' started\n\n'
done


#改回原来的BUILD_ID值
BUILD_ID=$OLD_BUILD_ID

hp=`cat ${WORKSPACE}/$run_dir/tools/host_port`
sed -i "s/__HOSTS__/$hp/g" $log_dir/$html
echo http://logs.taschain.com/file/${html}