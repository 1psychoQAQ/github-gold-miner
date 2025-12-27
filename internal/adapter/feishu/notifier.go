package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github-gold-miner/internal/common"
	"github-gold-miner/internal/domain"
)

type Notifier struct {
	webhookURL string
}

func NewNotifier(webhook string) *Notifier {
	if webhook == "" {
		log.Println("⚠️ 警告: 飞书 Webhook 为空，推送功能将无法工作！")
	}
	return &Notifier{webhookURL: webhook}
}

// Notify 发送飞书卡片消息 (Schema 2.0)
func (n *Notifier) Notify(ctx context.Context, repo *domain.Repo) error {
	if n.webhookURL == "" {
		return fmt.Errorf("Webhook URL 为空")
	}

	// 1. 准备标题
	title := fmt.Sprintf("🚨 发现AI编程工具: %s", repo.Name)

	// 2. 构造 Markdown 内容
	mdContent := fmt.Sprintf(`**⭐ Stars:** %d  |  **语言:** %s  |  **创建日期:** %s
**🏆 LLM评分:** %d/100

**📝 项目描述:**
%s

**🤖 AI评价:**
%s

**📈 Star增长速率:** %.2f stars/天
`,
		repo.Stars, repo.Language, repo.CreatedAt.Format("2006-01-02"),
		repo.LLMScore,
		repo.Description,
		repo.LLMReview,
		repo.StarGrowthRate)

	// 3. 构造 Schema 2.0 JSON 结构
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"schema": "2.0",
			"config": map[string]interface{}{
				"update_multi": true,
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": title,
				},
				"template": "blue",
			},
			"body": map[string]interface{}{
				"direction": "vertical",
				"elements": []map[string]interface{}{
					{
						"tag":       "markdown",
						"content":   mdContent,
						"text_size": "normal",
					},
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "🔗 查看源码",
						},
						"type": "primary",
						"behaviors": []map[string]interface{}{
							{
								"type":        "open_url",
								"default_url": repo.URL,
							},
						},
					},
				},
			},
		},
	}

	// 4. 发送请求 (带重试机制)
	body, _ := json.Marshal(payload)
	err := common.Do(ctx, func() error {
		resp, postErr := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(body))
		if postErr != nil {
			return postErr
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("飞书 API 报错: 状态码 %d", resp.StatusCode)
		}
		return nil
	},
		common.WithMaxRetries(3),
		common.WithInitialDelay(500*time.Millisecond),
	)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}

	return nil
}