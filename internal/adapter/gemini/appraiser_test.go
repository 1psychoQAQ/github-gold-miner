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
			input: `{"is_ai_programming_tool": true, "llm_score": 80, "llm_review": "Good tool"}`,
			expected: &aiResponse{
				IsAIProgrammingTool: true,
				LLMScore:            80,
				LLMReview:           "Good tool",
			},
		},
		{
			name:  "JSON with extra text",
			input: `Some text here {"is_ai_programming_tool": false, "llm_score": 30, "llm_review": "Not relevant"} and more text`,
			expected: &aiResponse{
				IsAIProgrammingTool: false,
				LLMScore:            30,
				LLMReview:           "Not relevant",
			},
		},
		{
			name:        "Invalid JSON",
			input:       `{"is_ai_programming_tool": true, "llm_score": }`,
			expectError: true,
		},
		{
			name:        "No JSON content",
			input:       `This is just plain text without JSON`,
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
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}