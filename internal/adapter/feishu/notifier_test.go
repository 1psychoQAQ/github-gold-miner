package feishu

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
)

// mockFeishuServer 创建模拟的飞书 Webhook 服务器
func mockFeishuServer(t *testing.T, statusCode int, validatePayload func(*testing.T, map[string]interface{})) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		assert.Equal(t, http.MethodPost, r.Method)

		// 验证 Content-Type
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// 读取并解析请求体
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var payload map[string]interface{}
		err = json.Unmarshal(body, &payload)
		assert.NoError(t, err)

		// 如果提供了验证函数，执行验证
		if validatePayload != nil {
			validatePayload(t, payload)
		}

		// 返回指定的状态码
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"code": 0, "msg": "success"}`))
	}))
}

func TestNotifier_Notify(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		repo            *domain.Repo
		statusCode      int
		expectError     bool
		errorSubstring  string
		validatePayload func(*testing.T, map[string]interface{})
	}{
		{
			name: "成功发送通知",
			repo: &domain.Repo{
				ID:              "github-123",
				Name:            "test/awesome-tool",
				URL:             "https://github.com/test/awesome-tool",
				Description:     "An awesome AI coding tool",
				Stars:           500,
				Language:        "Python",
				CreatedAt:       now.AddDate(0, 0, -5),
				UpdatedAt:       now,
				StarGrowthRate:  100.5,
				IsAIProgrammingTool: true,
				LLMScore:        85,
				LLMReview:       "Excellent AI tool with great potential",
				AlreadyNotified: false,
			},
			statusCode:  http.StatusOK,
			expectError: false,
			validatePayload: func(t *testing.T, payload map[string]interface{}) {
				// 验证消息类型
				assert.Equal(t, "interactive", payload["msg_type"])

				// 验证卡片结构
				card, ok := payload["card"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "2.0", card["schema"])

				// 验证标题
				header, ok := card["header"].(map[string]interface{})
				assert.True(t, ok)
				title, ok := header["title"].(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, title["content"], "test/awesome-tool")

				// 验证 body
				body, ok := card["body"].(map[string]interface{})
				assert.True(t, ok)
				elements, ok := body["elements"].([]interface{})
				assert.True(t, ok)
				assert.Equal(t, 2, len(elements)) // markdown + button
			},
		},
		{
			name: "高分项目通知",
			repo: &domain.Repo{
				ID:              "github-456",
				Name:            "ai/super-coder",
				URL:             "https://github.com/ai/super-coder",
				Description:     "Revolutionary AI coding assistant",
				Stars:           1000,
				Language:        "Go",
				CreatedAt:       now.AddDate(0, 0, -10),
				UpdatedAt:       now,
				StarGrowthRate:  200.0,
				IsAIProgrammingTool: true,
				LLMScore:        95,
				LLMReview:       "Outstanding tool that redefines AI-assisted coding",
				AlreadyNotified: false,
			},
			statusCode:  http.StatusOK,
			expectError: false,
			validatePayload: func(t *testing.T, payload map[string]interface{}) {
				card := payload["card"].(map[string]interface{})
				body := card["body"].(map[string]interface{})
				elements := body["elements"].([]interface{})

				// 验证 markdown 内容
				markdown := elements[0].(map[string]interface{})
				content := markdown["content"].(string)
				assert.Contains(t, content, "1000") // stars
				assert.Contains(t, content, "95")   // LLM score
				assert.Contains(t, content, "Go")   // language
				assert.Contains(t, content, "200.00") // growth rate
			},
		},
		{
			name: "包含特殊字符的项目",
			repo: &domain.Repo{
				ID:              "github-789",
				Name:            "test/tool-with-特殊字符",
				URL:             "https://github.com/test/tool-with-特殊字符",
				Description:     "Tool with special chars: <>&\"'",
				Stars:           50,
				Language:        "TypeScript",
				CreatedAt:       now.AddDate(0, 0, -3),
				UpdatedAt:       now,
				StarGrowthRate:  16.67,
				IsAIProgrammingTool: true,
				LLMScore:        70,
				LLMReview:       "Good tool with room for improvement",
				AlreadyNotified: false,
			},
			statusCode:  http.StatusOK,
			expectError: false,
			validatePayload: func(t *testing.T, payload map[string]interface{}) {
				card := payload["card"].(map[string]interface{})
				header := card["header"].(map[string]interface{})
				title := header["title"].(map[string]interface{})
				assert.Contains(t, title["content"], "特殊字符")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockFeishuServer(t, tt.statusCode, tt.validatePayload)
			defer server.Close()

			notifier := NewNotifier(server.URL)
			ctx := context.Background()

			err := notifier.Notify(ctx, tt.repo)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorSubstring != "" {
					assert.Contains(t, err.Error(), tt.errorSubstring)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotifier_Notify_ErrorCases(t *testing.T) {
	now := time.Now()
	testRepo := &domain.Repo{
		ID:              "github-999",
		Name:            "test/error-repo",
		URL:             "https://github.com/test/error-repo",
		Description:     "Test error handling",
		Stars:           100,
		Language:        "Rust",
		CreatedAt:       now.AddDate(0, 0, -2),
		UpdatedAt:       now,
		StarGrowthRate:  50.0,
		IsAIProgrammingTool: true,
		LLMScore:        75,
		LLMReview:       "Test review",
		AlreadyNotified: false,
	}

	tests := []struct {
		name           string
		setupNotifier  func() *Notifier
		repo           *domain.Repo
		expectError    bool
		errorSubstring string
	}{
		{
			name: "Webhook URL 为空",
			setupNotifier: func() *Notifier {
				return NewNotifier("")
			},
			repo:           testRepo,
			expectError:    true,
			errorSubstring: "Webhook URL 为空",
		},
		{
			name: "飞书 API 返回 400 错误",
			setupNotifier: func() *Notifier {
				server := mockFeishuServer(t, http.StatusBadRequest, nil)
				t.Cleanup(server.Close)
				return NewNotifier(server.URL)
			},
			repo:           testRepo,
			expectError:    true,
			errorSubstring: "飞书 API 报错",
		},
		{
			name: "飞书 API 返回 500 错误",
			setupNotifier: func() *Notifier {
				server := mockFeishuServer(t, http.StatusInternalServerError, nil)
				t.Cleanup(server.Close)
				return NewNotifier(server.URL)
			},
			repo:           testRepo,
			expectError:    true,
			errorSubstring: "飞书 API 报错",
		},
		{
			name: "飞书 API 返回 403 Forbidden",
			setupNotifier: func() *Notifier {
				server := mockFeishuServer(t, http.StatusForbidden, nil)
				t.Cleanup(server.Close)
				return NewNotifier(server.URL)
			},
			repo:           testRepo,
			expectError:    true,
			errorSubstring: "飞书 API 报错",
		},
		{
			name: "无效的 Webhook URL",
			setupNotifier: func() *Notifier {
				return NewNotifier("http://invalid-url-that-does-not-exist-12345.com")
			},
			repo:           testRepo,
			expectError:    true,
			errorSubstring: "发送请求失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := tt.setupNotifier()
			ctx := context.Background()

			err := notifier.Notify(ctx, tt.repo)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorSubstring != "" {
					assert.Contains(t, err.Error(), tt.errorSubstring)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotifier_Notify_ContextCancellation(t *testing.T) {
	// 创建一个慢速服务器
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // 模拟慢速响应
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	notifier := NewNotifier(slowServer.URL)

	// 创建一个会快速取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	now := time.Now()
	repo := &domain.Repo{
		ID:              "github-timeout",
		Name:            "test/timeout-repo",
		URL:             "https://github.com/test/timeout-repo",
		Description:     "Test timeout",
		Stars:           100,
		Language:        "Go",
		CreatedAt:       now,
		UpdatedAt:       now,
		StarGrowthRate:  50.0,
		IsAIProgrammingTool: true,
		LLMScore:        80,
		LLMReview:       "Test",
		AlreadyNotified: false,
	}

	err := notifier.Notify(ctx, repo)

	// 由于使用了重试机制，可能会收到上下文取消错误
	// 或者在重试期间超时
	if err != nil {
		// 验证错误包含预期的错误信息
		assert.Contains(t, err.Error(), "发送请求失败")
	}
}

func TestNotifier_Notify_PayloadStructure(t *testing.T) {
	now := time.Now()
	repo := &domain.Repo{
		ID:              "github-payload-test",
		Name:            "test/payload-structure",
		URL:             "https://github.com/test/payload-structure",
		Description:     "Test payload structure",
		Stars:           250,
		Language:        "JavaScript",
		CreatedAt:       now.AddDate(0, 0, -7),
		UpdatedAt:       now,
		StarGrowthRate:  35.71,
		IsAIProgrammingTool: true,
		LLMScore:        82,
		LLMReview:       "Solid AI coding tool with innovative features",
		AlreadyNotified: false,
	}

	server := mockFeishuServer(t, http.StatusOK, func(t *testing.T, payload map[string]interface{}) {
		// 深度验证 payload 结构
		assert.Equal(t, "interactive", payload["msg_type"])

		card, ok := payload["card"].(map[string]interface{})
		assert.True(t, ok)

		// 验证 schema
		assert.Equal(t, "2.0", card["schema"])

		// 验证 config
		config, ok := card["config"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, config["update_multi"])

		// 验证 header
		header, ok := card["header"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "blue", header["template"])
		title, ok := header["title"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "plain_text", title["tag"])
		assert.Contains(t, title["content"], "test/payload-structure")

		// 验证 body
		body, ok := card["body"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "vertical", body["direction"])

		// 验证 elements
		elements, ok := body["elements"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 2, len(elements))

		// 验证 markdown 元素
		markdownElement := elements[0].(map[string]interface{})
		assert.Equal(t, "markdown", markdownElement["tag"])
		assert.Equal(t, "normal", markdownElement["text_size"])
		content := markdownElement["content"].(string)
		assert.Contains(t, content, "250")      // stars
		assert.Contains(t, content, "82")       // LLM score
		assert.Contains(t, content, "35.71")    // growth rate
		assert.Contains(t, content, "JavaScript")

		// 验证 button 元素
		buttonElement := elements[1].(map[string]interface{})
		assert.Equal(t, "button", buttonElement["tag"])
		assert.Equal(t, "primary", buttonElement["type"])

		buttonText := buttonElement["text"].(map[string]interface{})
		assert.Equal(t, "plain_text", buttonText["tag"])
		assert.Contains(t, buttonText["content"], "查看源码")

		behaviors := buttonElement["behaviors"].([]interface{})
		assert.Equal(t, 1, len(behaviors))
		behavior := behaviors[0].(map[string]interface{})
		assert.Equal(t, "open_url", behavior["type"])
		assert.Equal(t, repo.URL, behavior["default_url"])
	})
	defer server.Close()

	notifier := NewNotifier(server.URL)
	ctx := context.Background()

	err := notifier.Notify(ctx, repo)
	assert.NoError(t, err)
}

func TestNewNotifier(t *testing.T) {
	tests := []struct {
		name    string
		webhook string
		verify  func(*testing.T, *Notifier)
	}{
		{
			name:    "有效的 Webhook URL",
			webhook: "https://open.feishu.cn/open-apis/bot/v2/hook/test-hook",
			verify: func(t *testing.T, n *Notifier) {
				assert.NotNil(t, n)
				assert.Equal(t, "https://open.feishu.cn/open-apis/bot/v2/hook/test-hook", n.webhookURL)
			},
		},
		{
			name:    "空 Webhook URL",
			webhook: "",
			verify: func(t *testing.T, n *Notifier) {
				assert.NotNil(t, n)
				assert.Equal(t, "", n.webhookURL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := NewNotifier(tt.webhook)
			tt.verify(t, notifier)
		})
	}
}

func TestNotifier_Notify_EdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		repo        *domain.Repo
		expectError bool
	}{
		{
			name: "零星项目",
			repo: &domain.Repo{
				ID:              "github-zero-stars",
				Name:            "test/zero-stars",
				URL:             "https://github.com/test/zero-stars",
				Description:     "Brand new project",
				Stars:           0,
				Language:        "Python",
				CreatedAt:       now,
				UpdatedAt:       now,
				StarGrowthRate:  0.0,
				IsAIProgrammingTool: true,
				LLMScore:        50,
				LLMReview:       "Promising but needs more work",
				AlreadyNotified: false,
			},
			expectError: false,
		},
		{
			name: "无语言项目",
			repo: &domain.Repo{
				ID:              "github-no-lang",
				Name:            "test/no-language",
				URL:             "https://github.com/test/no-language",
				Description:     "Documentation only",
				Stars:           100,
				Language:        "",
				CreatedAt:       now,
				UpdatedAt:       now,
				StarGrowthRate:  10.0,
				IsAIProgrammingTool: true,
				LLMScore:        60,
				LLMReview:       "Useful documentation",
				AlreadyNotified: false,
			},
			expectError: false,
		},
		{
			name: "空描述项目",
			repo: &domain.Repo{
				ID:              "github-no-desc",
				Name:            "test/no-description",
				URL:             "https://github.com/test/no-description",
				Description:     "",
				Stars:           200,
				Language:        "Go",
				CreatedAt:       now,
				UpdatedAt:       now,
				StarGrowthRate:  20.0,
				IsAIProgrammingTool: true,
				LLMScore:        70,
				LLMReview:       "Good tool",
				AlreadyNotified: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockFeishuServer(t, http.StatusOK, nil)
			defer server.Close()

			notifier := NewNotifier(server.URL)
			ctx := context.Background()

			err := notifier.Notify(ctx, tt.repo)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
