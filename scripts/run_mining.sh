#!/bin/bash

# 设置代理
export https_proxy=http://127.0.0.1:7897
export http_proxy=http://127.0.0.1:7897


# 进入项目目录
cd /Users/liujiahao/GolandProjects/github-gold-miner

# 执行挖矿任务，每30分钟一次，5个并发
./bin/github-gold-miner -mode=mine -interval=30 -concurrency=5

echo "$(date): 挖矿任务执行完成" >> /var/log/github-gold-miner.log