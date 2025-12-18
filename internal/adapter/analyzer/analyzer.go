package analyzer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github-gold-miner/internal/domain"
	"github-gold-miner/internal/port"
)

// RepoAnalyzer 实现了 port.Analyzer 接口
type RepoAnalyzer struct {
	appraiser     port.Appraiser
	maxGoroutines int // 最大并发数
	nowFunc       func() time.Time
}

// NewRepoAnalyzer 创建新的分析器实例
func NewRepoAnalyzer(appraiser port.Appraiser) *RepoAnalyzer {
	return &RepoAnalyzer{
		appraiser:     appraiser,
		maxGoroutines: 3,        // 默认并发数为3
		nowFunc:       time.Now, // 便于测试注入当前时间
	}
}

// SetMaxGoroutines 设置最大并发数
func (a *RepoAnalyzer) SetMaxGoroutines(max int) {
	if max > 0 {
		a.maxGoroutines = max
	}
}

// CalculateStarGrowthRate 计算Star增长率
func (a *RepoAnalyzer) CalculateStarGrowthRate(repos []*domain.Repo) []*domain.Repo {
	current := time.Now()
	if a != nil && a.nowFunc != nil {
		current = a.nowFunc()
	}

	for _, repo := range repos {
		// 计算项目存活天数
		daysAlive := current.Sub(repo.CreatedAt).Hours() / 24
		if daysAlive <= 0 {
			repo.StarGrowthRate = 0
		} else {
			// 计算每日Star增长速率
			repo.StarGrowthRate = float64(repo.Stars) / daysAlive
		}
	}
	return repos
}

// analyzeRepoWorker 工作协程，处理单个repo的分析
func (a *RepoAnalyzer) analyzeRepoWorker(
	ctx context.Context,
	jobs <-chan *domain.Repo,
	results chan<- *domain.Repo,
	errors chan<- error,
	wg *sync.WaitGroup,
	workerID int,
) {
	defer wg.Done()

	for repo := range jobs {
		fmt.Printf("   [Worker-%d] 正在分析 %s...\n", workerID, repo.Name)

		// 为每个项目设置超时时间(30秒)
		projectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		// 使用现有的Appraiser进行分析
		analyzedRepo, err := a.appraiser.Appraise(projectCtx, repo)
		cancel() // 立即释放资源

		if err != nil {
			// 如果分析失败，记录错误
			fmt.Printf("   [Worker-%d] ❌ %s 分析失败: %v\n", workerID, repo.Name, err)
			errors <- fmt.Errorf("分析 %s 失败: %w", repo.Name, err)
			// 即使失败也返回原始repo，这样不会阻塞主流程
			results <- repo
			continue
		}

		// 更新repo信息
		repo.IsAIProgrammingTool = analyzedRepo.IsAIProgrammingTool
		repo.LLMScore = analyzedRepo.LLMScore
		repo.LLMReview = analyzedRepo.LLMReview

		fmt.Printf("   [Worker-%d] ✅ %s 分析完成 (评分: %d)\n", workerID, repo.Name, repo.LLMScore)
		results <- repo
	}
}

// AnalyzeWithLLM 使用LLM并发分析项目是否为AI编程工具及其评分
func (a *RepoAnalyzer) AnalyzeWithLLM(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error) {
	fmt.Printf("🤖 开始LLM分析，共 %d 个项目，最大并发数: %d\n", len(repos), a.maxGoroutines)

	// 创建channel用于传递jobs和results
	jobs := make(chan *domain.Repo, len(repos))
	results := make(chan *domain.Repo, len(repos))
	errors := make(chan error, len(repos))

	// 启动workers
	var wg sync.WaitGroup
	for i := 0; i < a.maxGoroutines; i++ {
		wg.Add(1)
		go a.analyzeRepoWorker(ctx, jobs, results, errors, &wg, i+1)
	}

	// 发送jobs
	for _, repo := range repos {
		jobs <- repo
	}
	close(jobs)

	// 等待所有workers完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 所有任务完成
	case <-ctx.Done():
		// 上下文超时或取消
		fmt.Println("⏰ LLM分析因超时或取消而中断")
		return repos, ctx.Err()
	}

	// 关闭channels
	close(results)
	close(errors)

	// 收集结果
	analyzedRepos := make([]*domain.Repo, 0, len(repos))
	for result := range results {
		analyzedRepos = append(analyzedRepos, result)
	}

	// 打印错误信息（如果有）
	if len(errors) > 0 {
		fmt.Printf("⚠️  共有 %d 个分析错误:\n", len(errors))
		for err := range errors {
			fmt.Printf("   错误: %v\n", err)
		}
	}

	fmt.Println("✅ LLM分析完成")
	return analyzedRepos, nil
}
