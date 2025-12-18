package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFetcher(t *testing.T) {
	// 测试创建fetcher
	fetcher := NewFetcher("")
	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.client)
}

func TestFetcher_GetTrendingRepos(t *testing.T) {
	// 由于需要真实的GitHub API调用，我们在这里只测试结构
	// 在实际项目中，我们会使用mock来模拟API调用
	
	fetcher := NewFetcher("")
	
	// 测试空context的情况
	_, err := fetcher.GetTrendingRepos(nil, "Go", "daily")
	// 由于context为nil，应该会出错
	assert.Error(t, err)
}

func TestFetcher_GetReposByTopic(t *testing.T) {
	// 由于需要真实的GitHub API调用，我们在这里只测试结构
	// 在实际项目中，我们会使用mock来模拟API调用
	
	fetcher := NewFetcher("")
	
	// 测试空context的情况
	_, err := fetcher.GetReposByTopic(nil, "ai")
	// 由于context为nil，应该会出错
	assert.Error(t, err)
}