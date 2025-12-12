package repository

import (
	"context"
	"fmt"

	"github-gold-miner/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresRepo 实现了 port.Repository 接口
type PostgresRepo struct {
	db *gorm.DB
}

// NewPostgresRepo 初始化数据库连接并自动迁移表结构
func NewPostgresRepo(dsn string) (*PostgresRepo, error) {
	// 1. 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 2. 自动迁移 (Auto Migrate) - 这一步太省事了！
	// 它会自动在数据库里创建 repos 表，如果字段变了也会自动更新
	err = db.AutoMigrate(&domain.Repo{})
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return &PostgresRepo{db: db}, nil
}

// Save 保存或更新项目
func (r *PostgresRepo) Save(ctx context.Context, repo *domain.Repo) error {
	// Save 会自动处理 Insert 或 Update (Upsert)
	result := r.db.WithContext(ctx).Save(repo)
	return result.Error
}

// Exists 检查项目是否存在
func (r *PostgresRepo) Exists(ctx context.Context, repoID string) (bool, error) {
	var count int64
	// SELECT count(*) FROM repos WHERE id = ?
	err := r.db.WithContext(ctx).Model(&domain.Repo{}).Where("id = ?", repoID).Count(&count).Error
	return count > 0, err
}

// Search 根据关键词搜索 (对应你的提问查询需求)
func (r *PostgresRepo) Search(ctx context.Context, query string) ([]*domain.Repo, error) {
	var repos []*domain.Repo
	// MVP 简单粗暴：使用 LIKE 模糊查询
	// 搜索 名字、描述 或 AI 分析内容
	likeQuery := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("name LIKE ? OR description LIKE ? OR deep_analysis LIKE ?", likeQuery, likeQuery, likeQuery).
		Order("commercial_score DESC, educational_score DESC"). // 优先展示高价值项目
		Limit(10).                                              // 只返回前10条
		Find(&repos).Error

	return repos, err
}
