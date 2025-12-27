package github

import (
	"context"
	"fmt"
	"time"

	"github-gold-miner/internal/common"
	"github-gold-miner/internal/domain"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// Fetcher 实现了 port.Scouter 接口
type Fetcher struct {
	client *github.Client
}

// NewFetcher 初始化 GitHub 客户端
func NewFetcher(token string) *Fetcher {
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

	return &Fetcher{client: client}
}

// GetTrendingRepos 获取GitHub Trending项目
// 由于GitHub没有直接的Trending API，我们使用搜索功能按stars排序来模拟
func (f *Fetcher) GetTrendingRepos(ctx context.Context, language string, since string) ([]*domain.Repo, error) {
	// 计算时间范围
	var dateRange string
	switch since {
	case "daily":
		dateRange = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	case "weekly":
		dateRange = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	case "monthly":
		dateRange = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	default:
		dateRange = time.Now().AddDate(0, 0, -7).Format("2006-01-02") // 默认一周
	}

	query := fmt.Sprintf("language:%s created:>%s", language, dateRange)
	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 10, // 前10名
		},
	}

	var result *github.RepositoriesSearchResult
	err := common.Do(ctx, func() error {
		var apiErr error
		result, _, apiErr = f.client.Search.Repositories(ctx, query, opts)
		return apiErr
	},
		common.WithMaxRetries(3),
		common.WithInitialDelay(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("GitHub API 调用失败: %w", err)
	}

	var repos []*domain.Repo
	for _, item := range result.Repositories {
		repo := &domain.Repo{
			ID:          fmt.Sprintf("github-%d", item.GetID()),
			Name:        item.GetFullName(),
			URL:         item.GetHTMLURL(),
			Description: item.GetDescription(),
			Stars:       item.GetStargazersCount(),
			Language:    item.GetLanguage(),
			CreatedAt:   item.GetCreatedAt().Time,
			UpdatedAt:   item.GetUpdatedAt().Time,
		}
		repos = append(repos, repo)
	}

	return repos, nil
}

// GetReposByTopic 根据Topic获取项目
func (f *Fetcher) GetReposByTopic(ctx context.Context, topic string) ([]*domain.Repo, error) {
	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 3, // 前3名
		},
	}

	query := fmt.Sprintf("topic:%s", topic)
	var result *github.RepositoriesSearchResult
	err := common.Do(ctx, func() error {
		var apiErr error
		result, _, apiErr = f.client.Search.Repositories(ctx, query, opts)
		return apiErr
	},
		common.WithMaxRetries(3),
		common.WithInitialDelay(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("GitHub API 调用失败: %w", err)
	}

	var repos []*domain.Repo
	for _, item := range result.Repositories {
		repo := &domain.Repo{
			ID:          fmt.Sprintf("github-%d", item.GetID()),
			Name:        item.GetFullName(),
			URL:         item.GetHTMLURL(),
			Description: item.GetDescription(),
			Stars:       item.GetStargazersCount(),
			Language:    item.GetLanguage(),
			CreatedAt:   item.GetCreatedAt().Time,
			UpdatedAt:   item.GetUpdatedAt().Time,
		}
		repos = append(repos, repo)
	}

	return repos, nil
}