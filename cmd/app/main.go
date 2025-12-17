package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github-gold-miner/internal/adapter/analyzer"
	"github-gold-miner/internal/adapter/feishu"
	"github-gold-miner/internal/adapter/filter"
	"github-gold-miner/internal/adapter/gemini"
	"github-gold-miner/internal/adapter/github"
	"github-gold-miner/internal/adapter/repository"
	"github-gold-miner/internal/domain"
	"github-gold-miner/internal/port"
)

func main() {
	// 1. 定义命令行参数
	mode := flag.String("mode", "mine", "运行模式: mine (挖矿) 或 search (搜索)")
	query := flag.String("q", "", "搜索关键词 (仅在 search 模式下有效)")
	interval := flag.Int("interval", 0, "定时执行间隔（分钟），0表示只执行一次")
	concurrency := flag.Int("concurrency", 3, "LLM分析并发数")
	flag.Parse()

	// 2. 初始化公共依赖 (数据库)
	// 确保环境变量已设置
	dsn := "host=localhost user=postgres password=123456 dbname=gold_miner port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	repoStore, err := repository.NewPostgresRepo(dsn)
	if err != nil {
		log.Fatalf("❌ DB 初始化失败: %v", err)
	}

	// 3. 初始化 AI 依赖
	ctx := context.Background()
	geminiKey := os.Getenv("GEMINI_API_KEY")
	appraiser, err := gemini.NewGeminiAppraiser(ctx, geminiKey)
	if err != nil {
		log.Fatalf("❌ AI 初始化失败: %v", err)
	}

	// 4. 根据模式分流
	if *interval > 0 {
		// 定时执行模式
		runScheduledMining(repoStore, appraiser, *interval, *concurrency)
	} else {
		// 单次执行模式
		switch *mode {
		case "search":
			runSearch(repoStore, appraiser, *query)
		case "mine":
			runMining(repoStore, appraiser, *concurrency)
		default:
			fmt.Println("❌ 未知模式，请使用 -mode=mine 或 -mode=search")
		}
	}
}

// runScheduledMining 运行定时挖矿任务
func runScheduledMining(repoStore port.Repository, appraiser port.Appraiser, interval int, concurrency int) {
	// 创建带取消功能的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置信号处理，优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()
	
	fmt.Printf("⏰ 定时执行模式已启动，每 %d 分钟执行一次\n", interval)
	fmt.Println("按下 Ctrl+C 可以优雅停止程序")
	
	// 立即执行一次
	executeMiningCycle(repoStore, appraiser, concurrency)
	
	// 定时执行
	for {
		select {
		case <-ticker.C:
			executeMiningCycle(repoStore, appraiser, concurrency)
		case <-sigChan:
			fmt.Println("\n👋 收到停止信号，正在退出...")
			return
		case <-ctx.Done():
			fmt.Println("👋 定时任务已停止")
			return
		}
	}
}

// executeMiningCycle 执行一次挖矿周期
func executeMiningCycle(repoStore port.Repository, appraiser port.Appraiser, concurrency int) {
	// 获取环境变量
	githubToken := os.Getenv("GITHUB_TOKEN")
	feishuWebhook := os.Getenv("FEISHU_WEBHOOK")

	// 为整个挖矿周期设置超时时间(5分钟)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 初始化组件
	fetcher := github.NewFetcher(githubToken)
	repoFilter := filter.NewRepoFilter(githubToken)
	repoAnalyzer := analyzer.NewRepoAnalyzer(appraiser)
	repoAnalyzer.SetMaxGoroutines(concurrency) // 设置并发数
	notifier := feishu.NewNotifier(feishuWebhook)

	fmt.Println("🚀 [挖矿模式] 开始搜寻AI编程工具金矿...")

	// 1. 数据源 (Fetcher)
	fmt.Println("📥 正在抓取 GitHub Trending 项目...")
	trendingRepos, err := fetcher.GetTrendingRepos(ctx, "all", "weekly")
	if err != nil {
		log.Printf("❌ 获取 trending repos 失败: %v", err)
	} else {
		fmt.Printf("✅ 成功获取 %d 个 trending 项目\n", len(trendingRepos))
	}

	// 获取指定 topics 的项目
	topics := []string{"ai-coding", "ide-extension", "dev-tools"}
	var topicRepos []*domain.Repo
	for _, topic := range topics {
		fmt.Printf("📥 正在抓取 topic '%s' 的项目...\n", topic)
		repos, err := fetcher.GetReposByTopic(ctx, topic)
		if err != nil {
			log.Printf("❌ 获取 topic '%s' 的 repos 失败: %v", topic, err)
			continue
		}
		topicRepos = append(topicRepos, repos...)
		fmt.Printf("✅ 成功获取 %d 个 '%s' topic 项目\n", len(repos), topic)
	}

	// 合并所有项目
	allRepos := append(trendingRepos, topicRepos...)

	// 2. 初筛漏斗 (Hard Filter)
	fmt.Println("🔍 开始初筛...")
	// 时效性过滤：创建时间在10天内
	filteredRepos := repoFilter.FilterByCreatedAt(allRepos, 10)
	fmt.Printf("✅ 时效性过滤后剩余 %d 个项目\n", len(filteredRepos))

	// 活跃度过滤：近期有commit提交
	filteredRepos, err = repoFilter.FilterByRecentCommit(ctx, filteredRepos)
	if err != nil {
		log.Printf("⚠️ 活跃度过滤出错: %v", err)
	}
	fmt.Printf("✅ 活跃度过滤后剩余 %d 个项目\n", len(filteredRepos))

	// 3. 深度分析 (Analyzer)
	fmt.Println("🧠 开始深度分析...")
	// 数学模型分析：计算Star增长速率
	reposWithGrowthRate := repoAnalyzer.CalculateStarGrowthRate(filteredRepos)
	fmt.Printf("✅ 已计算 %d 个项目的Star增长速率\n", len(reposWithGrowthRate))

	// LLM分析：判断是否为AI编程工具并评分
	analyzedRepos, err := repoAnalyzer.AnalyzeWithLLM(ctx, reposWithGrowthRate)
	if err != nil {
		log.Printf("⚠️ LLM分析出错: %v", err)
	}
	fmt.Printf("✅ 已完成 %d 个项目的LLM分析\n", len(analyzedRepos))

	// 4. 存储和推送
	fmt.Println("💾 开始存储和推送...")
	successCount := 0
	for _, repo := range analyzedRepos {
		// 检查context是否已超时或取消
		select {
		case <-ctx.Done():
			fmt.Println("⏰ 执行时间过长，提前结束存储和推送阶段")
			goto finish
		default:
		}

		// 只处理被识别为AI编程工具且评分较高的项目
		// 降低阈值以便更容易推送项目进行测试
		if !repo.IsAIProgrammingTool || repo.LLMScore < 50 {
			continue
		}

		// 检查是否已存在
		exists, _ := repoStore.Exists(ctx, repo.ID)
		if exists {
			fmt.Printf("⏭️ 项目 %s 已存在\n", repo.Name)
			continue
		}

		// 保存到数据库
		if err := repoStore.Save(ctx, repo); err != nil {
			log.Printf("❌ 保存项目 %s 失败: %v", repo.Name, err)
			continue
		}

		// 推送到飞书
		if err := notifier.Notify(ctx, repo); err != nil {
			log.Printf("⚠️ 推送项目 %s 失败: %v", repo.Name, err)
		} else {
			// 标记为已推送
			repoStore.MarkAsNotified(ctx, repo.ID)
			fmt.Printf("📲 已推送项目 %s\n", repo.Name)
			successCount++
		}

		// 避免触发 API 限制
		time.Sleep(3 * time.Second)
	}

finish:
	fmt.Printf("🎉 本轮挖矿完成，共推送 %d 个项目\n", successCount)
}

// --- 搜索模式逻辑 ---
func runSearch(repoStore port.Repository, appraiser port.Appraiser, query string) {
	if query == "" {
		fmt.Println("⚠️ 请输入你的需求，用大白话就行。")
		fmt.Println("例如: -q '我想找一个Python的机器学习库' 或 -q '有没有好用的代码生成工具'")
		return
	}

	fmt.Println("🤖 正在读取数据库，并进行 AI 语义分析...")

	// 1. 取出候选项目 (比如最近入库的 50 个)
	candidates, err := repoStore.GetAllCandidates(context.Background())
	if err != nil {
		log.Fatalf("读取数据库失败: %v", err)
	}

	if len(candidates) == 0 {
		fmt.Println("📭 数据库是空的。请先运行 -mode=mine 抓取一些项目！")
		return
	}

	fmt.Printf("📚 已加载 %d 个项目作为上下文，AI 正在匹配你的需求: [%s] ...\n", len(candidates), query)

	// 2. 这里的 query 不再是 SQL 关键词，而是你的自然语言问题
	answer, err := appraiser.SemanticSearch(context.Background(), candidates, query)
	if err != nil {
		log.Printf("❌ AI 分析失败: %v", err)
		return
	}

	// 3. 打印结果
	fmt.Println("\n================ [ 智能搜索结果 ] ================")
	fmt.Println(answer)
	fmt.Println("==================================================")
}

// --- 挖矿模式逻辑 ---
func runMining(repoStore port.Repository, appraiser port.Appraiser, concurrency int) {
	executeMiningCycle(repoStore, appraiser, concurrency)
}