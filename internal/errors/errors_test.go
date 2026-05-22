package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := &AppError{
		Type:      ErrorTypeNetwork,
		Operation: "download",
		Resource:  "file.jpg",
		Message:   "connection failed",
		Inner:     errors.New("dial timeout"),
	}
	expected := "[network] operation: download, resource: file.jpg, connection failed, inner: dial timeout"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &AppError{
		Type:  ErrorTypeConfig,
		Inner: inner,
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestNewConfigError(t *testing.T) {
	inner := errors.New("file not found")
	err := NewConfigError("load", "failed to read config", inner)
	if err.Type != ErrorTypeConfig {
		t.Errorf("expected type config, got %v", err.Type)
	}
	if err.Operation != "load" {
		t.Errorf("expected operation load, got %v", err.Operation)
	}
	if err.Message != "failed to read config" {
		t.Errorf("unexpected message: %v", err.Message)
	}
	if err.Inner != inner {
		t.Error("inner error mismatch")
	}
	if err.Retryable {
		t.Error("config error should not be retryable")
	}
}

func TestNewRetryableError(t *testing.T) {
	inner := errors.New("timeout")
	err := NewRetryableError(ErrorTypeNetwork, "fetch", "request timed out", inner)
	if err.Type != ErrorTypeNetwork {
		t.Errorf("expected type network, got %v", err.Type)
	}
	if err.Operation != "fetch" {
		t.Errorf("expected operation fetch, got %v", err.Operation)
	}
	if err.Message != "request timed out" {
		t.Errorf("unexpected message: %v", err.Message)
	}
	if err.Inner != inner {
		t.Error("inner error mismatch")
	}
	if !err.Retryable {
		t.Error("retryable error should be retryable")
	}
}

func TestNewSkipError(t *testing.T) {
	err := NewSkipError(ErrorTypeSkip, "process", "post-123", "duplicate post")
	if err.Type != ErrorTypeSkip {
		t.Errorf("expected type skip, got %v", err.Type)
	}
	if err.Operation != "process" {
		t.Errorf("expected operation process, got %v", err.Operation)
	}
	if err.Resource != "post-123" {
		t.Errorf("expected resource post-123, got %v", err.Resource)
	}
	if err.Message != "duplicate post" {
		t.Errorf("unexpected message: %v", err.Message)
	}
	if err.Retryable {
		t.Error("skip error should not be retryable")
	}
	if !err.ShouldSkip {
		t.Error("skip error should have ShouldSkip true")
	}
}

func TestIsRetryable_AppError(t *testing.T) {
	retryableErr := NewRetryableError(ErrorTypeNetwork, "op", "msg", nil)
	if !IsRetryable(retryableErr) {
		t.Error("retryable AppError should be retryable")
	}
	nonRetryableErr := NewConfigError("op", "msg", nil)
	if IsRetryable(nonRetryableErr) {
		t.Error("non-retryable AppError should not be retryable")
	}
}

func TestIsRetryable_NetworkError(t *testing.T) {
	err := errors.New("network timeout")
	if !IsRetryable(err) {
		t.Error("network error should be retryable")
	}
	err2 := errors.New("something else")
	if IsRetryable(err2) {
		t.Error("non-network error should not be retryable")
	}
}

func TestAs(t *testing.T) {
	appErr := NewConfigError("test", "test", nil)
	var target *AppError
	if !As(appErr, &target) {
		t.Error("As should match AppError")
	}
	if target != appErr {
		t.Error("target should point to same error")
	}
	// Test with wrapped error
	wrapped := &AppError{
		Type:  ErrorTypeNetwork,
		Inner: appErr,
	}
	var innerTarget *AppError
	if !As(wrapped, &innerTarget) {
		t.Error("As should find inner AppError")
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"network", errors.New("network error"), true},
		{"timeout", errors.New("timeout"), true},
		{"connection", errors.New("connection refused"), true},
		{"other", errors.New("validation failed"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkError(tt.err); got != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, expected %v", tt.err, got, tt.expected)
			}
		})
	}
}