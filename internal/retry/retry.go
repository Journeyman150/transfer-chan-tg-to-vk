package retry

import (
	"context"
	"math"
	"math/rand"
	"strings"
	"time"

	"ai-transfer-tg-to-vk/internal/config"
	"ai-transfer-tg-to-vk/internal/errors"
	"go.uber.org/zap"
)

// Retrier implements exponential backoff retry logic
type Retrier struct {
	config config.RetryConfig
	logger *zap.Logger
}

// NewRetrier creates a new retrier with the given configuration
func NewRetrier(cfg config.RetryConfig) *Retrier {
	return &Retrier{
		config: cfg,
		logger: zap.L().Named("retrier"),
	}
}

// Do executes the operation with retries
func (r *Retrier) Do(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Execute the operation
		err := fn()

		// Success
		if err == nil {
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry",
					zap.String("operation", operation),
					zap.Int("attempt", attempt))
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryable(err) {
			return err
		}

		// Check context cancellation
		if ctx.Err() != nil {
			return errors.NewRetryableError(errors.ErrorTypeTimeout, operation, "context cancelled", lastErr)
		}

		// If this was the last attempt, break
		if attempt == r.config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff
		delay := r.calculateDelay(attempt)

		r.logger.Warn("Operation failed, retrying",
			zap.String("operation", operation),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", r.config.MaxAttempts),
			zap.Duration("delay", delay),
			zap.Error(err))

		// Wait before retry
		select {
		case <-ctx.Done():
			return errors.NewRetryableError(errors.ErrorTypeTimeout, operation, "context cancelled while waiting", lastErr)
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return errors.NewRetryableError(errors.ErrorTypeRetry, operation, "failed after maximum retries", lastErr)
}

// isRetryable checks if error is retryable
func (r *Retrier) isRetryable(err error) bool {
	// Check if it's our AppError
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		return appErr.Retryable
	}

	// Default: network errors are retryable
	return isNetworkError(err)
}

// calculateDelay computes exponential backoff delay with jitter
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: base * 2^(attempt-1)
	base := time.Duration(r.config.BaseDelay)
	max := time.Duration(r.config.MaxDelay)

	delay := base * time.Duration(math.Pow(2, float64(attempt-1)))

	// Cap at max delay
	if delay > max {
		delay = max
	}

	// Add jitter (±20%)
	if r.config.Jitter {
		delay = addJitter(delay, 0.2)
	}

	return delay
}

// addJitter adds random jitter to delay
func addJitter(delay time.Duration, jitterFactor float64) time.Duration {
	if jitterFactor <= 0 {
		return delay
	}
	jitter := float64(delay) * jitterFactor
	// random value between -jitter and +jitter
	randomJitter := rand.Float64()*2*jitter - jitter
	return time.Duration(float64(delay) + randomJitter)
}

// isNetworkError determines if error is a network error (simplified)
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// In a real implementation, check for net.OpError, etc.
	return containsAny(errStr, "network", "timeout", "connection", "reset", "refused", "dial")
}

// containsAny checks if string contains any of the substrings (case-insensitive)
func containsAny(s string, substrings ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}