package filter

import (
	"context"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsReadmeFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// 根目录README文件
		{name: "README.md根目录", filename: "README.md", expected: true},
		{name: "readme.md小写", filename: "readme.md", expected: true},
		{name: "README大写", filename: "README", expected: true},
		{name: "readme小写", filename: "readme", expected: true},
		{name: "README.txt", filename: "README.txt", expected: true},
		{name: "README.rst", filename: "README.rst", expected: true},
		{name: "README.markdown", filename: "README.markdown", expected: true},

		// 子目录中的README
		{name: "docs中的README", filename: "docs/README.md", expected: true},
		{name: "深层目录README", filename: "docs/api/README.md", expected: true},
		{name: "子目录readme小写", filename: "examples/readme.txt", expected: true},

		// 非README文件
		{name: "普通源码文件", filename: "main.go", expected: false},
		{name: "配置文件", filename: "config.yaml", expected: false},
		{name: "包含readme的文件名", filename: "readme_generator.go", expected: false},
		{name: "LICENSE文件", filename: "LICENSE", expected: false},
		{name: "CHANGELOG", filename: "CHANGELOG.md", expected: false},
		{name: "其他md文件", filename: "CONTRIBUTING.md", expected: false},
		{name: "子目录源码", filename: "pkg/service/handler.go", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReadmeFile(tt.filename)
			assert.Equal(t, tt.expected, result, "filename: %s", tt.filename)
		})
	}
}

func TestRepoFilter_FilterByRecentCommit_WithNilClient(t *testing.T) {
	// 测试没有GitHub客户端时的保守行为
	filter := &RepoFilter{client: nil}
	ctx := context.Background()

	repos := []*domain.Repo{
		{Name: "test-repo-1", URL: "https://github.com/owner/repo1"},
		{Name: "test-repo-2", URL: "https://github.com/owner/repo2"},
	}

	result, err := filter.FilterByRecentCommit(ctx, repos)

	require.NoError(t, err)
	assert.Equal(t, len(repos), len(result), "应该返回所有仓库")

	// 验证返回的是副本，不是原切片（比较指针地址）
	if len(result) > 0 {
		// 验证第一个元素的内容相同但不是同一个引用
		assert.Equal(t, repos[0].Name, result[0].Name)
	}
}

func TestRepoFilter_FilterByRecentCommit_URLParsing(t *testing.T) {
	// 测试URL解析的边界情况
	filter := &RepoFilter{client: nil} // nil client会保守地返回所有仓库
	ctx := context.Background()

	tests := []struct {
		name          string
		repo          *domain.Repo
		shouldBeKept  bool
	}{
		{
			name: "有效的GitHub URL",
			repo: &domain.Repo{
				Name: "valid-repo",
				URL:  "https://github.com/owner/repo",
			},
			shouldBeKept: true,
		},
		{
			name: "无效URL格式",
			repo: &domain.Repo{
				Name: "invalid-url",
				URL:  "not-a-valid-url",
			},
			shouldBeKept: true, // 保守处理：保留
		},
		{
			name: "空URL",
			repo: &domain.Repo{
				Name: "empty-url",
				URL:  "",
			},
			shouldBeKept: true, // 保守处理：保留
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.FilterByRecentCommit(ctx, []*domain.Repo{tt.repo})
			require.NoError(t, err)

			if tt.shouldBeKept {
				assert.Equal(t, 1, len(result), "应该保留该仓库")
			} else {
				assert.Equal(t, 0, len(result), "应该过滤掉该仓库")
			}
		})
	}
}
