package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-transfer-tg-to-vk/internal/config"
	apperrors "ai-transfer-tg-to-vk/internal/errors"
)

func TestNewRetrier(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(1 * time.Second),
		MaxDelay:    config.Duration(30 * time.Second),
		Jitter:      true,
	}
	retrier := NewRetrier(cfg)
	if retrier == nil {
		t.Fatal("expected retrier not to be nil")
	}
}

func TestRetrier_Do_SuccessFirstAttempt(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_Do_SuccessAfterRetry(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		if attempts < 2 {
			return errors.New("network timeout")
		}
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetrier_Do_MaxAttemptsExceeded(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		return errors.New("network timeout")
	})
	if err == nil {
		t.Error("expected error after max attempts")
	}
	var retryErr *apperrors.AppError
	if !errors.As(err, &retryErr) {
		t.Errorf("expected AppError, got %T", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetrier_Do_NonRetryableError(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	expectedErr := errors.New("non-retryable")
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("expected same error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_Do_ContextCancelled(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(100 * time.Millisecond),
		MaxDelay:    config.Duration(1 * time.Second),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	attempts := 0
	err := retrier.Do(ctx, "test", func() error {
		attempts++
		return errors.New("network error")
	})
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
	// The operation will be executed once before checking context cancellation
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_Do_RetryableAppError(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	retryableErr := apperrors.NewRetryableError(apperrors.ErrorTypeNetwork, "test", "network error", nil)
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		return retryableErr
	})
	if err == nil {
		t.Error("expected error after max attempts")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetrier_Do_NonRetryableAppError(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(10 * time.Millisecond),
		MaxDelay:    config.Duration(100 * time.Millisecond),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	attempts := 0
	nonRetryableErr := &apperrors.AppError{
		Type:      apperrors.ErrorTypeValidation,
		Operation: "test",
		Message:   "validation error",
		Retryable: false,
	}
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		return nonRetryableErr
	})
	if err != nonRetryableErr {
		t.Errorf("expected same error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetrier_isRetryable_NetworkError(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(1 * time.Second),
		MaxDelay:    config.Duration(30 * time.Second),
		Jitter:      true,
	}
	retrier := NewRetrier(cfg)
	err := errors.New("network timeout")
	if !retrier.isRetryable(err) {
		t.Error("network error should be retryable")
	}
}

func TestRetrier_isRetryable_NonNetworkError(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(1 * time.Second),
		MaxDelay:    config.Duration(30 * time.Second),
		Jitter:      true,
	}
	retrier := NewRetrier(cfg)
	err := errors.New("something else")
	if retrier.isRetryable(err) {
		t.Error("non-network error should not be retryable")
	}
}

func TestRetrier_calculateDelay_ExponentialBackoff(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   config.Duration(1 * time.Second),
		MaxDelay:    config.Duration(30 * time.Second),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	// attempt 1: base * 2^0 = 1s
	delay := retrier.calculateDelay(1)
	if delay != 1*time.Second {
		t.Errorf("expected 1s, got %v", delay)
	}
	// attempt 2: base * 2^1 = 2s
	delay = retrier.calculateDelay(2)
	if delay != 2*time.Second {
		t.Errorf("expected 2s, got %v", delay)
	}
	// attempt 3: 4s
	delay = retrier.calculateDelay(3)
	if delay != 4*time.Second {
		t.Errorf("expected 4s, got %v", delay)
	}
}

func TestRetrier_calculateDelay_CappedAtMax(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   config.Duration(10 * time.Second),
		MaxDelay:    config.Duration(15 * time.Second),
		Jitter:      false,
	}
	retrier := NewRetrier(cfg)
	// attempt 2: 20s > max, should cap at 15s
	delay := retrier.calculateDelay(2)
	if delay != 15*time.Second {
		t.Errorf("expected 15s, got %v", delay)
	}
}

func TestRetrier_calculateDelay_Jitter(t *testing.T) {
	cfg := config.RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   config.Duration(1 * time.Second),
		MaxDelay:    config.Duration(30 * time.Second),
		Jitter:      true,
	}
	retrier := NewRetrier(cfg)
	// Jitter adds randomness, we can't assert exact value, but ensure it's within bounds
	delay := retrier.calculateDelay(1)
	if delay < 800*time.Millisecond || delay > 1200*time.Millisecond {
		t.Errorf("delay %v outside expected jitter range (0.8s - 1.2s)", delay)
	}
}

func TestAddJitter(t *testing.T) {
	tests := []struct {
		name         string
		delay        time.Duration
		jitterFactor float64
		expectRange  [2]time.Duration
	}{
		{
			name:         "positive jitter",
			delay:        100 * time.Millisecond,
			jitterFactor: 0.2,
			expectRange:  [2]time.Duration{80 * time.Millisecond, 120 * time.Millisecond},
		},
		{
			name:         "zero jitter factor",
			delay:        100 * time.Millisecond,
			jitterFactor: 0,
			expectRange:  [2]time.Duration{100 * time.Millisecond, 100 * time.Millisecond},
		},
		{
			name:         "negative jitter factor",
			delay:        100 * time.Millisecond,
			jitterFactor: -0.5,
			expectRange:  [2]time.Duration{100 * time.Millisecond, 100 * time.Millisecond},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addJitter(tt.delay, tt.jitterFactor)
			if got < tt.expectRange[0] || got > tt.expectRange[1] {
				t.Errorf("addJitter(%v, %v) = %v, expected between %v and %v",
					tt.delay, tt.jitterFactor, got, tt.expectRange[0], tt.expectRange[1])
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"network error", errors.New("network timeout"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"dial error", errors.New("dial tcp 127.0.0.1:8080: connectex"), true},
		{"reset", errors.New("connection reset by peer"), true},
		{"non-network", errors.New("validation failed"), false},
		{"nil error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkError(tt.err); got != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, expected %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("network timeout", "network", "timeout") {
		t.Error("should contain")
	}
	if containsAny("hello world", "network", "timeout") {
		t.Error("should not contain")
	}
}