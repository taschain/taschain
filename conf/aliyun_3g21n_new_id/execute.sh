#!/bin/bash

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/home/go:${WORKSPACE}

OLD_BUILD_ID=$BUILD_ID
echo 'build id is: '$OLD_BUILD_ID


#build gtas in /var/lib/jenkins/workspace/tas_aliyun_3g21n_new_id
rm -f ./gtas
echo 'building gtas...'
go build -o ./gtas ./src/gtas/main.go
echo -e 'gtas build finished.\n'


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
cp -r lib/linux/p2p_core.so $run_dir/
cp conf/$config_dir/stop.sh $run_dir/
cp conf/$config_dir/start.sh $run_dir/
cp conf/$config_dir/host_list $run_dir
cp conf/$config_dir/sync_list $run_dir

rm -rf $run_dir/genesis_config
mkdir $run_dir/genesis_config
cp -r conf/$config_dir/tas*.ini $run_dir/genesis_config
cp -r conf/$config_dir/joined_group.config.* $run_dir/genesis_config


cd $run_dir
hosts=`cat host_list`
echo -e 'romote hosts: '$hosts'\n'

host_arr=(${hosts//,/ })

for host in ${host_arr[@]}
do
  echo -e 'stoping host: '$host
  #ssh to remote host,stop previous program and clean logs and database
  ssh $user@${host} "mkdir -p $remote_home/$run_dir;cd $remote_home/$run_dir;bash -x stop.sh;rm -rf *;exit"
  if [ $clear_data = true ]; then
  	ssh $user@${host} "cd $remote_home/$run_dir;rm -rf d*; rm -rf logs/*;exit"
  fi
  if [ $clear_ini = true ]; then
    ssh $user@${host} "cd $remote_home/$run_dir;rm -rf tas*.ini; rm -rf joined_group.config*;exit"
  fi
  echo -e '\n'
done
$stop_only && exit

instance_index=1
#每台机器部署实例数量
instance_count=7
host_count=1

for host in ${host_arr[@]}
do
  if [ $host_count -eq 1 ];then
      cd genesis_config
      rsync  -rvatz --progress . $user@${host}:$remote_home/$run_dir
      cd ..
  fi
  host_count=$(($host_count+1))
  #sync config file to host
  echo 'start sync config file to host:'$host
  rsync -rvatz --progress --include-from=sync_list --exclude=/* . $user@${host}:$remote_home/$run_dir

  echo -e '\n'$host'  booting...'
  #ssh to remote host to start to run new program
  ssh $user@${host} "cd $remote_home/$run_dir;bash -x start.sh $instance_index $instance_count;exit"


  for((i=instance_index;i<instance_index+instance_count;i++))
  do
      port=$[8100+$i]
      echo -n "${host}:${port},">> instance_info
  done

  instance_index=$(($instance_index+$instance_count))
  echo -e $host' started\n\n'
done


#改回原来的BUILD_ID值
BUILD_ID=$OLD_BUILD_ID

instance_list=`cat instance_info`
instance_list=${instance_list%?}
sed -i "s/__HOSTS__/$instance_list/g" $log_dir/$html
echo http://logs.taschain.com/page/${html}