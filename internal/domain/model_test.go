package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepo(t *testing.T) {
	now := time.Now()
	
	repo := &Repo{
		ID:                  "test-id",
		Name:                "test-repo",
		URL:                 "https://github.com/test/test-repo",
		Description:         "A test repository",
		Stars:               100,
		Language:            "Go",
		CreatedAt:           now,
		UpdatedAt:           now,
		StarGrowthRate:      10.5,
		IsAIProgrammingTool: true,
		LLMScore:            85,
		LLMReview:           "This is a great AI tool",
		AlreadyNotified:     false,
	}
	
	assert.Equal(t, "test-id", repo.ID)
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, "https://github.com/test/test-repo", repo.URL)
	assert.Equal(t, "A test repository", repo.Description)
	assert.Equal(t, 100, repo.Stars)
	assert.Equal(t, "Go", repo.Language)
	assert.Equal(t, now, repo.CreatedAt)
	assert.Equal(t, now, repo.UpdatedAt)
	assert.Equal(t, 10.5, repo.StarGrowthRate)
	assert.True(t, repo.IsAIProgrammingTool)
	assert.Equal(t, 85, repo.LLMScore)
	assert.Equal(t, "This is a great AI tool", repo.LLMReview)
	assert.False(t, repo.AlreadyNotified)
}