package errors

import (
	"fmt"
	"strings"
	stderrors "errors"
)

// ErrorType categorizes errors for handling
type ErrorType string

const (
	// Configuration errors
	ErrorTypeConfig     ErrorType = "config"
	ErrorTypeValidation ErrorType = "validation"

	// API errors
	ErrorTypeTelegramAPI ErrorType = "telegram_api"
	ErrorTypeVKAPI       ErrorType = "vk_api"
	ErrorTypeRateLimit   ErrorType = "rate_limit"
	ErrorTypeAuth        ErrorType = "auth"

	// Network errors
	ErrorTypeNetwork ErrorType = "network"
	ErrorTypeTimeout ErrorType = "timeout"

	// Media errors
	ErrorTypeDownload   ErrorType = "download"
	ErrorTypeUpload     ErrorType = "upload"
	ErrorTypeProcessing ErrorType = "processing"
	ErrorTypeStorage    ErrorType = "storage"

	// Content errors
	ErrorTypeTransformation ErrorType = "transformation"
	ErrorTypeFormat         ErrorType = "format"

	// System errors
	ErrorTypeIO     ErrorType = "io"
	ErrorTypeMemory ErrorType = "memory"
	ErrorTypePanic  ErrorType = "panic"

	// Business logic errors
	ErrorTypeSkip  ErrorType = "skip"  // Non-fatal, can skip
	ErrorTypeRetry ErrorType = "retry" // Should retry
	ErrorTypeFatal ErrorType = "fatal" // Cannot continue
)

// AppError is the main error type
type AppError struct {
	Type       ErrorType
	Operation  string // What operation failed
	Resource   string // Which resource (file, post ID, etc.)
	Message    string
	Inner      error // Wrapped error
	Retryable  bool
	ShouldSkip bool // If true, can skip this item and continue
	Metadata   map[string]interface{}
}

func (e *AppError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation: %s", e.Operation))
	}

	if e.Resource != "" {
		parts = append(parts, fmt.Sprintf("resource: %s", e.Resource))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.Inner != nil {
		parts = append(parts, fmt.Sprintf("inner: %v", e.Inner))
	}

	return fmt.Sprintf("[%s] %s", e.Type, strings.Join(parts, ", "))
}

func (e *AppError) Unwrap() error {
	return e.Inner
}

// Helper constructors
func NewConfigError(operation, message string, inner error) *AppError {
	return &AppError{
		Type:      ErrorTypeConfig,
		Operation: operation,
		Message:   message,
		Inner:     inner,
		Retryable: false,
	}
}

func NewRetryableError(errType ErrorType, operation, message string, inner error) *AppError {
	return &AppError{
		Type:      errType,
		Operation: operation,
		Message:   message,
		Inner:     inner,
		Retryable: true,
	}
}

func NewSkipError(errType ErrorType, operation, resource, message string) *AppError {
	return &AppError{
		Type:       errType,
		Operation:  operation,
		Resource:   resource,
		Message:    message,
		Retryable:  false,
		ShouldSkip: true,
	}
}

// IsRetryable checks if error is retryable
func IsRetryable(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Retryable
	}
	// Default: network errors are retryable
	return isNetworkError(err)
}

// As is a wrapper for errors.As
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

// isNetworkError determines if error is a network error (simplified)
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// In a real implementation, check for net.OpError, etc.
	return strings.Contains(strings.ToLower(err.Error()), "network") ||
		strings.Contains(strings.ToLower(err.Error()), "timeout") ||
		strings.Contains(strings.ToLower(err.Error()), "connection")
}