package gemini

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAIResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    *aiResponse
	}{
		{
			name:  "Valid JSON response",
			input: `{"is_ai_programming_tool": true, "llm_score": 85, "llm_review": "Excellent AI tool"}`,
			expected: &aiResponse{
				IsAIProgrammingTool: true,
				LLMScore:            85,
				LLMReview:           "Excellent AI tool",
			},
		},
		{
			name: "JSON with extra text",
			input: `Some introduction text
			{
				"is_ai_programming_tool": false,
				"llm_score": 30,
				"llm_review": "Not an AI tool"
			}
			Some trailing text`,
			expected: &aiResponse{
				IsAIProgrammingTool: false,
				LLMScore:            30,
				LLMReview:           "Not an AI tool",
			},
		},
		{
			name:        "Invalid JSON",
			input:       `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "No JSON content",
			input:       `Just some text without JSON`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAIResponse(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.IsAIProgrammingTool, result.IsAIProgrammingTool)
				assert.Equal(t, tt.expected.LLMScore, result.LLMScore)
				assert.Equal(t, tt.expected.LLMReview, result.LLMReview)
			}
		})
	}
}