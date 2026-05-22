package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, 10*time.Second)
	if cb == nil {
		t.Fatal("expected circuit breaker not to be nil")
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Errorf("expected failure count 0, got %d", cb.FailureCount())
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	err := cb.Execute("test", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Errorf("expected failure count 0, got %d", cb.FailureCount())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)
	expectedErr := errors.New("operation failed")
	err := cb.Execute("test", func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state closed (threshold not reached), got %v", cb.State())
	}
	if cb.FailureCount() != 1 {
		t.Errorf("expected failure count 1, got %d", cb.FailureCount())
	}
}

func TestCircuitBreaker_Execute_OpenAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)
	// First failure
	err := cb.Execute("test", func() error {
		return errors.New("fail")
	})
	if err == nil {
		t.Error("expected error")
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
	// Second failure - should open
	err = cb.Execute("test", func() error {
		return errors.New("fail")
	})
	if err == nil {
		t.Error("expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state open, got %v", cb.State())
	}
	// Third attempt should fail fast with ErrCircuitOpen
	err = cb.Execute("test", func() error {
		t.Error("should not be called")
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_Execute_HalfOpenRecovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	// Cause opening
	cb.Execute("test", func() error { return errors.New("fail") })
	cb.Execute("test", func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatalf("expected state open, got %v", cb.State())
	}
	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)
	// Now circuit should be half-open and allow a request
	// Success should transition to closed after successThreshold (default 3)
	// First success
	err := cb.Execute("test", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("expected state half-open, got %v", cb.State())
	}
	// Second success
	err = cb.Execute("test", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Third success should close
	err = cb.Execute("test", func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
}

func TestCircuitBreaker_Execute_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	// Cause opening
	cb.Execute("test", func() error { return errors.New("fail") })
	cb.Execute("test", func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Fatalf("expected state open, got %v", cb.State())
	}
	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)
	// Now circuit is half-open, a failure should re-open
	err := cb.Execute("test", func() error {
		return errors.New("fail")
	})
	if err == nil {
		t.Error("expected error")
	}
	if cb.State() != StateOpen {
		t.Errorf("expected state open, got %v", cb.State())
	}
	// Should be open again, need to wait again
	time.Sleep(10 * time.Millisecond) // not enough time
	err = cb.Execute("test", func() error {
		t.Error("should not be called")
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_StateMonitoring(t *testing.T) {
	cb := NewCircuitBreaker(1, 1*time.Second)
	if cb.State() != StateClosed {
		t.Errorf("expected state closed, got %v", cb.State())
	}
	cb.Execute("test", func() error { return errors.New("fail") })
	if cb.State() != StateOpen {
		t.Errorf("expected state open, got %v", cb.State())
	}
	if cb.FailureCount() != 1 {
		t.Errorf("expected failure count 1, got %d", cb.FailureCount())
	}
}

func TestErrCircuitOpen(t *testing.T) {
	err := ErrCircuitOpen
	if err.Error() != "circuit breaker is open" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}