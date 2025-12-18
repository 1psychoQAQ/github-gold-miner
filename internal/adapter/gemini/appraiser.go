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
	model  ContentGenerator // 👈 修改点：这里使用接口类型，而不是具体的结构体指针
}

func NewGeminiAppraiser(ctx context.Context, apiKey string) (*GeminiAppraiser, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-2.5-pro")
	// 强制要求返回 JSON，降低解析错误的概率
	model.ResponseMIMEType = "application/json"

	return &GeminiAppraiser{
		client: client,
		model:  model,
	}, nil
}

// ContentGenerator 定义了我们需要用到的 AI 能力
// 这样我们在测试时就可以用假的实现来替换真的 SDK
type ContentGenerator interface {
	GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}

// 1. 修改接收 AI 结果的结构体 (在 Appraiser 结构体下方)
type aiResponse struct {
	IsAIProgrammingTool bool   `json:"is_ai_programming_tool"`
	LLMScore            int    `json:"llm_score"`
	LLMReview           string `json:"llm_review"`
}

// Appraise 评估项目是否为AI编程工具
func (g *GeminiAppraiser) Appraise(ctx context.Context, repo *domain.Repo) (*domain.Repo, error) {
	// 这是一个极具针对性的 Prompt
	prompt := fmt.Sprintf(`
请分析以下GitHub项目，判断它是否为AI编程工具（如AI代码助手、机器学习库、自然语言处理工具等）。

项目名称：%s
项目描述：%s
项目URL：%s

请严格按照以下JSON格式返回结果（严禁Markdown，必须是纯JSON）：
{
  "is_ai_programming_tool": true/false,
  "llm_score": 1-100的整数分数（如果是AI编程工具则分数较高，否则较低）,
  "llm_review": "简短评价，说明为什么认为它是或不是AI编程工具"
}
`, repo.Name, repo.Description, repo.URL)

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

	// ... 获取到 rawContent 字符串后 ...
	rawContent := string(jsonStr)

	// 👇 修改点：直接调用提取出来的函数
	res, err := parseAIResponse(rawContent)
	if err != nil {
		return repo, fmt.Errorf("解析响应失败: %w | 原文: %s", err, rawContent)
	}

	// 回填数据
	repo.IsAIProgrammingTool = res.IsAIProgrammingTool
	repo.LLMScore = res.LLMScore
	repo.LLMReview = res.LLMReview

	return repo, nil
}

// parseAIResponse 是一个纯函数，专门负责从 AI 的乱七八糟的回复中提取干净的数据
// 我们把它独立出来，就可以专门针对它写测试，而不需要真的去调 Gemini API
func parseAIResponse(rawContent string) (*aiResponse, error) {
	// 1. 智能清洗：只提取 {} 中间的内容
	start := strings.Index(rawContent, "{")
	end := strings.LastIndex(rawContent, "}")

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("无法提取 JSON")
	}

	cleanJson := rawContent[start : end+1]

	// 2. 解析 JSON
	var res aiResponse
	if err := json.Unmarshal([]byte(cleanJson), &res); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	return &res, nil
}

// SemanticSearch 让 AI 根据用户意图，从数据库中筛选项目
func (g *GeminiAppraiser) SemanticSearch(ctx context.Context, repos []*domain.Repo, userQuery string) (string, error) {
	// 1. 数据精简：为了节省 Token，我们只把关键字段喂给 AI
	// 我们创建一个临时的精简结构体，或者直接拼接字符串
	var promptData strings.Builder
	for i, r := range repos {
		promptData.WriteString(fmt.Sprintf("%d. ID: %s | 名称: %s\n", i+1, r.ID, r.Name))
		promptData.WriteString(fmt.Sprintf("   [描述]: %s\n", r.Description))
		promptData.WriteString(fmt.Sprintf("   [LLM评分]: %d\n", r.LLMScore))
		promptData.WriteString(fmt.Sprintf("   [LLM评价]: %s\n", r.LLMReview))
		promptData.WriteString("---\n")
	}

	// 2. 构造"AI 选品"提示词
	prompt := fmt.Sprintf(`
你是一个智能项目库检索助手。你的数据库里有以下 AI 编程工具项目：
%s

用户的搜索请求是："%s"

请根据用户的真实意图，从上述列表中**挑选出最匹配的 1-3 个项目**。

请按以下格式输出分析结果（直接输出文本，不要 JSON）：

### 🎯 最佳匹配：[项目名称]
- **匹配理由**：为什么这个项目符合用户的请求？
- **功能简介**：它是什么，解决了什么问题。
- **行动建议**：建议用户如何使用这个项目。

（如果没有匹配的项目，请直接回答"没有找到合适的项目"）
`, promptData.String(), userQuery)

	// 3. 调用 AI
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("AI 检索失败: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "AI 未返回结果", nil
	}

	part := resp.Candidates[0].Content.Parts[0]
	result, ok := part.(genai.Text)
	if !ok {
		return "", fmt.Errorf("AI 返回格式错误")
	}

	return string(result), nil
}
