package analyzer

import (
	"context"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAppraiser 是一个模拟的Appraiser实现
type MockAppraiser struct {
	mock.Mock
}

func (m *MockAppraiser) Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).(*domain.Repo), args.Error(1)
}

func (m *MockAppraiser) SemanticSearch(ctx context.Context, repos []*domain.Repo, userQuery string) (string, error) {
	args := m.Called(ctx, repos, userQuery)
	return args.String(0), args.Error(1)
}

func TestRepoAnalyzer_CalculateStarGrowthRate(t *testing.T) {
	analyzer := &RepoAnalyzer{}

	now := time.Now()
	repos := []*domain.Repo{
		{
			ID:        "1",
			Name:      "repo1",
			Stars:     100,
			CreatedAt: now.AddDate(0, 0, -10), // 10天前创建
		},
		{
			ID:        "2",
			Name:      "repo2",
			Stars:     50,
			CreatedAt: now.AddDate(0, 0, -5), // 5天前创建
		},
	}

	result := analyzer.CalculateStarGrowthRate(repos)

	assert.Equal(t, 2, len(result))
	assert.InDelta(t, 10.0, result[0].StarGrowthRate, 0.1) // 100 stars / 10 days
	assert.InDelta(t, 10.0, result[1].StarGrowthRate, 0.1) // 50 stars / 5 days
}

func TestRepoAnalyzer_AnalyzeWithLLM(t *testing.T) {
	mockAppraiser := new(MockAppraiser)
	analyzer := NewRepoAnalyzer(mockAppraiser)

	ctx := context.Background()
	repos := []*domain.Repo{
		{
			ID:   "1",
			Name: "test-repo",
		},
	}

	// 设置mock期望
	mockAppraiser.On("Appraise", ctx, repos[0]).Return(&domain.Repo{
		ID:                  "1",
		Name:                "test-repo",
		IsAIProgrammingTool: true,
		LLMScore:            80,
		LLMReview:           "Good AI tool",
	}, nil)

	result, err := analyzer.AnalyzeWithLLM(ctx, repos)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.True(t, result[0].IsAIProgrammingTool)
	assert.Equal(t, 80, result[0].LLMScore)
	assert.Equal(t, "Good AI tool", result[0].LLMReview)

	// 验证mock被正确调用
	mockAppraiser.AssertExpectations(t)
}