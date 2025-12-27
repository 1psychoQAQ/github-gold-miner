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

## 快速开始

```bash
# 构建
go build -o bin/github-gold-miner cmd/app/main.go

# 单次执行
./bin/github-gold-miner -mode=mine

# 间隔执行（每30分钟）
./bin/github-gold-miner -interval=30 -concurrency=5

# 定点执行（每天9:30）
./bin/github-gold-miner -schedule="30 9 * * *" -concurrency=5

# 语义搜索
./bin/github-gold-miner -mode=search -q="代码生成工具"
```

**启动脚本:** `scripts/run_interval.sh`（间隔模式）、`scripts/run_scheduled.sh`（定点模式）

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
