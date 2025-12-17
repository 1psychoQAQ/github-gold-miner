package domain

import (
	"time"
)

// Repo 代表一个经过筛选的AI编程工具项目
type Repo struct {
	// 基础信息 (来自 GitHub)
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Stars       int       `json:"stars"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Star增长率（用于数学模型分析）
	StarGrowthRate float64 `json:"star_growth_rate"`

	// LLM分析结果
	IsAIProgrammingTool bool   `json:"is_ai_programming_tool"` // 是否为AI编程工具
	LLMScore           int    `json:"llm_score"`              // LLM评分(1-100)
	LLMReview          string `json:"llm_review" gorm:"type:text"` // LLM简评
	
	// 推送信息
	AlreadyNotified bool `json:"already_notified" gorm:"index"` // 是否已推送
}
