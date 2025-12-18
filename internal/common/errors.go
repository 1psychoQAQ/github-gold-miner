package common

import "fmt"

// AppError 应用级错误结构
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// WrapError 包装错误
func WrapError(code, message string, err error) error {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewError 创建新错误
func NewError(code, message string) error {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// 错误码常量
const (
	ErrCodeGitHubAPI     = "GITHUB_API_ERROR"
	ErrCodeDatabase      = "DATABASE_ERROR"
	ErrCodeAIProcessing  = "AI_PROCESSING_ERROR"
	ErrCodeNotification  = "NOTIFICATION_ERROR"
	ErrCodeInvalidInput  = "INVALID_INPUT"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeInternal      = "INTERNAL_ERROR"
)