package filter

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github-gold-miner/internal/common"
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

// FilterByRecentCommit 过滤掉没有近期提交的项目，以及仅有 README 提交的项目
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
		u, err := url.Parse(repo.URL)
		if err != nil || u.Host != "github.com" {
			// 如果无法解析URL，保留该项目以防万一
			log.Printf("[Filter] 无法解析仓库URL %s: %v", repo.URL, err)
			filtered = append(filtered, repo)
			continue
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			log.Printf("[Filter] 无法解析仓库URL %s: 路径格式不正确", repo.URL)
			filtered = append(filtered, repo)
			continue
		}
		owner, repoName = parts[0], parts[1]

		// 检查是否有实际代码提交(非README-only)
		hasRealCommit, err := f.hasNonReadmeCommit(ctx, owner, repoName)
		if err != nil {
			// API调用失败，保守地保留该项目
			log.Printf("[Filter] 检查仓库 %s/%s 提交时出错: %v，保留该项目", owner, repoName, err)
			filtered = append(filtered, repo)
			continue
		}

		if hasRealCommit {
			filtered = append(filtered, repo)
		} else {
			log.Printf("[Filter] 过滤掉仅有README提交的仓库: %s/%s", owner, repoName)
		}
	}

	if len(filtered) == 0 {
		return []*domain.Repo{}, nil
	}

	return filtered, nil
}

// hasNonReadmeCommit 检查仓库是否有非README相关的提交
// 返回 true 表示有实际代码提交，false 表示只有README提交
func (f *RepoFilter) hasNonReadmeCommit(ctx context.Context, owner, repoName string) (bool, error) {
	const maxCommitsToCheck = 10 // 检查最近10个提交
	const minCommitsThreshold = 2 // 如果少于2个提交，放宽标准

	// 获取最近的提交列表
	commits, _, err := f.client.Repositories.ListCommits(ctx, owner, repoName, &github.CommitsListOptions{
		ListOptions: github.ListOptions{PerPage: maxCommitsToCheck},
	})

	if err != nil {
		return false, fmt.Errorf("获取提交列表失败: %w", err)
	}

	if len(commits) == 0 {
		// 没有任何提交，过滤掉
		return false, nil
	}

	// 如果提交数很少(新项目)，只需要有至少一个非README提交即可
	if len(commits) < minCommitsThreshold {
		for _, commit := range commits {
			hasNonReadme, err := f.commitHasNonReadmeChanges(ctx, owner, repoName, commit.GetSHA())
			if err != nil {
				// 如果API调用失败，保守处理:继续检查下一个
				continue
			}
			if hasNonReadme {
				return true, nil
			}
		}
		// 所有提交都是README
		return false, nil
	}

	// 对于有多个提交的项目，要求至少有一个非README提交
	for _, commit := range commits {
		hasNonReadme, err := f.commitHasNonReadmeChanges(ctx, owner, repoName, commit.GetSHA())
		if err != nil {
			// API调用失败，继续检查下一个
			log.Printf("[Filter] 检查提交 %s 时出错: %v", commit.GetSHA(), err)
			continue
		}

		if hasNonReadme {
			return true, nil
		}
	}

	// 所有提交都只修改了README
	return false, nil
}

// commitHasNonReadmeChanges 检查单个提交是否包含非README文件的修改
func (f *RepoFilter) commitHasNonReadmeChanges(ctx context.Context, owner, repoName, sha string) (bool, error) {
	var commit *github.RepositoryCommit
	var err error

	// 使用重试机制获取提交详情
	retryErr := common.Do(ctx, func() error {
		commit, _, err = f.client.Repositories.GetCommit(ctx, owner, repoName, sha, nil)
		if err != nil {
			return err
		}
		return nil
	}, common.WithMaxRetries(2), common.WithInitialDelay(500*time.Millisecond))

	if retryErr != nil {
		return false, fmt.Errorf("获取提交详情失败 (SHA: %s): %w", sha, retryErr)
	}

	if commit == nil || len(commit.Files) == 0 {
		// 没有文件变更信息，保守地认为有实际提交
		return true, nil
	}

	// 检查是否有非README文件的修改
	for _, file := range commit.Files {
		if file.Filename == nil {
			continue
		}

		filename := *file.Filename
		if !isReadmeFile(filename) {
			// 发现非README文件，返回true
			return true, nil
		}
	}

	// 所有文件都是README相关
	return false, nil
}

// isReadmeFile 判断文件名是否为README相关文件
func isReadmeFile(filename string) bool {
	// 转为小写进行比较
	lower := strings.ToLower(filename)

	// 精确匹配常见的README文件名
	readmePatterns := []string{
		"readme",
		"readme.md",
		"readme.txt",
		"readme.rst",
		"readme.markdown",
		"readme.mdown",
		"readme.mkdn",
	}

	for _, pattern := range readmePatterns {
		if lower == pattern {
			return true
		}
	}

	// 检查是否在子目录中的README(例如 docs/README.md)
	// 但这里我们只关注根目录或简单路径的README
	parts := strings.Split(lower, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		for _, pattern := range readmePatterns {
			if lastPart == pattern {
				return true
			}
		}
	}

	return false
}
