package port

import (
	"context"
	"github-gold-miner/internal/domain"
)

// Scouter (侦察兵): 负责去 GitHub 发现新项目
type Scouter interface {
	// 获取GitHub Trending项目
	GetTrendingRepos(ctx context.Context, language string, since string) ([]*domain.Repo, error)

	// 根据Topic获取项目
	GetReposByTopic(ctx context.Context, topic string) ([]*domain.Repo, error)
}

// Filter (过滤器): 负责按规则过滤项目
type Filter interface {
	// 过滤掉创建时间超过指定天数的项目
	FilterByCreatedAt(repos []*domain.Repo, maxDaysOld int) []*domain.Repo

	// 过滤掉没有近期提交的项目
	FilterByRecentCommit(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error)
}

// Analyzer (分析器): 负责调用数学模型和LLM进行分析
type Analyzer interface {
	// 计算Star增长率
	CalculateStarGrowthRate(repos []*domain.Repo) []*domain.Repo

	// 使用LLM分析项目是否为AI编程工具及其评分
	AnalyzeWithLLM(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error)
	
	// 设置并发数
	SetMaxGoroutines(max int)
}

// Appraiser (鉴定师): 负责调用 LLM 进行价值评估
type Appraiser interface {
	// 输入原始项目，输出包含评分和分析的完整项目
	Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error)
	SemanticSearch(ctx context.Context, repos []*domain.Repo, userQuery string) (string, error)
}

// Notifier (信使): 负责推送到手机 (飞书/钉钉)
type Notifier interface {
	// 推送单个"金矿"项目
	Notify(ctx context.Context, repo *domain.Repo) error
}

// Repository (仓库管理员): 负责存储和查询
type Repository interface {
	// 保存项目
	Save(ctx context.Context, repo *domain.Repo) error

	// 判断是否已经处理过 (防重)
	Exists(ctx context.Context, repoID string) (bool, error)

	// 标记为已推送
	MarkAsNotified(ctx context.Context, repoID string) error

	// Search 对应你的"提问查询"功能
	// MVP 阶段：可以是 SQL 的 LIKE 查询
	// 进阶阶段：可以是 Vector 向量搜索
	Search(ctx context.Context, query string) ([]*domain.Repo, error)
	GetAllCandidates(ctx context.Context) ([]*domain.Repo, error)

	// 获取未推送的项目
	GetUnnotifiedRepos(ctx context.Context) ([]*domain.Repo, error)
}