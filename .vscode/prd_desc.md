项目重构需求：Github Gold Miner (AI Tool Edition)
1. 核心目标 从 GitHub 海量项目中，利用“规则+AI”双重过滤，自动挖掘出创建时间在 10 天内、且具备高增长潜力的靠谱 AI 编程工具，并推送到飞书。

2. 核心流程 Fetcher (采集) -> Filter (硬规则过滤) -> Analyzer (数学+AI分析) -> Storage (去重) -> Notifier (推送)

3. 详细功能定义

A. 数据源 (Fetcher)

范围：

GitHub Trending (All languages) 前 10 名。

指定 Topics (ai-coding, ide-extension, dev-tools) 下的前 3 名。

策略：混合采集，支持配置化运行（单次/定期）。

B. 初筛漏斗 (Hard Filter)

时效性：Created At 必须在 近 10 天内。

活跃度：近期必须有 Commit 提交（避免只有 README 的空壳）。

C. 深度分析 (Analyzer) - 本次重构核心

数学模型（潜力值计算）：

你需要一个算法来判定“早期快速增长”。

拟定策略：计算 Star速率 = Stars数量 / 上线天数。对于 10 天内的新项目，我们需要设定一个阈值（例如每天至少获得 5-10 个 Star，或者符合指数增长曲线）。

语义评分（LLM）：

输入：项目的 README.md 内容。

模型：支持切换 Ollama（本地）或 在线 API。

Prompt 目标：判断该项目是否为“编程工具/IDE插件/DevOps工具”，并根据功能描述评分（1-100），剔除仅仅是“教程”、“资源列表”或“PPT项目”。

D. 持久化与去重 (Storage)

技术栈：PostgreSQL。

逻辑：已推送过的 Repo ID 不再推送；但如果项目评分发生剧烈变化（如翻红），可考虑二次更新（本次先按“不重复推送”处理）。

E. 触达 (Notifier)

渠道：飞书 Webhook。

内容：项目名、简介、Star 增速、LLM 评分及简评、GitHub 链接。