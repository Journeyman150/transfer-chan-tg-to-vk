package circuitbreaker

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state
type State string

const (
	StateClosed   State = "closed"   // Normal operation
	StateOpen     State = "open"     // Fail fast
	StateHalfOpen State = "halfopen" // Testing recovery
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	successThreshold int

	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex
	logger          *zap.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		successThreshold: 3,
		state:            StateClosed,
		logger:           zap.L().Named("circuitbreaker"),
	}
}

// Execute runs the operation through the circuit breaker
func (cb *CircuitBreaker) Execute(operation string, fn func() error) error {
	// Check if circuit is open
	if !cb.allowRequest() {
		cb.logger.Warn("Circuit breaker open, failing fast",
			zap.String("operation", operation),
			zap.String("state", string(cb.state)))
		return ErrCircuitOpen
	}

	// Execute the operation
	err := fn()

	if err == nil {
		cb.onSuccess()
		return nil
	}

	cb.onFailure()
	return err
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateHalfOpen:
		// Allow a limited number of requests to test recovery
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			// Time to try recovery
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			cb.logger.Info("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	default:
		return true
	}
}

// onSuccess records a successful operation
func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure count on consecutive successes
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.logger.Info("Circuit breaker closed after successful recovery")
		}
	}
}

// onFailure records a failed operation
func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastFailureTime = time.Now()
			cb.logger.Warn("Circuit breaker opened due to failures",
				zap.Int("failure_count", cb.failureCount),
				zap.Int("threshold", cb.failureThreshold))
		}
	case StateHalfOpen:
		// Single failure in half-open state trips back to open
		cb.state = StateOpen
		cb.lastFailureTime = time.Now()
		cb.logger.Warn("Circuit breaker re-opened after failure in half-open state")
	}
}

// State returns the current state (for monitoring)
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// FailureCount returns the current failure count (for monitoring)
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = &CircuitOpenError{}

// CircuitOpenError represents a circuit breaker open error
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}