package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github-gold-miner/internal/domain"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiAppraiser struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// 定义一个内部结构体来接收 AI 返回的 JSON
type aiResponse struct {
	CommercialScore  int    `json:"commercial_score"`
	EducationalScore int    `json:"educational_score"`
	Summary          string `json:"summary"`
	DeepAnalysis     string `json:"deep_analysis"`
}

func NewGeminiAppraiser(ctx context.Context, apiKey string) (*GeminiAppraiser, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-2.5-flash-lite")
	// 强制要求返回 JSON，降低解析错误的概率
	model.ResponseMIMEType = "application/json"

	return &GeminiAppraiser{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiAppraiser) Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error) {
	// 1. 构造 Prompt
	prompt := fmt.Sprintf(`
你是一个精通 Golang 的技术专家和极具商业嗅觉的投资人。请分析以下开源项目：

项目名称: %s
项目地址: %s
项目描述: %s

请严格按照 JSON 格式返回分析结果，包含以下字段：
1. commercial_score (0-100): 商业变现潜力。
2. educational_score (0-100): 学习价值。
3. summary: 一句话的中文简评。
4. deep_analysis: 详细的中文分析。

请直接返回 JSON，不要包含 Markdown 格式标记。
`, repo.Name, repo.URL, repo.Description)

	// 2. 调用 AI (增加重试或错误处理)
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// 即使 AI 挂了，也要返回 repo，防止 main.go 崩溃
		return repo, fmt.Errorf("AI 调用失败: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return repo, fmt.Errorf("AI 返回内容为空")
	}

	// 3. 解析结果 (智能清洗逻辑)
	part := resp.Candidates[0].Content.Parts[0]
	jsonStr, ok := part.(genai.Text)
	if !ok {
		return repo, fmt.Errorf("AI 返回格式错误")
	}

	rawContent := string(jsonStr)

	// 【核心修复】智能寻找 JSON 的起止位置
	// 即使 AI 返回 "```json { ... } \n ```"，我们也能精准抠出中间的 { ... }
	start := strings.Index(rawContent, "{")
	end := strings.LastIndex(rawContent, "}")

	if start == -1 || end == -1 || end <= start {
		// 如果找不到花括号，说明 AI 没返回 JSON
		return repo, fmt.Errorf("无法提取 JSON, AI 原文: %s", rawContent)
	}

	// 截取合法的 JSON 部分
	cleanJson := rawContent[start : end+1]

	var res aiResponse
	if err := json.Unmarshal([]byte(cleanJson), &res); err != nil {
		// 这就是你刚才发给我的那行代码
		return repo, fmt.Errorf("JSON 解析失败: %s | 原文: %s", err, cleanJson)
	}

	// 4. 回填数据
	repo.CommercialScore = res.CommercialScore
	repo.EducationalScore = res.EducationalScore
	repo.Summary = res.Summary
	repo.DeepAnalysis = res.DeepAnalysis

	return repo, nil
}
