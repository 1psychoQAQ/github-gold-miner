package port

import (
	"context"
	"github-gold-miner/internal/domain"
)

// Scouter (侦察兵): 负责去 GitHub/HackerNews 发现新项目
// 它可以是爬虫，也可以是调 GitHub API
type Scouter interface {
	// 比如：Scout(ctx, "Go", "daily")
	Scout(ctx context.Context, lang string) ([]*domain.Repo, error)
}

// Appraiser (鉴定师): 负责调用 LLM (Gemini) 进行价值评估
type Appraiser interface {
	// 输入原始项目，输出包含评分和分析的完整项目
	Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error)
}

// Notifier (信使): 负责推送到手机 (飞书/钉钉)
type Notifier interface {
	// 推送单个“金矿”项目
	Notify(ctx context.Context, repo *domain.Repo) error
}

// Repository (仓库管理员): 负责存储和查询
// 这里对应你的“接口提问查询”需求
type Repository interface {
	// 保存项目
	Save(ctx context.Context, repo *domain.Repo) error

	// 判断是否已经处理过 (防重)
	Exists(ctx context.Context, repoID string) (bool, error)

	// Search 对应你的“提问查询”功能
	// MVP 阶段：可以是 SQL 的 LIKE 查询
	// 进阶阶段：可以是 Vector 向量搜索
	Search(ctx context.Context, query string) ([]*domain.Repo, error)
}
