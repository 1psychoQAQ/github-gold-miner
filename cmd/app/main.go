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
	"github-gold-miner/internal/port"
	"github-gold-miner/internal/service"
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

	// 初始化通知器
	feishuWebhook := os.Getenv("FEISHU_WEBHOOK")
	notifier := feishu.NewNotifier(feishuWebhook)

	// 4. 根据模式分流
	if *interval > 0 {
		// 定时执行模式
		runScheduledMining(repoStore, appraiser, notifier, *interval, *concurrency)
	} else {
		// 单次执行模式
		switch *mode {
		case "search":
			runSearch(repoStore, appraiser, *query)
		case "mine":
			runMining(repoStore, appraiser, notifier, *concurrency)
		default:
			fmt.Println("❌ 未知模式，请使用 -mode=mine 或 -mode=search")
		}
	}
}

// runScheduledMining 运行定时挖矿任务
func runScheduledMining(repoStore port.Repository, appraiser port.Appraiser, notifier port.Notifier, interval int, concurrency int) {
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
	executeMiningCycle(repoStore, appraiser, notifier, concurrency)
	
	// 定时执行
	for {
		select {
		case <-ticker.C:
			executeMiningCycle(repoStore, appraiser, notifier, concurrency)
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
func executeMiningCycle(repoStore port.Repository, appraiser port.Appraiser, notifier port.Notifier, concurrency int) {
	// 获取环境变量
	githubToken := os.Getenv("GITHUB_TOKEN")

	// 为整个挖矿周期设置超时时间(5分钟)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 初始化组件
	fetcher := github.NewFetcher(githubToken)
	repoFilter := filter.NewRepoFilter(githubToken)
	repoAnalyzer := analyzer.NewRepoAnalyzer(appraiser)
	repoAnalyzer.SetMaxGoroutines(concurrency) // 设置并发数

	// 创建挖矿服务
	miningService := service.NewMiningService(fetcher, repoFilter, repoAnalyzer, repoStore, appraiser, notifier)

	// 执行挖矿周期
	miningService.ExecuteMiningCycle(ctx, concurrency)
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
func runMining(repoStore port.Repository, appraiser port.Appraiser, notifier port.Notifier, concurrency int) {
	executeMiningCycle(repoStore, appraiser, notifier, concurrency)
}
