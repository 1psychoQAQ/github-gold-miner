package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/assert"
)

// setupMockGitHubServer 创建一个模拟的 GitHub API 服务器
func setupMockGitHubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Fetcher) {
	server := httptest.NewServer(handler)

	// 创建一个使用测试服务器的客户端
	client := github.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = baseURL

	fetcher := &Fetcher{client: client}
	return server, fetcher
}

// mockSearchResponse 创建模拟的 GitHub 搜索响应
func mockSearchResponse(repos []*github.Repository) *github.RepositoriesSearchResult {
	total := len(repos)
	result := &github.RepositoriesSearchResult{
		Total:        github.Int(total),
		Repositories: repos,
	}
	return result
}

// createMockRepo 创建模拟的 GitHub 仓库对象
func createMockRepo(id int64, fullName, description, language string, stars int, createdAt, updatedAt time.Time) *github.Repository {
	return &github.Repository{
		ID:              github.Int64(id),
		FullName:        github.String(fullName),
		HTMLURL:         github.String("https://github.com/" + fullName),
		Description:     github.String(description),
		StargazersCount: github.Int(stars),
		Language:        github.String(language),
		CreatedAt:       &github.Timestamp{Time: createdAt},
		UpdatedAt:       &github.Timestamp{Time: updatedAt},
	}
}

func TestFetcher_GetTrendingRepos(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		language    string
		since       string
		mockRepos   []*github.Repository
		expectError bool
		verify      func(*testing.T, []*domain.Repo)
	}{
		{
			name:     "成功获取每日趋势项目",
			language: "go",
			since:    "daily",
			mockRepos: []*github.Repository{
				createMockRepo(1, "test/repo1", "Test repo 1", "Go", 100, now.AddDate(0, 0, -1), now),
				createMockRepo(2, "test/repo2", "Test repo 2", "Go", 50, now.AddDate(0, 0, -1), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 2, len(repos))
				assert.Equal(t, "github-1", repos[0].ID)
				assert.Equal(t, "test/repo1", repos[0].Name)
				assert.Equal(t, "https://github.com/test/repo1", repos[0].URL)
				assert.Equal(t, "Test repo 1", repos[0].Description)
				assert.Equal(t, 100, repos[0].Stars)
				assert.Equal(t, "Go", repos[0].Language)
			},
		},
		{
			name:     "成功获取每周趋势项目",
			language: "python",
			since:    "weekly",
			mockRepos: []*github.Repository{
				createMockRepo(3, "test/weekly-repo", "Weekly trending", "Python", 200, now.AddDate(0, 0, -3), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 1, len(repos))
				assert.Equal(t, "github-3", repos[0].ID)
				assert.Equal(t, "test/weekly-repo", repos[0].Name)
				assert.Equal(t, "Python", repos[0].Language)
			},
		},
		{
			name:     "成功获取每月趋势项目",
			language: "typescript",
			since:    "monthly",
			mockRepos: []*github.Repository{
				createMockRepo(4, "test/monthly-repo", "Monthly trending", "TypeScript", 300, now.AddDate(0, 0, -20), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 1, len(repos))
				assert.Equal(t, "TypeScript", repos[0].Language)
			},
		},
		{
			name:     "默认时间范围(weekly)",
			language: "rust",
			since:    "",
			mockRepos: []*github.Repository{
				createMockRepo(5, "test/default-repo", "Default time range", "Rust", 150, now.AddDate(0, 0, -5), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 1, len(repos))
			},
		},
		{
			name:        "空结果",
			language:    "java",
			since:       "daily",
			mockRepos:   []*github.Repository{},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 0, len(repos))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟服务器
			server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
				// 验证请求路径
				assert.Equal(t, "/search/repositories", r.URL.Path)

				// 验证查询参数
				query := r.URL.Query().Get("q")
				assert.Contains(t, query, "language:"+tt.language)

				// 返回模拟响应
				response := mockSearchResponse(tt.mockRepos)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})
			defer server.Close()

			// 执行测试
			ctx := context.Background()
			repos, err := fetcher.GetTrendingRepos(ctx, tt.language, tt.since)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.verify(t, repos)
			}
		})
	}
}

func TestFetcher_GetTrendingRepos_APIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "GitHub API 返回 403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"message": "API rate limit exceeded"}`,
			expectError:    true,
			errorSubstring: "GitHub API 调用失败",
		},
		{
			name:           "GitHub API 返回 500 内部错误",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"message": "Internal server error"}`,
			expectError:    true,
			errorSubstring: "GitHub API 调用失败",
		},
		{
			name:           "GitHub API 返回 404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"message": "Not Found"}`,
			expectError:    true,
			errorSubstring: "GitHub API 调用失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			})
			defer server.Close()

			ctx := context.Background()
			repos, err := fetcher.GetTrendingRepos(ctx, "go", "daily")

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorSubstring)
				assert.Nil(t, repos)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetcher_GetReposByTopic(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		topic       string
		mockRepos   []*github.Repository
		expectError bool
		verify      func(*testing.T, []*domain.Repo)
	}{
		{
			name:  "成功获取ai-coding话题项目",
			topic: "ai-coding",
			mockRepos: []*github.Repository{
				createMockRepo(10, "ai/coder", "AI coding assistant", "Python", 500, now.AddDate(0, 0, -5), now),
				createMockRepo(11, "ai/copilot", "AI pair programmer", "TypeScript", 450, now.AddDate(0, 0, -3), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 2, len(repos))
				assert.Equal(t, "github-10", repos[0].ID)
				assert.Equal(t, "ai/coder", repos[0].Name)
				assert.Equal(t, 500, repos[0].Stars)
				assert.Equal(t, "AI coding assistant", repos[0].Description)
			},
		},
		{
			name:  "成功获取ide-extension话题项目",
			topic: "ide-extension",
			mockRepos: []*github.Repository{
				createMockRepo(20, "ext/vscode-tools", "VSCode extension", "JavaScript", 300, now.AddDate(0, 0, -7), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 1, len(repos))
				assert.Equal(t, "github-20", repos[0].ID)
				assert.Equal(t, "ext/vscode-tools", repos[0].Name)
			},
		},
		{
			name:  "成功获取dev-tools话题项目",
			topic: "dev-tools",
			mockRepos: []*github.Repository{
				createMockRepo(30, "tools/cli", "CLI development tool", "Go", 250, now.AddDate(0, 0, -2), now),
				createMockRepo(31, "tools/debugger", "Advanced debugger", "Rust", 200, now.AddDate(0, 0, -4), now),
				createMockRepo(32, "tools/profiler", "Performance profiler", "C++", 180, now.AddDate(0, 0, -6), now),
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 3, len(repos))
				assert.Equal(t, "github-30", repos[0].ID)
				assert.Equal(t, "github-31", repos[1].ID)
				assert.Equal(t, "github-32", repos[2].ID)
			},
		},
		{
			name:        "话题无匹配项目",
			topic:       "non-existent-topic",
			mockRepos:   []*github.Repository{},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 0, len(repos))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/search/repositories", r.URL.Path)

				// 验证查询参数包含 topic
				query := r.URL.Query().Get("q")
				assert.Contains(t, query, "topic:"+tt.topic)

				response := mockSearchResponse(tt.mockRepos)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})
			defer server.Close()

			ctx := context.Background()
			repos, err := fetcher.GetReposByTopic(ctx, tt.topic)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.verify(t, repos)
			}
		})
	}
}

func TestFetcher_GetReposByTopic_APIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectError    bool
		errorSubstring string
	}{
		{
			name:           "API 速率限制",
			statusCode:     http.StatusForbidden,
			expectError:    true,
			errorSubstring: "GitHub API 调用失败",
		},
		{
			name:           "服务器错误",
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
			errorSubstring: "GitHub API 调用失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"message": "Error"}`))
			})
			defer server.Close()

			ctx := context.Background()
			repos, err := fetcher.GetReposByTopic(ctx, "test-topic")

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorSubstring)
				assert.Nil(t, repos)
			}
		})
	}
}

func TestNewFetcher(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		verify func(*testing.T, *Fetcher)
	}{
		{
			name:  "使用令牌创建",
			token: "ghp_test_token_1234567890",
			verify: func(t *testing.T, f *Fetcher) {
				assert.NotNil(t, f)
				assert.NotNil(t, f.client)
			},
		},
		{
			name:  "无令牌创建",
			token: "",
			verify: func(t *testing.T, f *Fetcher) {
				assert.NotNil(t, f)
				assert.NotNil(t, f.client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewFetcher(tt.token)
			tt.verify(t, fetcher)
		})
	}
}

func TestFetcher_GetTrendingRepos_ContextCancellation(t *testing.T) {
	// 创建一个已取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
		// 不应该到达这里，因为上下文已取消
		t.Fatal("should not reach here due to context cancellation")
	})
	defer server.Close()

	repos, err := fetcher.GetTrendingRepos(ctx, "go", "daily")

	assert.Error(t, err)
	assert.Nil(t, repos)
	assert.Contains(t, err.Error(), "GitHub API 调用失败")
}

func TestFetcher_GetReposByTopic_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server, fetcher := setupMockGitHubServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach here due to context cancellation")
	})
	defer server.Close()

	repos, err := fetcher.GetReposByTopic(ctx, "ai-coding")

	assert.Error(t, err)
	assert.Nil(t, repos)
	assert.Contains(t, err.Error(), "GitHub API 调用失败")
}
