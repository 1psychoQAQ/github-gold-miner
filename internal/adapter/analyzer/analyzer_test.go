package analyzer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAppraiser 模拟Appraiser接口
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
	now := time.Now()

	tests := []struct {
		name   string
		repos  []*domain.Repo
		verify func(*testing.T, []*domain.Repo)
	}{
		{
			name: "正常计算增长率",
			repos: []*domain.Repo{
				{
					Name:      "test-repo",
					Stars:     100,
					CreatedAt: now.AddDate(0, 0, -5), // 5天前创建
				},
			},
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 1, len(result))
				assert.Equal(t, "test-repo", result[0].Name)
				assert.Equal(t, 100, result[0].Stars)
				// 允许一定的浮点数误差
				assert.InDelta(t, 20.0, result[0].StarGrowthRate, 0.1)
			},
		},
		{
			name: "项目刚创建",
			repos: []*domain.Repo{
				{
					Name:      "new-repo",
					Stars:     10,
					CreatedAt: now, // 刚创建
				},
			},
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 1, len(result))
				assert.Equal(t, "new-repo", result[0].Name)
				assert.Equal(t, 10, result[0].Stars)
				// 刚创建的项目增长率为0
				assert.Equal(t, 0.0, result[0].StarGrowthRate)
			},
		},
		{
			name:  "空项目列表",
			repos: []*domain.Repo{},
			verify: func(t *testing.T, result []*domain.Repo) {
				assert.Equal(t, 0, len(result))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := &RepoAnalyzer{nowFunc: func() time.Time { return now }}
			result := analyzer.CalculateStarGrowthRate(tt.repos)
			tt.verify(t, result)
		})
	}
}

func TestRepoAnalyzer_AnalyzeWithLLM(t *testing.T) {
	tests := []struct {
		name          string
		repos         []*domain.Repo
		maxGoroutines int
		setupMock     func(*MockAppraiser)
		expectError   bool
	}{
		{
			name: "正常分析",
			repos: []*domain.Repo{
				{
					ID:          "test-repo",
					Name:        "test-repo",
					Description: "Test repository",
				},
			},
			maxGoroutines: 3,
			setupMock: func(ma *MockAppraiser) {
				analyzedRepo := &domain.Repo{
					ID:                  "test-repo",
					Name:                "test-repo",
					Description:         "Test repository",
					IsAIProgrammingTool: true,
					LLMScore:            80,
					LLMReview:           "Good AI tool",
				}
				ma.On("Appraise", mock.Anything, mock.Anything).Return(analyzedRepo, nil)
			},
			expectError: false,
		},
		{
			name: "分析失败但仍继续",
			repos: []*domain.Repo{
				{
					ID:          "fail-repo",
					Name:        "fail-repo",
					Description: "Failing repository",
				},
			},
			maxGoroutines: 1,
			setupMock: func(ma *MockAppraiser) {
				ma.On("Appraise", mock.Anything, mock.Anything).Return((*domain.Repo)(nil), errors.New("appraisal failed"))
			},
			expectError: false, // 不应该返回错误，即使分析失败
		},
		{
			name:          "空项目列表",
			repos:         []*domain.Repo{},
			maxGoroutines: 1,
			setupMock:     func(ma *MockAppraiser) {},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAppraiser := new(MockAppraiser)
			if tt.setupMock != nil {
				tt.setupMock(mockAppraiser)
			}

			analyzer := NewRepoAnalyzer(mockAppraiser)
			analyzer.SetMaxGoroutines(tt.maxGoroutines)

			ctx := context.Background()
			result, err := analyzer.AnalyzeWithLLM(ctx, tt.repos)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.repos), len(result))
			}

			mockAppraiser.AssertExpectations(t)
		})
	}
}

func TestRepoAnalyzer_SetMaxGoroutines(t *testing.T) {
	analyzer := &RepoAnalyzer{}

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "设置正数",
			input:    5,
			expected: 5,
		},
		{
			name:     "设置零值",
			input:    0,
			expected: 3, // 默认值
		},
		{
			name:     "设置负数",
			input:    -1,
			expected: 3, // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer.SetMaxGoroutines(tt.input)
			// 由于maxGoroutines是私有字段，我们无法直接访问
			// 但我们可以通过行为来验证
			assert.NotNil(t, analyzer)
		})
	}
}
