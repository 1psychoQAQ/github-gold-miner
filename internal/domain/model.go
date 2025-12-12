package domain

import "time"

// Repo 代表一个经过筛选的开源项目
type Repo struct {
	// 基础信息 (来自 GitHub)
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name"` // 例如 "gohugoio/hugo"
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Stars       int       `json:"stars"`
	Language    string    `json:"language"` // 主要是 Go
	UpdatedAt   time.Time `json:"updated_at"`

	// --- 核心竞争力：AI 分析维度 ---

	// 商业价值评分 (0-100)：是否适合做 SaaS，是否解决刚需
	CommercialScore int `json:"commercial_score"`

	// 技术学习评分 (0-100)：架构是否优雅，是否用了泛型/并发模式
	EducationalScore int `json:"educational_score"`

	// AI 简评：一句话告诉我有啥用 (用于推送到手机)
	Summary string `json:"summary"`

	// AI 深度分析：详细的商业/技术分析 (用于问答查询)
	DeepAnalysis string `json:"deep_analysis" gorm:"type:text"`
}

// IsGold 判断是否是“金矿” (根据你的标准动态调整)
func (r *Repo) IsGold() bool {
	// 逻辑：商业分高 OR 学习分高
	return r.CommercialScore >= 80 || r.EducationalScore >= 85
}
