# Error Handling and Logging Design

## Overview
Comprehensive error handling strategy and structured logging system for the Telegram to VK transfer application.

## Error Hierarchy

### Custom Error Types
```go
package errors

import (
	"fmt"
	"strings"
)

// ErrorType categorizes errors for handling
type ErrorType string

const (
	// Configuration errors
	ErrorTypeConfig        ErrorType = "config"
	ErrorTypeValidation    ErrorType = "validation"
	
	// API errors
	ErrorTypeTelegramAPI   ErrorType = "telegram_api"
	ErrorTypeVKAPI         ErrorType = "vk_api"
	ErrorTypeRateLimit     ErrorType = "rate_limit"
	ErrorTypeAuth          ErrorType = "auth"
	
	// Network errors
	ErrorTypeNetwork       ErrorType = "network"
	ErrorTypeTimeout       ErrorType = "timeout"
	
	// Media errors
	ErrorTypeDownload      ErrorType = "download"
	ErrorTypeUpload        ErrorType = "upload"
	ErrorTypeProcessing    ErrorType = "processing"
	ErrorTypeStorage       ErrorType = "storage"
	
	// Content errors
	ErrorTypeTransformation ErrorType = "transformation"
	ErrorTypeFormat         ErrorType = "format"
	
	// System errors
	ErrorTypeIO            ErrorType = "io"
	ErrorTypeMemory        ErrorType = "memory"
	ErrorTypePanic         ErrorType = "panic"
	
	// Business logic errors
	ErrorTypeSkip          ErrorType = "skip"      // Non-fatal, can skip
	ErrorTypeRetry         ErrorType = "retry"     // Should retry
	ErrorTypeFatal         ErrorType = "fatal"     // Cannot continue
)

// AppError is the main error type
type AppError struct {
	Type        ErrorType
	Operation   string      // What operation failed
	Resource    string      // Which resource (file, post ID, etc.)
	Message     string
	Inner       error       // Wrapped error
	Retryable   bool
	ShouldSkip  bool        // If true, can skip this item and continue
	Metadata    map[string]interface{}
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
		Type:       ErrorTypeConfig,
		Operation:  operation,
		Message:    message,
		Inner:      inner,
		Retryable:  false,
		ShouldSkip: false,
	}
}

func NewRetryableError(errType ErrorType, operation, message string, inner error) *AppError {
	return &AppError{
		Type:       errType,
		Operation:  operation,
		Message:    message,
		Inner:      inner,
		Retryable:  true,
		ShouldSkip: false,
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
```

## Error Handling Strategy

### Retry Logic
```go
package retry

import (
	"context"
	"math"
	"time"
	
	"ai-transfer-tg-to-vk/internal/errors"
)

type RetryConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Jitter        bool
	RetryableErrors []errors.ErrorType
}

type Retrier struct {
	config RetryConfig
	logger *zap.Logger
}

func NewRetrier(config RetryConfig) *Retrier {
	return &Retrier{
		config: config,
		logger: zap.L().Named("retrier"),
	}
}

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
			return fmt.Errorf("context cancelled: %w", lastErr)
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
			return fmt.Errorf("context cancelled while waiting: %w", lastErr)
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
	
	return fmt.Errorf("failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

func (r *Retrier) isRetryable(err error) bool {
	// Check if it's our AppError
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		return appErr.Retryable
	}
	
	// Check error type
	for _, retryableType := range r.config.RetryableErrors {
		if appErr != nil && appErr.Type == retryableType {
			return true
		}
	}
	
	// Default: network errors are retryable
	return isNetworkError(err)
}

func (r *Retrier) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: base * 2^(attempt-1)
	delay := r.config.BaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	
	// Cap at max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}
	
	// Add jitter (±20%)
	if r.config.Jitter {
		delay = addJitter(delay, 0.2)
	}
	
	return delay
}
```

### Circuit Breaker Pattern
```go
package circuitbreaker

import (
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "closed"   // Normal operation
	StateOpen     State = "open"     // Fail fast
	StateHalfOpen State = "halfopen" // Testing recovery
)

type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	successThreshold int
	
	state            State
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	mu               sync.RWMutex
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		successThreshold: 3,
		state:            StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()
	
	// Check if circuit is open
	if state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			// Time to try recovery
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}
	
	// Execute the function
	err := fn()
	
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	
	return err
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	
	if cb.state == StateHalfOpen || 
	   (cb.state == StateClosed && cb.failureCount >= cb.failureThreshold) {
		cb.state = StateOpen
		cb.lastFailureTime = time.Now()
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
		}
	} else {
		cb.failureCount = 0
	}
}
```

## Logging System

### Structured Logging with Zap
```go
package logger

import (
	"os"
	"path/filepath"
	
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level      string `yaml:"level"`      // debug, info, warn, error
	Encoding   string `yaml:"encoding"`   // json, console
	OutputPath string `yaml:"output_path"` // File path, empty for stdout
	MaxSize    int    `yaml:"max_size"`   // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`    // Days
}

func NewLogger(config Config) (*zap.Logger, error) {
	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	
	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// Create encoder
	var encoder zapcore.Encoder
	if config.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	
	// Create write syncer
	var writeSyncer zapcore.WriteSyncer
	if config.OutputPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		
		// Create rotating file writer
		rotator := &lumberjack.Logger{
			Filename:   config.OutputPath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   true,
		}
		writeSyncer = zapcore.AddSync(rotator)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	
	// Create core
	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		level,
	)
	
	// Create logger with options
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	
	return logger, nil
}

// ContextLogger adds contextual fields to logs
type ContextLogger struct {
	*zap.Logger
	fields []zap.Field
}

func (cl *ContextLogger) With(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{
		Logger: cl.Logger.With(fields...),
		fields: append(cl.fields[:len(cl.fields):len(cl.fields)], fields...),
	}
}

func (cl *ContextLogger) Operation(name string) *ContextLogger {
	return cl.With(zap.String("operation", name))
}

func (cl *ContextLogger) Resource(id string) *ContextLogger {
	return cl.With(zap.String("resource", id))
}
```

### Log Categories and Levels
```go
// Log levels for different operations
const (
	// DEBUG: Detailed debugging information
	// - API request/response details
	// - File download progress
	// - Transformation steps
	
	// INFO: Normal operational messages
	// - Start/stop of transfer
	// - Posts processed count
	// - Configuration loaded
	
	// WARN: Unexpected but recoverable situations
	// - Rate limiting encountered
	// - Skipped media/files
	// - API deprecation warnings
	
	// ERROR: Operation failures
	// - API errors
	// - File system errors
	// - Network failures
	
	// FATAL: Cannot continue
	// - Configuration errors
	// - Authentication failures
	// - Critical system errors
)

// Example usage
logger.Info("Starting transfer",
	zap.String("source", "telegram"),
	zap.String("target", "vk"),
	zap.Int("batch_size", 100))

logger.Warn("Skipping unsupported media",
	zap.String("type", "sticker"),
	zap.String("post_id", "12345"))

logger.Error("Failed to upload media",
	zap.String("operation", "vk.upload"),
	zap.String("file", "photo.jpg"),
	zap.Error(err))
```

## Error Recovery Strategies

### Per-Component Recovery
```go
package recovery

import (
	"context"
	"runtime"
	
	"ai-transfer-tg-to-vk/internal/errors"
	"go.uber.org/zap"
)

// RecoverPanic handles goroutine panics
func RecoverPanic(logger *zap.Logger, operation string) {
	if r := recover(); r != nil {
		stack := make([]byte, 4096)
		length := runtime.Stack(stack, false)
		
		logger.Error("Panic recovered",
			zap.String("operation", operation),
			zap.Any("panic", r),
			zap.String("stack", string(stack[:length])))
		
		// Convert panic to error
		panicErr := errors.NewAppError(
			errors.ErrorTypePanic,
			operation,
			"",
			fmt.Sprintf("panic: %v", r),
			nil,
			false,
			false,
			map[string]interface{}{
				"stack": string(stack[:length]),
			},
		)
		
		// Could send to error channel or metrics
		_ = panicErr
	}
}

// WithRecovery wraps a function with panic recovery
func WithRecovery(logger *zap.Logger, operation string, fn func() error) (err error) {
	defer RecoverPanic(logger, operation)
	return fn()
}
```

### Checkpoint System for Resumable Transfers
```go
package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Checkpoint struct {
	LastPostID    int64     `json:"last_post_id"`
	LastPostDate  time.Time `json:"last_post_date"`
	ProcessedCount int      `json:"processed_count"`
	FailedCount   int      `json:"failed_count"`
	SkippedCount  int      `json:"skipped_count"`
	MediaCount    int      `json:"media_count"`
	StartTime     time.Time `json:"start_time"`
	LastUpdate    time.Time `json:"last_update"`
	Errors        []string `json:"errors,omitempty"`
}

type Manager struct {
	filePath string
	checkpoint Checkpoint
	mu       sync.RWMutex
	logger   *zap.Logger
}

func NewManager(filePath string) (*Manager, error) {
	m := &Manager{
		filePath: filePath,
		checkpoint: Checkpoint{
			StartTime: time.Now(),
		},
		logger: zap.L().Named("checkpoint"),
	}
	
	// Load existing checkpoint if available
	if err := m.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load checkpoint: %w", err)
	}
	
	return m, nil
}

func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.checkpoint.LastUpdate = time.Now()
	
	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create checkpoint directory: %w", err)
	}
	
	data, err := json.MarshalIndent(m.checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	
	m.logger.Debug("Checkpoint saved",
		zap.String("path", m.filePath),
		zap.Int("processed", m.checkpoint.ProcessedCount))
	
	return nil
}

func (m *Manager) RecordSuccess(postID int64, postDate time.Time, mediaCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.checkpoint.LastPostID = postID
	m.checkpoint.LastPostDate = postDate
	m.checkpoint.ProcessedCount++
	m.checkpoint.MediaCount += mediaCount
}

func (m *Manager) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.checkpoint.FailedCount++
	if len(m.checkpoint.Errors) < 100 { // Keep last 100 errors
		m.checkpoint.Errors = append(m.checkpoint.Errors, err.Error())
	}
}
```

## Monitoring and Metrics

### Prometheus-style Metrics
```go
package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	mu sync.RWMutex
	
	// Counters
	PostsProcessed   int64
	PostsFailed      int64
	PostsSkipped     int64
	MediaDownloaded  int64
	MediaUploaded    int64
	MediaFailed      int64
	
	// Gauges
	ActiveDownloads  int64
	ActiveUploads    int64
	QueueSize        int64
	
	// Histograms
	DownloadDuration []time.Duration
	UploadDuration   []time.Duration
	ProcessingDuration []time.Duration
	
	// Timestamps
	StartTime       time.Time
	LastSuccessTime time.Time
	LastErrorTime   time.Time
}

func (m *Metrics) RecordPostProcessed(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if success {
		m.PostsProcessed++
		m.LastSuccessTime = time.Now()
	} else {
		m.PostsFailed++
		m.LastErrorTime = time.Now()
	}
}

func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return MetricsSnapshot{
		PostsProcessed:   m.PostsProcessed,
		PostsFailed:      m.PostsFailed,
		PostsSkipped:     m.PostsSkipped,
		MediaDownloaded:  m.MediaDownloaded,
		MediaUploaded:    m.MediaUploaded,
		Uptime:           time.Since(m.StartTime),
		SuccessRate:      float64(m.PostsProcessed) / float64(m.PostsProcessed+m.PostsFailed+m.PostsSkipped),
	}
}
```

## Configuration

```yaml
error_handling:
  retry:
    max_attempts: 3
    base_delay: 1s
    max_delay: 30s
    jitter: true
    
  circuit_breaker:
    failure_threshold: 5
    reset_timeout: 60s
    
  checkpoint:
    enabled: true
    file_path: "./checkpoints/last_run.json"
    save_interval: 60s
    
  recovery:
    panic_recovery: true
    max_errors_before_stop: 100
    
logging:
  level: "info"
  encoding: "json"
  output_path: "./logs/transfer.log"
  max_size: 100  # MB
  max_backups: 10
  max_age: 30    # days
  
  # Structured logging fields
  include_caller: true
  include_stacktrace: true
  timestamp_format: "iso8601"
  
metrics:
  enabled: true
  export_interval: 30s
  export_format: "json"  # json, prometheus
  export_path: "./metrics/"
```

## Testing Error Handling

### Test Strategies
1. **Error Injection**: Mock components to return specific errors
2. **Network Simulation**: Use `net/http/httptest` to simulate failures
3. **Rate Limit Testing**: Test exponential backoff behavior
4. **Recovery Testing**: Verify checkpoint system works
5. **Log Verification**: Check logs contain expected error information

### Example Test
```go
func TestRetryLogic(t *testing.T) {
	retrier := NewRetrier(RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
	})
	
	attempts := 0
	err := retrier.Do(context.Background(), "test", func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}
```

## Best Practices
1. **Always wrap errors** with context
2. **Use structured logging** for machine parsing
3. **Implement graceful degradation** - skip non-critical failures
4. **Provide actionable error messages** for users
5. **Log at appropriate levels** - don't spam debug logs in production
6. **Monitor error rates** and set up alerts
7. **Test error paths** as thoroughly as happy paths

## Next Steps
1. Implement custom error types
2. Set up structured logging with Zap
3. Add retry logic for API calls
4. Implement checkpoint system
5. Add metrics collection
6. Write error recovery tests