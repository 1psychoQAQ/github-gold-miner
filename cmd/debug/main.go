package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github-gold-miner/internal/adapter/analyzer"
	"github-gold-miner/internal/adapter/filter"
	"github-gold-miner/internal/adapter/gemini"
	"github-gold-miner/internal/adapter/github"
)

func main() {
	// 获取环境变量
	githubToken := os.Getenv("GITHUB_TOKEN")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	ctx := context.Background()

	// 初始化组件
	fetcher := github.NewFetcher(githubToken)
	repoFilter := filter.NewRepoFilter(githubToken)
	appraiser, err := gemini.NewGeminiAppraiser(ctx, geminiKey)
	if err != nil {
		log.Fatalf("❌ AI 初始化失败: %v", err)
	}
	repoAnalyzer := analyzer.NewRepoAnalyzer(appraiser)

	fmt.Println("🔍 调试模式：获取并分析项目")

	// 1. 获取一些项目用于测试
	fmt.Println("📥 正在抓取 GitHub Trending 项目...")
	trendingRepos, err := fetcher.GetTrendingRepos(ctx, "all", "weekly")
	if err != nil {
		log.Printf("❌ 获取 trending repos 失败: %v", err)
		return
	}
	fmt.Printf("✅ 成功获取 %d 个 trending 项目\n", len(trendingRepos))

	if len(trendingRepos) == 0 {
		fmt.Println("❌ 没有获取到任何项目")
		return
	}

	// 2. 初筛漏斗 (Hard Filter)
	fmt.Println("🔍 开始初筛...")
	// 时效性过滤：创建时间在10天内
	filteredRepos := repoFilter.FilterByCreatedAt(trendingRepos, 10)
	fmt.Printf("✅ 时效性过滤后剩余 %d 个项目\n", len(filteredRepos))

	if len(filteredRepos) == 0 {
		fmt.Println("❌ 时效性过滤后没有剩余项目")
		return
	}

	// 活跃度过滤：近期有commit提交
	filteredRepos, err = repoFilter.FilterByRecentCommit(ctx, filteredRepos)
	if err != nil {
		log.Printf("⚠️ 活跃度过滤出错: %v", err)
	}
	fmt.Printf("✅ 活跃度过滤后剩余 %d 个项目\n", len(filteredRepos))

	if len(filteredRepos) == 0 {
		fmt.Println("❌ 活跃度过滤后没有剩余项目")
		return
	}

	// 3. 深度分析 (Analyzer)
	fmt.Println("🧠 开始深度分析...")
	// 数学模型分析：计算Star增长速率
	reposWithGrowthRate := repoAnalyzer.CalculateStarGrowthRate(filteredRepos)
	fmt.Printf("✅ 已计算 %d 个项目的Star增长速率\n", len(reposWithGrowthRate))

	// LLM分析：判断是否为AI编程工具并评分
	fmt.Printf("🧠 对前%d个项目进行LLM分析:\n", min(3, len(reposWithGrowthRate)))
	for i, repo := range reposWithGrowthRate {
		if i >= 3 { // 只分析前3个项目以节省时间和API调用
			break
		}
		
		fmt.Printf("  分析项目 #%d: %s\n", i+1, repo.Name)
		analyzedRepo, err := appraiser.Appraise(ctx, repo)
		if err != nil {
			log.Printf("    ⚠️ 分析失败: %v", err)
			continue
		}
		
		fmt.Printf("    是否AI工具: %v\n", analyzedRepo.IsAIProgrammingTool)
		fmt.Printf("    LLM评分: %d\n", analyzedRepo.LLMScore)
		fmt.Printf("    LLM评价: %s\n", analyzedRepo.LLMReview)
		fmt.Printf("    Star增长速率: %.2f stars/天\n", analyzedRepo.StarGrowthRate)
		fmt.Println()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}