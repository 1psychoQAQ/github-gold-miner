package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type MockScouter struct {
	mock.Mock
}

func (m *MockScouter) GetTrendingRepos(ctx context.Context, language string, since string) ([]*domain.Repo, error) {
	args := m.Called(ctx, language, since)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

func (m *MockScouter) GetReposByTopic(ctx context.Context, topic string) ([]*domain.Repo, error) {
	args := m.Called(ctx, topic)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

type MockFilter struct {
	mock.Mock
}

func (m *MockFilter) FilterByCreatedAt(repos []*domain.Repo, maxDaysOld int) []*domain.Repo {
	args := m.Called(repos, maxDaysOld)
	return args.Get(0).([]*domain.Repo)
}

func (m *MockFilter) FilterByRecentCommit(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error) {
	args := m.Called(ctx, repos)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

type MockAnalyzer struct {
	mock.Mock
}

func (m *MockAnalyzer) CalculateStarGrowthRate(repos []*domain.Repo) []*domain.Repo {
	args := m.Called(repos)
	return args.Get(0).([]*domain.Repo)
}

// 添加缺失的AnalyzeWithLLM方法
func (m *MockAnalyzer) AnalyzeWithLLM(ctx context.Context, repos []*domain.Repo) ([]*domain.Repo, error) {
	args := m.Called(ctx, repos)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

// 添加SetMaxGoroutines方法
func (m *MockAnalyzer) SetMaxGoroutines(max int) {
	m.Called(max)
}

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

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Save(ctx context.Context, repo *domain.Repo) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockRepository) Exists(ctx context.Context, repoID string) (bool, error) {
	args := m.Called(ctx, repoID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) MarkAsNotified(ctx context.Context, repoID string) error {
	args := m.Called(ctx, repoID)
	return args.Error(0)
}

func (m *MockRepository) Search(ctx context.Context, query string) ([]*domain.Repo, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

func (m *MockRepository) GetAllCandidates(ctx context.Context) ([]*domain.Repo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

func (m *MockRepository) GetUnnotifiedRepos(ctx context.Context) ([]*domain.Repo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Repo), args.Error(1)
}

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) Notify(ctx context.Context, repo *domain.Repo) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func TestNewMiningService(t *testing.T) {
	mockScouter := new(MockScouter)
	mockFilter := new(MockFilter)
	mockAnalyzer := new(MockAnalyzer)
	mockRepository := new(MockRepository)
	mockAppraiser := new(MockAppraiser)
	mockNotifier := new(MockNotifier)

	service := NewMiningService(
		mockScouter,
		mockFilter,
		mockAnalyzer,
		mockRepository,
		mockAppraiser,
		mockNotifier,
	)

	assert.NotNil(t, service)
	assert.Equal(t, mockScouter, service.fetcher)
	assert.Equal(t, mockFilter, service.filter)
	assert.Equal(t, mockAnalyzer, service.analyzer)
	assert.Equal(t, mockRepository, service.repoStore)
	assert.Equal(t, mockAppraiser, service.appraiser)
	assert.Equal(t, mockNotifier, service.notifier)
}

// 表驱动测试用例
func TestMiningService_ExecuteMiningCycle(t *testing.T) {
	// 准备测试数据
	testRepo := &domain.Repo{
		ID:                  "test/repo",
		Name:                "test/repo",
		Description:         "Test repository",
		URL:                 "https://github.com/test/repo",
		Stars:               100,
		CreatedAt:           time.Now().AddDate(0, 0, -5), // 5天前创建
		UpdatedAt:           time.Now(),
		IsAIProgrammingTool: true,
		LLMScore:            80,
		LLMReview:           "Good AI programming tool",
		StarGrowthRate:      20.0,
	}

	tests := []struct {
		name        string
		setupMocks  func(*MockScouter, *MockFilter, *MockAnalyzer, *MockRepository, *MockAppraiser, *MockNotifier)
		expectError bool
	}{
		{
			name: "正常流程",
			setupMocks: func(ms *MockScouter, mf *MockFilter, ma *MockAnalyzer, mr *MockRepository, ma2 *MockAppraiser, notifier *MockNotifier) {
				ms.On("GetTrendingRepos", mock.Anything, "all", "weekly").Return([]*domain.Repo{testRepo}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ai-coding").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ide-extension").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "dev-tools").Return([]*domain.Repo{}, nil)
				mf.On("FilterByCreatedAt", mock.Anything, 10).Return([]*domain.Repo{testRepo})
				mf.On("FilterByRecentCommit", mock.Anything, mock.Anything).Return([]*domain.Repo{testRepo}, nil)
				ma.On("SetMaxGoroutines", mock.Anything).Return()
				ma.On("CalculateStarGrowthRate", mock.Anything).Return([]*domain.Repo{testRepo})
				// 为内部创建的analyzer设置mock
				ma.On("AnalyzeWithLLM", mock.Anything, mock.Anything).Return([]*domain.Repo{testRepo}, nil)
				mr.On("Exists", mock.Anything, testRepo.ID).Return(false, nil)
				mr.On("Save", mock.Anything, testRepo).Return(nil)
				mr.On("MarkAsNotified", mock.Anything, testRepo.ID).Return(nil)
				notifier.On("Notify", mock.Anything, testRepo).Return(nil)
			},
			expectError: false,
		},
		{
			name: "获取Trending Repos失败",
			setupMocks: func(ms *MockScouter, mf *MockFilter, ma *MockAnalyzer, mr *MockRepository, ma2 *MockAppraiser, notifier *MockNotifier) {
				ms.On("GetTrendingRepos", mock.Anything, "all", "weekly").Return([]*domain.Repo{}, errors.New("network error"))
				ms.On("GetReposByTopic", mock.Anything, "ai-coding").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ide-extension").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "dev-tools").Return([]*domain.Repo{}, nil)
				mf.On("FilterByCreatedAt", mock.Anything, 10).Return([]*domain.Repo{})
				mf.On("FilterByRecentCommit", mock.Anything, mock.Anything).Return([]*domain.Repo{}, nil)
				ma.On("SetMaxGoroutines", mock.Anything).Return()
				ma.On("CalculateStarGrowthRate", mock.Anything).Return([]*domain.Repo{})
				ma.On("AnalyzeWithLLM", mock.Anything, mock.Anything).Return([]*domain.Repo{}, nil)
			},
			expectError: false, // 不应该返回错误，只是记录日志
		},
		{
			name: "活跃度过滤失败",
			setupMocks: func(ms *MockScouter, mf *MockFilter, ma *MockAnalyzer, mr *MockRepository, ma2 *MockAppraiser, notifier *MockNotifier) {
				ms.On("GetTrendingRepos", mock.Anything, "all", "weekly").Return([]*domain.Repo{testRepo}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ai-coding").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ide-extension").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "dev-tools").Return([]*domain.Repo{}, nil)
				mf.On("FilterByCreatedAt", mock.Anything, 10).Return([]*domain.Repo{testRepo})
				mf.On("FilterByRecentCommit", mock.Anything, mock.Anything).Return([]*domain.Repo{}, errors.New("filter error"))
				ma.On("SetMaxGoroutines", mock.Anything).Return()
				ma.On("CalculateStarGrowthRate", mock.Anything).Return([]*domain.Repo{}) // 活跃度过滤失败后，没有项目进入分析阶段
				ma.On("AnalyzeWithLLM", mock.Anything, mock.Anything).Return([]*domain.Repo{}, nil)
				// 注意：活跃度过滤失败后，不会有项目进入存储阶段，所以不需要设置mr的mock
			},
			expectError: false,
		},
		{
			name: "LLM分析失败",
			setupMocks: func(ms *MockScouter, mf *MockFilter, ma *MockAnalyzer, mr *MockRepository, ma2 *MockAppraiser, notifier *MockNotifier) {
				ms.On("GetTrendingRepos", mock.Anything, "all", "weekly").Return([]*domain.Repo{testRepo}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ai-coding").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ide-extension").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "dev-tools").Return([]*domain.Repo{}, nil)
				mf.On("FilterByCreatedAt", mock.Anything, 10).Return([]*domain.Repo{testRepo})
				mf.On("FilterByRecentCommit", mock.Anything, mock.Anything).Return([]*domain.Repo{testRepo}, nil)
				ma.On("SetMaxGoroutines", mock.Anything).Return()
				ma.On("CalculateStarGrowthRate", mock.Anything).Return([]*domain.Repo{testRepo})
				ma.On("AnalyzeWithLLM", mock.Anything, mock.Anything).Return([]*domain.Repo{}, errors.New("LLM error"))
				// 注意：LLM分析失败后，不会有项目进入存储阶段，所以不需要设置mr的mock
			},
			expectError: false,
		},
		{
			name: "活跃度过滤后无项目",
			setupMocks: func(ms *MockScouter, mf *MockFilter, ma *MockAnalyzer, mr *MockRepository, ma2 *MockAppraiser, notifier *MockNotifier) {
				ms.On("GetTrendingRepos", mock.Anything, "all", "weekly").Return([]*domain.Repo{testRepo}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ai-coding").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "ide-extension").Return([]*domain.Repo{}, nil)
				ms.On("GetReposByTopic", mock.Anything, "dev-tools").Return([]*domain.Repo{}, nil)
				mf.On("FilterByCreatedAt", mock.Anything, 10).Return([]*domain.Repo{testRepo})
				mf.On("FilterByRecentCommit", mock.Anything, mock.Anything).Return([]*domain.Repo{}, nil) // 过滤后无项目
				ma.On("SetMaxGoroutines", mock.Anything).Return()
				ma.On("CalculateStarGrowthRate", mock.Anything).Return([]*domain.Repo{}) // 无项目
				ma.On("AnalyzeWithLLM", mock.Anything, mock.Anything).Return([]*domain.Repo{}, nil)
				// 注意：没有项目需要存储，所以不需要设置mr的mock
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScouter := new(MockScouter)
			mockFilter := new(MockFilter)
			mockAnalyzer := new(MockAnalyzer)
			mockRepository := new(MockRepository)
			mockAppraiser := new(MockAppraiser)
			mockNotifier := new(MockNotifier)

			// 设置mock
			tt := tt
			tt.setupMocks(mockScouter, mockFilter, mockAnalyzer, mockRepository, mockAppraiser, mockNotifier)

			service := NewMiningService(
				mockScouter,
				mockFilter,
				mockAnalyzer,
				mockRepository,
				mockAppraiser,
				mockNotifier,
			)

			ctx := context.Background()
			
			// 执行测试
			err := service.ExecuteMiningCycle(ctx, 3)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 验证mock被正确调用
			mockScouter.AssertExpectations(t)
			mockFilter.AssertExpectations(t)
			mockAnalyzer.AssertExpectations(t)
			mockRepository.AssertExpectations(t)
			mockAppraiser.AssertExpectations(t)
			mockNotifier.AssertExpectations(t)
		})
	}
}
