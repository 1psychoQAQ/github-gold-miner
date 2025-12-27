package common_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github-gold-miner/internal/common"
)

// ExampleDo_basic demonstrates basic usage of the retry mechanism.
func ExampleDo_basic() {
	ctx := context.Background()

	err := common.Do(ctx, func() error {
		// Your API call here
		return nil
	})

	if err != nil {
		fmt.Println("Failed:", err)
	}
	// Output:
}

// ExampleDo_withOptions demonstrates retry with custom configuration.
func ExampleDo_withOptions() {
	ctx := context.Background()

	err := common.Do(ctx,
		func() error {
			// Your API call here
			return nil
		},
		common.WithMaxRetries(5),
		common.WithInitialDelay(time.Second),
		common.WithMaxDelay(30*time.Second),
	)

	if err != nil {
		fmt.Println("Failed:", err)
	}
	// Output:
}

// ExampleDo_githubAPI shows how to use retry with GitHub API calls.
func ExampleDo_githubAPI() {
	ctx := context.Background()

	var repos []string

	err := common.Do(ctx,
		func() error {
			// Simulate GitHub API call
			resp, err := http.Get("https://api.github.com/repos/golang/go")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				return errors.New("server error")
			}

			if resp.StatusCode == 429 {
				return errors.New("rate limited")
			}

			// Process response...
			return nil
		},
		common.WithMaxRetries(3),
		common.WithInitialDelay(time.Second),
	)

	if err != nil {
		fmt.Println("GitHub API call failed:", err)
		return
	}

	fmt.Println("Repos:", repos)
}

// ExampleDo_geminiAPI shows how to use retry with Gemini API calls.
func ExampleDo_geminiAPI() {
	ctx := context.Background()

	var aiResponse string

	err := common.Do(ctx,
		func() error {
			// Simulate Gemini API call
			// In real code, this would be your actual Gemini client call
			resp := simulateGeminiCall()
			if resp == "" {
				return errors.New("empty response")
			}
			aiResponse = resp
			return nil
		},
		common.WithMaxRetries(3),
		common.WithInitialDelay(2*time.Second),
		common.WithMaxDelay(30*time.Second),
	)

	if err != nil {
		fmt.Println("Gemini API call failed:", err)
		return
	}

	fmt.Println("AI Response:", aiResponse)
}

// ExampleDo_feishuWebhook shows how to use retry with Feishu webhooks.
func ExampleDo_feishuWebhook() {
	ctx := context.Background()

	webhookURL := "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
	message := "New AI tool discovered!"

	err := common.Do(ctx,
		func() error {
			// Simulate webhook call
			resp, err := http.Post(webhookURL, "application/json", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return fmt.Errorf("webhook failed with status: %d", resp.StatusCode)
			}

			return nil
		},
		common.WithMaxRetries(3),
		common.WithInitialDelay(500*time.Millisecond),
		common.WithMaxDelay(5*time.Second),
	)

	if err != nil {
		fmt.Println("Feishu notification failed:", err)
		return
	}

	fmt.Println("Notification sent:", message)
}

// ExampleDo_contextTimeout demonstrates using retry with context timeout.
func ExampleDo_contextTimeout() {
	// Create a context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := common.Do(ctx,
		func() error {
			// Long-running operation
			return errors.New("temporary failure")
		},
		common.WithMaxRetries(10),
		common.WithInitialDelay(time.Second),
	)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("Operation timed out")
		} else {
			fmt.Println("Operation failed:", err)
		}
	}
}

// Helper function for example
func simulateGeminiCall() string {
	return "AI analysis result"
}
