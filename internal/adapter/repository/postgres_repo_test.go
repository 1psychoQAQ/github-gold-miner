package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupMockDB 创建一个模拟的数据库连接
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	// 创建 SQL mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	// 创建 GORM 数据库实例，禁用日志以减少输出
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return gormDB, mock, cleanup
}

func TestPostgresRepo_Save(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		repo        *domain.Repo
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "成功保存新项目",
			repo: &domain.Repo{
				ID:              "github-123",
				Name:            "test/awesome-tool",
				URL:             "https://github.com/test/awesome-tool",
				Description:     "An awesome tool",
				Stars:           500,
				Language:        "Go",
				CreatedAt:       now,
				UpdatedAt:       now,
				StarGrowthRate:  50.0,
				IsAIProgrammingTool: true,
				LLMScore:        85,
				LLMReview:       "Excellent tool",
				AlreadyNotified: false,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// GORM Save uses UPDATE with primary key condition
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "repos"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name: "更新已存在的项目",
			repo: &domain.Repo{
				ID:              "github-456",
				Name:            "test/existing-tool",
				URL:             "https://github.com/test/existing-tool",
				Description:     "Updated description",
				Stars:           1000,
				Language:        "Python",
				CreatedAt:       now,
				UpdatedAt:       now,
				StarGrowthRate:  100.0,
				IsAIProgrammingTool: true,
				LLMScore:        90,
				LLMReview:       "Outstanding",
				AlreadyNotified: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "repos"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			err := repo.Save(ctx, tt.repo)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresRepo_Exists(t *testing.T) {
	tests := []struct {
		name        string
		repoID      string
		setupMock   func(sqlmock.Sqlmock)
		expectExists bool
		expectError bool
	}{
		{
			name:   "项目存在",
			repoID: "github-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectExists: true,
			expectError:  false,
		},
		{
			name:   "项目不存在",
			repoID: "github-999",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectExists: false,
			expectError:  false,
		},
		{
			name:   "数据库错误",
			repoID: "github-error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "repos"`)).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectExists: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			exists, err := repo.Exists(ctx, tt.repoID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectExists, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresRepo_MarkAsNotified(t *testing.T) {
	tests := []struct {
		name        string
		repoID      string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name:   "成功标记为已通知",
			repoID: "github-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// GORM also updates updated_at automatically
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "repos"`)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "更新不存在的项目",
			repoID: "github-999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "repos"`)).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:   "数据库错误",
			repoID: "github-error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE "repos"`)).
					WillReturnError(gorm.ErrInvalidDB)
				mock.ExpectRollback()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			err := repo.MarkAsNotified(ctx, tt.repoID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresRepo_Search(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		query       string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		verify      func(*testing.T, []*domain.Repo)
	}{
		{
			name:  "成功搜索项目",
			query: "AI coding",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				}).
					AddRow(
						"github-1", "ai/coder", "https://github.com/ai/coder",
						"AI coding assistant", 500, "Python",
						now, now, 50.0,
						true, 85, "Great tool", false,
					).
					AddRow(
						"github-2", "ai/copilot", "https://github.com/ai/copilot",
						"AI pair programmer", 1000, "TypeScript",
						now, now, 100.0,
						true, 90, "Excellent", false,
					)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 2, len(repos))
				if len(repos) >= 1 {
					assert.Equal(t, "github-1", repos[0].ID)
					assert.Equal(t, "ai/coder", repos[0].Name)
					assert.Equal(t, 85, repos[0].LLMScore)
				}
			},
		},
		{
			name:  "搜索无结果",
			query: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 0, len(repos))
			},
		},
		{
			name:  "数据库错误",
			query: "error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectError: true,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Nil(t, repos)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			repos, err := repo.Search(ctx, tt.query)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.verify(t, repos)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresRepo_GetAllCandidates(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		verify      func(*testing.T, []*domain.Repo)
	}{
		{
			name: "成功获取候选项目",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				}).
					AddRow(
						"github-1", "test/repo1", "https://github.com/test/repo1",
						"Test repo 1", 100, "Go",
						now, now, 10.0,
						true, 70, "Good", false,
					).
					AddRow(
						"github-2", "test/repo2", "https://github.com/test/repo2",
						"Test repo 2", 200, "Python",
						now.AddDate(0, 0, -1), now, 20.0,
						true, 80, "Great", false,
					)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 2, len(repos))
				if len(repos) >= 2 {
					assert.Equal(t, "github-1", repos[0].ID)
					assert.Equal(t, "github-2", repos[1].ID)
				}
			},
		},
		{
			name: "空结果",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 0, len(repos))
			},
		},
		{
			name: "数据库错误",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectError: true,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Nil(t, repos)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			repos, err := repo.GetAllCandidates(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.verify(t, repos)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresRepo_GetUnnotifiedRepos(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		verify      func(*testing.T, []*domain.Repo)
	}{
		{
			name: "成功获取未通知的项目",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				}).
					AddRow(
						"github-1", "test/unnotified1", "https://github.com/test/unnotified1",
						"Unnotified repo 1", 500, "Go",
						now, now, 50.0,
						true, 90, "Excellent", false,
					).
					AddRow(
						"github-2", "test/unnotified2", "https://github.com/test/unnotified2",
						"Unnotified repo 2", 300, "Python",
						now, now, 30.0,
						true, 80, "Great", false,
					)

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 2, len(repos))
				if len(repos) >= 2 {
					assert.Equal(t, "github-1", repos[0].ID)
					assert.Equal(t, 90, repos[0].LLMScore)
					assert.False(t, repos[0].AlreadyNotified)
					assert.Equal(t, "github-2", repos[1].ID)
					assert.Equal(t, 80, repos[1].LLMScore)
				}
			},
		},
		{
			name: "所有项目已通知",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "url", "description", "stars", "language",
					"created_at", "updated_at", "star_growth_rate",
					"is_ai_programming_tool", "llm_score", "llm_review", "already_notified",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnRows(rows)
			},
			expectError: false,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Equal(t, 0, len(repos))
			},
		},
		{
			name: "数据库错误",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "repos"`)).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectError: true,
			verify: func(t *testing.T, repos []*domain.Repo) {
				assert.Nil(t, repos)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock, cleanup := setupMockDB(t)
			defer cleanup()

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			repo := &PostgresRepo{db: gormDB}
			ctx := context.Background()

			repos, err := repo.GetUnnotifiedRepos(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.verify(t, repos)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNewPostgresRepo_ConnectionError(t *testing.T) {
	// 测试无效的连接字符串
	invalidDSN := "invalid-connection-string"

	repo, err := NewPostgresRepo(invalidDSN)

	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.Contains(t, err.Error(), "连接数据库失败")
}

func TestPostgresRepo_ContextCancellation(t *testing.T) {
	gormDB, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := &PostgresRepo{db: gormDB}

	// 使用正常的上下文，但让数据库返回上下文取消错误
	ctx := context.Background()

	// 测试 Exists 方法的上下文取消
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "repos"`)).
		WillReturnError(context.Canceled)

	exists, err := repo.Exists(ctx, "github-123")

	assert.Error(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}
