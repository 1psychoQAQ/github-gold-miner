#!/bin/bash
# 定点执行模式：使用 cron 表达式，如每天 9:30

cd "$(dirname "$0")/.."

# 代理设置（按需修改）
# export https_proxy=http://127.0.0.1:7897
# export http_proxy=http://127.0.0.1:7897

./bin/github-gold-miner -schedule="30 9 * * *" -concurrency=5
