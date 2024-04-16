#!/bin/bash


CURRENT_DIR=$(cd $(dirname $0);pwd)
BASE_DIR=$(cd ..;pwd)

mkdir -p "$BASE_DIR/logs"
echo $BASE_DIR
cd $BASE_DIR
sudo setcap cap_net_bind_service=+eip  $BASE_DIR/proxy_server
nohup $BASE_DIR/proxy_server -backend_addr=localhost.sangfor.com.cn:8081 -proxy_addr=0.0.0.0:8082 >>  $BASE_DIR/logs/web_proxy.log 2>&1 &
ps -ef | grep "proxy_server" | grep -v "grep" | awk '{print $2}'
