#!/bin/bash
# 间隔执行模式：每 N 分钟执行一次

cd "$(dirname "$0")/.."

# 代理设置（按需修改）
# export https_proxy=http://127.0.0.1:7897
# export http_proxy=http://127.0.0.1:7897

./bin/github-gold-miner -mode=mine -interval=30 -concurrency=5
