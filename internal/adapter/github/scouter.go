package github

import (
	"context"
	"fmt"
	"time"

	"github-gold-miner/internal/domain"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// Scouter 实现了 port.Scouter 接口
type Scouter struct {
	client *github.Client
}

// NewScouter 初始化 GitHub 客户端
// token: GitHub Personal Access Token (如果是空字符串，就是匿名访问，限制 60次/小时)
func NewScouter(token string) *Scouter {
	var client *github.Client

	if token == "" {
		client = github.NewClient(nil)
	} else {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}

	return &Scouter{client: client}
}

// Scout 搜索最近热门的项目
func (s *Scouter) Scout(ctx context.Context, lang string) ([]*domain.Repo, error) {
	// 1. 构造查询条件
	// 策略：搜索最近 7 天创建的，Star 数大于 50 的项目
	// 这样能过滤掉很多老项目，专注于“新金矿”
	sevenDaysAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	query := fmt.Sprintf("language:%s created:>%s stars:>50", lang, sevenDaysAgo)

	// 2. 调用 Search API
	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 10, // MVP 每次只抓前 10 个，节省 AI Token
		},
	}

	result, _, err := s.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("GitHub API 调用失败: %w", err)
	}

	// 3. 将 GitHub 的数据结构转换为我们的 Domain 实体 (DTO 转换)
	var repos []*domain.Repo
	for _, item := range result.Repositories {
		repo := &domain.Repo{
			ID:          fmt.Sprintf("github-%d", item.GetID()), // 加上前缀防止冲突
			Name:        item.GetFullName(),
			URL:         item.GetHTMLURL(),
			Description: item.GetDescription(),
			Stars:       item.GetStargazersCount(),
			Language:    item.GetLanguage(),
			UpdatedAt:   item.GetUpdatedAt().Time,
			// 下面的字段留给 AI 填
			CommercialScore:  0,
			EducationalScore: 0,
		}
		repos = append(repos, repo)
	}

	return repos, nil
}
