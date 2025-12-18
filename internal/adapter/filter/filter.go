package filter

import (
	"context"
	"fmt"
	"time"

	"github-gold-miner/internal/domain"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// RepoFilter 实现了 port.Filter 接口
type RepoFilter struct {
	client  *github.Client
	nowFunc func() time.Time
}

// NewRepoFilter 创建新的过滤器实例
func NewRepoFilter(token string) *RepoFilter {
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

	return &RepoFilter{
		client:  client,
		nowFunc: time.Now,
	}
}

// FilterByCreatedAt 过滤掉创建时间超过指定天数的项目
func (f *RepoFilter) FilterByCreatedAt(repos []*domain.Repo, maxDaysOld int) []*domain.Repo {
	var filtered []*domain.Repo
	maxAge := time.Duration(maxDaysOld) * 24 * time.Hour
	current := time.Now()
	if f != nil && f.nowFunc != nil {
		current = f.nowFunc()
	}

	for _, repo := range repos {
		if current.Sub(repo.CreatedAt) <= maxAge {
			filtered = append(filtered, repo)
		}
	}

	return filtered
}

// FilterByRecentCommit 过滤掉没有近期提交的项目
func (f *RepoFilter) FilterByRecentCommit(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error) {
	if f.client == nil {
		// 没有GitHub客户端时无法检查提交，保守地返回原列表
		cloned := make([]*domain.Repo, len(repos))
		copy(cloned, repos)
		return cloned, nil
	}

	var filtered []*domain.Repo

	for _, repo := range repos {
		// 从repo URL中提取owner和repo name
		// URL格式: https://github.com/owner/repo
		var owner, repoName string
		_, err := fmt.Sscanf(repo.URL, "https://github.com/%s/%s", &owner, &repoName)
		if err != nil {
			// 如果无法解析URL，保留该项目以防万一
			filtered = append(filtered, repo)
			continue
		}

		// 获取默认分支的最新提交
		commits, _, err := f.client.Repositories.ListCommits(ctx, owner, repoName, &github.CommitsListOptions{
			ListOptions: github.ListOptions{PerPage: 1},
		})

		if err != nil {
			// 如果获取提交信息失败，保留该项目以防万一
			filtered = append(filtered, repo)
			continue
		}

		// 检查是否有提交且提交时间在近期内
		if len(commits) > 0 && commits[0].Commit != nil && commits[0].Commit.Committer != nil {
			// 如果有最近提交，则保留该项目
			filtered = append(filtered, repo)
		}
		// 否则过滤掉该项目
	}

	if len(filtered) == 0 {
		return []*domain.Repo{}, nil
	}

	return filtered, nil
}
