package feishu

import (
	"testing"

	"github-gold-miner/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewNotifier(t *testing.T) {
	// 测试正常创建
	notifier := NewNotifier("https://example.com/webhook")
	assert.NotNil(t, notifier)
	assert.Equal(t, "https://example.com/webhook", notifier.webhookURL)

	// 测试空webhook
	notifier = NewNotifier("")
	assert.NotNil(t, notifier)
	assert.Equal(t, "", notifier.webhookURL)
}

func TestNotifier_Notify(t *testing.T) {
	// 由于Notify方法涉及网络请求，我们在这里只做基本的结构测试
	// 在实际项目中，我们会使用mock来模拟HTTP请求
	
	notifier := NewNotifier("")
	
	repo := &domain.Repo{
		ID:          "test-id",
		Name:        "test-repo",
		URL:         "https://github.com/test/test-repo",
		Description: "A test repository",
		Stars:       100,
		Language:    "Go",
		LLMScore:    80,
		LLMReview:   "This is a great AI tool",
	}
	
	// 测试空webhook的情况
	err := notifier.Notify(nil, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Webhook URL 为空")
}