package filter

import (
	"context"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestRepoFilter_FilterByCreatedAt(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		repos      []*domain.Repo
		maxDaysOld int
		verify     func(*testing.T, []*domain.Repo)
	}{
		{
			name: "过滤掉老项目",
			repos: []*domain.Repo{
				{
					Name:      "new-repo",
					CreatedAt: now.AddDate(0, 0, -5), // 5天前
				},
				{
					Name:      "old-repo",
					CreatedAt: now.AddDate(0, 0, -15), // 15天前
				},
			},
			maxDaysOld: 10,
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 1, len(result))
				assert.Equal(t, "new-repo", result[0].Name)
			},
		},
		{
			name: "保留边界项目",
			repos: []*domain.Repo{
				{
					Name:      "boundary-repo",
					CreatedAt: now.AddDate(0, 0, -10), // 正好10天前
				},
			},
			maxDaysOld: 10,
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 1, len(result))
				assert.Equal(t, "boundary-repo", result[0].Name)
			},
		},
		{
			name:       "空列表",
			repos:      []*domain.Repo{},
			maxDaysOld: 10,
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 0, len(result))
			},
		},
		{
			name: "所有项目都太老",
			repos: []*domain.Repo{
				{
					Name:      "very-old-repo",
					CreatedAt: now.AddDate(0, 0, -20),
				},
			},
			maxDaysOld: 10,
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 0, len(result))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &RepoFilter{nowFunc: func() time.Time { return now }}
			result := filter.FilterByCreatedAt(tt.repos, tt.maxDaysOld)
			tt := tt
			tt.verify(t, result)
		})
	}
}

func TestRepoFilter_FilterByRecentCommit(t *testing.T) {
	tests := []struct {
		name   string
		repos  []*domain.Repo
		verify func(*testing.T, []*domain.Repo, error)
	}{
		{
			name: "正常处理项目",
			repos: []*domain.Repo{
				{
					Name: "valid-repo",
					URL:  "https://github.com/owner/repo",
				},
			},
			verify: func(t *testing.T, result []*domain.Repo, err error) {
				assert.NoError(t, err)
				// 由于我们没有设置真实的GitHub客户端，所以结果可能为空
				// 但我们主要关心的是不返回错误
				assert.NotNil(t, result)
			},
		},
		{
			name: "无效URL格式",
			repos: []*domain.Repo{
				{
					Name: "invalid-repo",
					URL:  "not-a-github-url",
				},
			},
			verify: func(t *testing.T, result []*domain.Repo, err error) {
				assert.NoError(t, err)
				// 不应该返回错误，即使URL无效
				assert.NotNil(t, result)
			},
		},
		{
			name:  "空列表",
			repos: []*domain.Repo{},
			verify: func(t *testing.T, result []*domain.Repo, err error) {
				assert.NoError(t, err)
				// 空列表应该返回空结果而不是nil
				if result == nil {
					assert.Fail(t, "result should not be nil")
				} else {
					assert.Equal(t, 0, len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &RepoFilter{}
			ctx := context.Background()

			result, err := filter.FilterByRecentCommit(ctx, tt.repos)
			tt.verify(t, result, err)
		})
	}
}
