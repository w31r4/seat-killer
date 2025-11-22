package retry

import (
	"errors"
	"log"
	"time"
)

// Func is a function that can be retried.
type Func func() error

// UnretryableError marks an error as not suitable for retry.
type UnretryableError struct {
	Err error
}

// Error implements the error interface.
func (e *UnretryableError) Error() string {
	return e.Err.Error()
}

// Unwrap provides compatibility for `errors.Is` and `errors.As`.
func (e *UnretryableError) Unwrap() error {
	return e.Err
}

// WrapUnretryable wraps an error to mark it as unretryable.
func WrapUnretryable(err error) error {
	if err == nil {
		return nil
	}
	return &UnretryableError{Err: err}
}

// WithRetry executes a function with a specified number of retry attempts and delay.
// It stops retrying if the function returns an UnretryableError.
func WithRetry(fn Func, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil // Success
		}

		var unretryableErr *UnretryableError
		if errors.As(err, &unretryableErr) {
			log.Printf("Attempt %d/%d failed with unretryable error: %v", i+1, attempts, err)
			return unretryableErr.Unwrap() // Not retryable, return original error
		}

		log.Printf("Attempt %d/%d failed: %v. Retrying in %s...", i+1, attempts, err, delay)
		time.Sleep(delay)
	}
	return err // Return the last error
}
