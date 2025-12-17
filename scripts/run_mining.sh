#!/bin/bash

export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897
# 进入项目目录
cd /Users/liujiahao/GolandProjects/github-gold-miner

# 执行挖矿任务
./bin/github-gold-miner -mode=mine

echo "$(date): 挖矿任务执行完成" >> /var/log/github-gold-miner.log