# Github Gold Miner (AI Tool Edition)

## 项目介绍

这是一个自动化的AI编程工具挖掘系统，能够从GitHub上发现最近创建的、具有高增长潜力的AI编程工具项目，并通过飞书推送给开发者。

## 核心功能

1. **数据采集**：抓取GitHub Trending项目和指定Topics下的项目
2. **规则过滤**：过滤掉创建时间超过10天或没有近期提交的项目
3. **AI分析**：使用LLM判断项目是否为AI编程工具并进行评分
4. **数据存储**：使用PostgreSQL存储项目信息，防止重复推送
5. **消息推送**：将符合条件的项目通过飞书Webhook推送到群聊

## 技术架构

```
Fetcher (采集) -> Filter (硬规则过滤) -> Analyzer (数学+AI分析) -> Storage (去重) -> Notifier (推送)
```

## 环境变量

- `GITHUB_TOKEN`: GitHub Personal Access Token
- `GEMINI_API_KEY`: Gemini API Key
- `FEISHU_WEBHOOK`: 飞书群机器人Webhook地址
- `DATABASE_URL`: PostgreSQL数据库连接字符串

## 运行方式

```bash
# 挖掘AI编程工具项目（单次执行）
go run cmd/app/main.go -mode=mine

# 搜索已存储的项目
go run cmd/app/main.go -mode=search -q="机器学习库"

# 定时执行（每30分钟执行一次）
go run cmd/app/main.go -mode=mine -interval=30

# 控制并发数（默认为3）
go run cmd/app/main.go -mode=mine -concurrency=5
```

> 提示：如果你希望实时收到推送，请在运行前设置 `FEISHU_WEBHOOK` 环境变量，否则服务会保守地跳过通知步骤。

### 执行脚本

仓库自带 `scripts/run_mining.sh`，会在进入项目目录后以 30 分钟间隔、5 个并发持续运行挖矿流程，并将执行结果写入 `/var/log/github-gold-miner.log`。默认脚本也展示了如何配置代理环境变量；根据需要修改代理地址、执行参数或日志路径，然后配合 `chmod +x scripts/run_mining.sh` 与 `crontab`/`launchd` 等调度工具即可实现无人值守运行。

## 定时执行配置

定时执行模式支持优雅关闭，当收到SIGINT或SIGTERM信号时会完成当前任务后退出。

使用定时执行模式：
```bash
# 每30分钟执行一次，最大并发数为5
./bin/github-gold-miner -mode=mine -interval=30 -concurrency=5
```

在定时执行模式下，按下 Ctrl+C 可以优雅停止程序。

除了使用内置的定时执行功能，您还可以使用系统的cron服务来定时执行任务。

1. 编辑crontab：
   ```bash
   crontab -e
   ```

2. 添加定时任务（例如每小时执行一次）：
   ```bash
   0 * * * * /Users/liujiahao/GolandProjects/github-gold-miner/scripts/run_mining.sh
   ```

3. 查看当前的cron任务：
   ```bash
   crontab -l
   ```

## 配置说明

### 数据库配置

项目使用PostgreSQL作为存储后端，默认连接配置为：
```
host=localhost user=postgres password=123456 dbname=gold_miner port=5432 sslmode=disable TimeZone=Asia/Shanghai
```

### 项目过滤规则

1. 项目创建时间不超过10天
2. 项目近期有代码提交活动
3. 项目被LLM识别为AI编程工具且评分≥50

### 并发控制

LLM分析阶段支持并发执行，默认并发数为3。可以通过 `-concurrency` 参数调整并发数：
- 较高的并发数可以加快分析速度，但可能会触发API限制
- 较低的并发数可以减少API压力，但分析时间会相应增加

## 开发指南

### 项目结构

```
├── cmd/
│   └── app/           # 主程序入口
├── internal/
│   ├── adapter/       # 适配器实现
│   │   ├── analyzer/  # 分析器
│   │   ├── filter/    # 过滤器
│   │   ├── github/    # GitHub数据源
│   │   ├── gemini/    # Gemini AI分析
│   │   ├── feishu/    # 飞书推送
│   │   └── repository/ # 数据库存储
│   ├── domain/        # 领域模型
│   └── port/          # 接口定义
```
