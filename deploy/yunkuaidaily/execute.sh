#!/bin/bash

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/home/go:${WORKSPACE}

OLD_BUILD_ID=$BUILD_ID
echo 'build id is: '$OLD_BUILD_ID


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
hosts=`cat host_list`
echo -e 'romote  hosts: '$hosts'\n'
host_arr=(${hosts//,/ })

#remote stop
echo 'Stop program...\n'
#stop heavy nodes
for host in ${hosts[@]}
do
  echo -e 'stoping heavy host: '$host
  #ssh to remote host,stop previous program and clean logs and database
  ssh $user@${host} "mkdir -p $remote_run_home;cd $remote_run_home;bash -x stop.sh;exit"
  if [ $clear_data = true ]; then
  	ssh $user@${host} "cd $remote_run_home;rm -rf d*; rm -rf logs/*;exit"
  fi
  echo -e '\n'
done

#remote start
echo 'Booting...\n'

#start heavy nodes
cd ..
instance_index=1
node_num_per_host=3
genesis_host_num=9

host_count=1
for host in ${hosts[@]}
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

  for((i=instance_index;i<instance_index+node_num_per_host;i++))
  do
      port=$[8100+$i]
      echo -n "${host}:${port},">> instance_info
  done

  instance_index=$(($instance_index+$node_num_per_host))
  echo -e $host' started\n\n'
done

#改回原来的BUILD_ID值
BUILD_ID=$OLD_BUILD_ID

instance_list=`cat instance_info`
instance_list=${instance_list%?}
sed -i "s/__HOSTS__/$instance_list/g" $log_dir/$html
echo http://logs.taschain.com/yunkuai-test/${html}