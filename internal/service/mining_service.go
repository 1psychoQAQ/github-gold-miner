package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github-gold-miner/internal/domain"
	"github-gold-miner/internal/port"
)

// MiningService 处理挖矿逻辑
type MiningService struct {
	fetcher    port.Scouter
	filter     port.Filter
	analyzer   port.Analyzer
	repoStore  port.Repository
	appraiser  port.Appraiser
	notifier   port.Notifier
}

// NewMiningService 创建新的挖矿服务
func NewMiningService(
	fetcher port.Scouter,
	filter port.Filter,
	analyzer port.Analyzer,
	repoStore port.Repository,
	appraiser port.Appraiser,
	notifier port.Notifier,
) *MiningService {
	return &MiningService{
		fetcher:   fetcher,
		filter:    filter,
		analyzer:  analyzer,
		repoStore: repoStore,
		appraiser: appraiser,
		notifier:  notifier,
	}
}

// ExecuteMiningCycle 执行一次挖矿周期
func (m *MiningService) ExecuteMiningCycle(ctx context.Context, concurrency int) error {
	// 设置并发数
	m.analyzer.SetMaxGoroutines(concurrency)

	fmt.Println("🚀 [挖矿模式] 开始搜寻AI编程工具金矿...")

	// 1. 数据源 (Fetcher)
	fmt.Println("📥 正在抓取 GitHub Trending 项目...")
	trendingRepos, err := m.fetcher.GetTrendingRepos(ctx, "all", "weekly")
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
		repos, err := m.fetcher.GetReposByTopic(ctx, topic)
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
	filteredRepos := m.filter.FilterByCreatedAt(allRepos, 10)
	fmt.Printf("✅ 时效性过滤后剩余 %d 个项目\n", len(filteredRepos))

	// 活跃度过滤：近期有commit提交
	filteredRepos, err = m.filter.FilterByRecentCommit(ctx, filteredRepos)
	if err != nil {
		log.Printf("⚠️ 活跃度过滤出错: %v", err)
		// 如果活跃度过滤出错，我们仍然可以继续处理已有的项目
	}
	fmt.Printf("✅ 活跃度过滤后剩余 %d 个项目\n", len(filteredRepos))

	// 3. 深度分析 (Analyzer)
	fmt.Println("🧠 开始深度分析...")
	// 数学模型分析：计算Star增长速率
	reposWithGrowthRate := m.analyzer.CalculateStarGrowthRate(filteredRepos)
	fmt.Printf("✅ 已计算 %d 个项目的Star增长速率\n", len(reposWithGrowthRate))

	// LLM分析：判断是否为AI编程工具并评分
	analyzedRepos, err := m.analyzer.AnalyzeWithLLM(ctx, reposWithGrowthRate)
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
		exists, err := m.repoStore.Exists(ctx, repo.ID)
		if err != nil {
			log.Printf("❌ 检查项目 %s 是否存在时出错: %v，跳过该项目", repo.Name, err)
			continue
		}
		if exists {
			fmt.Printf("⏭️ 项目 %s 已存在\n", repo.Name)
			continue
		}

		// 保存到数据库
		if err := m.repoStore.Save(ctx, repo); err != nil {
			log.Printf("❌ 保存项目 %s 失败: %v", repo.Name, err)
			continue
		}

		if m.notifier == nil {
			log.Printf("⚠️ 未配置通知通道，跳过推送项目 %s", repo.Name)
			continue
		}

		if err := m.notifier.Notify(ctx, repo); err != nil {
			log.Printf("❌ 推送项目 %s 到通知通道失败: %v", repo.Name, err)
			continue
		}

		if err := m.repoStore.MarkAsNotified(ctx, repo.ID); err != nil {
			log.Printf("⚠️ 标记项目 %s 为已通知失败: %v", repo.Name, err)
			continue
		}
		fmt.Printf("📲 已处理项目 %s\n", repo.Name)
		successCount++

		// 避免触发 API 限制
		time.Sleep(3 * time.Second)
	}

finish:
	fmt.Printf("🎉 本轮挖矿完成，共处理 %d 个项目\n", successCount)
	return nil
}
