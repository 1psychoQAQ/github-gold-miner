package filter

import (
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRepoFilter_FilterByCreatedAt(t *testing.T) {
	filter := &RepoFilter{}

	now := time.Now()
	repos := []*domain.Repo{
		{
			ID:        "1",
			Name:      "repo1",
			CreatedAt: now.AddDate(0, 0, -5), // 5天前创建
		},
		{
			ID:        "2",
			Name:      "repo2",
			CreatedAt: now.AddDate(0, 0, -15), // 15天前创建
		},
		{
			ID:        "3",
			Name:      "repo3",
			CreatedAt: now.AddDate(0, 0, -8), // 8天前创建
		},
	}

	// 过滤10天内创建的项目
	filtered := filter.FilterByCreatedAt(repos, 10)

	assert.Equal(t, 2, len(filtered))
	assert.Equal(t, "1", filtered[0].ID)
	assert.Equal(t, "3", filtered[1].ID)
}

func TestRepoFilter_FilterByRecentCommit(t *testing.T) {
	// 这个测试比较难模拟，因为我们依赖外部GitHub API
	// 在实际应用中，我们会使用mock来模拟GitHub客户端行为
	// 这里只是示例框架
	filter := &RepoFilter{}
	
	repos := []*domain.Repo{
		{
			ID:  "1",
			URL: "https://github.com/test/repo1",
		},
	}
	
	// 由于需要真实的GitHub API调用，我们在这里只是演示测试结构
	// 在实际测试中，我们会mock GitHub客户端
	filtered, err := filter.FilterByRecentCommit(nil, repos)
	
	// 在实际测试中，我们会做更具体的断言
	assert.NoError(t, err)
	assert.NotNil(t, filtered)
}