#!/bin/bash

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/home/go:${WORKSPACE}

OLD_BUILD_ID=$BUILD_ID
echo 'build id is: '$OLD_BUILD_ID


#build gtas in /var/lib/jenkins/workspace/tas_aliyun_master
rm -f ./gtas
echo 'building gtas...'
go build -o ./gtas ./src/gtas/main.go
echo -e 'gtas build finished.\n'

#prepare summary.html
log_dir=${WORKSPACE}/tas_log
rm -rf $log_dir
mkdir $log_dir
html=summary_$BUILD_ID.html
cp src/gtas/fronted/summary.html $log_dir/
mv $log_dir/summary.html $log_dir/$html
BUILD_ID=dontKillMe

#prepare run dir
compile_dir='run'
rm -rf $compile_dir
mkdir $compile_dir

#load config
mv gtas $compile_dir/
cp lib/linux/p2p_core.so $compile_dir/

cp deploy/$config_dir/stop.sh $compile_dir/
cp deploy/$config_dir/start.sh $compile_dir/

cp deploy/$config_dir/sync_list $compile_dir
cp deploy/$config_dir/genesis_sgi.config $compile_dir

cp -r deploy/$config_dir/host $compile_dir/host
cp -r deploy/$config_dir/genesis_info $compile_dir/genesis_info


#remote deploy
remote_run_home="/data/tas/run"
user="root"

cd $compile_dir/host
hosts_heavy=`cat host_heavy`
hosts_light=`cat host_light`
echo -e 'romote heavy hosts: '$hosts_heavy'\n'
echo -e 'romote light hosts: '$hosts_light'\n'
host_heavy_arr=(${hosts//,/ })
host_light_arr=(${hosts_light//,/ })

#remote stop
echo 'Stop program...\n'
#stop heavy nodes
for host in ${hosts_heavy[@]}
do
  echo -e 'stoping heavy host: '$host
  #ssh to remote host,stop previous program and clean logs and database
  ssh $user@${host} "mkdir -p $remote_run_home;cd $remote_run_home;bash -x stop.sh;exit"
  if [ $clear_data = true ]; then
  	ssh $user@${host} "cd $remote_run_home;rm -rf d*; rm -rf logs/*;exit"
  fi
  if [ $clear_ini = true ]; then
    ssh $user@${host} "cd $remote_run_home;rm -rf tas*.ini; rm -rf joined_group.config*;rm -rf keystore*;exit"
  fi
  echo -e '\n'
done

#stop light nodes
for host in ${host_light_arr[@]}
do
  echo -e 'stoping light host: '$host
  #ssh to remote host,stop previous program and clean logs and database
  ssh $user@${host} "mkdir -p $remote_run_home;cd $remote_run_home;bash -x stop.sh;exit"
  if [ $clear_data = true ]; then
  	ssh $user@${host} "cd $remote_run_home;rm -rf d*; rm -rf logs/*;exit"
  fi
  if [ $clear_ini = true ]; then
    ssh $user@${host} "cd $remote_run_home;rm -rf tas*.ini; rm -rf joined_group.config*;rm -rf keystore*;exit"
  fi
  echo -e '\n'
done
echo 'Stop program end!\n'
$stop_only && exit

#remote start
echo 'Booting...\n'

#start heavy nodes
cd ..
instance_index=1
node_num_per_host=1
genesis_host_num=3

host_count=1
for host in ${hosts_heavy[@]}
do
  if [ $host_count -le $genesis_host_num ];then
      cd genesis_info
      rsync  -rvatz --progress . $user@${host}:$remote_run_home
      cd ..
  fi

  host_count=$(($host_count+1))
  apply='heavy'

  #sync config file to host
  echo 'start sync config file to host:'$host
  rsync -rvatz --progress --include-from=sync_list --exclude=/* . $user@${host}:$remote_run_home
  echo -e '\n'$host'  booting...'
  #ssh to remote host to start to run new program
  ssh $user@${host} "cd $remote_run_home;bash -x start.sh $instance_index $node_num_per_host $nat_server  $apply ;exit"

  port=$[8100+$instance_index]
  echo -n "${host}:${port},">> instance_info
  instance_index=$(($instance_index+$node_num_per_host))
  echo -e $host' started\n\n'
done


#start light nodes
node_num_per_host=2
for host in ${host_light_arr[@]}
do
  apply='light'

  #sync config file to host
  echo 'start sync config file to host:'$host
  rsync -rvatz --progress --include-from=sync_list --exclude=/* . $user@${host}:$remote_run_home

  echo -e '\n'$host'  booting...'
  #ssh to remote host to start to run new program
  ssh $user@${host} "cd $remote_run_home;bash -x start.sh $instance_index $node_num_per_host $nat_server  $apply ;exit"

  for((i=instance_index;i<instance_index+node_num_per_host;i++))
  do
      port=$[8100+$i]
      echo -n "${host}:${port},">> instance_info
  done

  instance_index=$(($instance_index+$node_num_per_host))
  echo -e $host' started...\n\n'
done


#改回原来的BUILD_ID值
BUILD_ID=$OLD_BUILD_ID

instance_list=`cat instance_info`
instance_list=${instance_list%?}
sed -i "s/__HOSTS__/$instance_list/g" $log_dir/$html
echo http://logs.taschain.com/master/${html}