package infrastructure_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/circuitbreaker"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Closed_Success(t *testing.T) {
	// Arrange
	cb := circuitbreaker.NewCircuitBreaker("test", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    time.Second * 10,
		Timeout:     time.Second * 30,
	})

	ctx := context.Background()

	// Act
	result, err := cb.Execute(ctx, func() (interface{}, error) {
		return "success", nil
	})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, circuitbreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	// Arrange
	cb := circuitbreaker.NewCircuitBreaker("test", circuitbreaker.Config{
		MaxRequests: 3,
		Interval:    time.Second * 10,
		Timeout:     time.Second * 1,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	ctx := context.Background()
	failFunc := func() (interface{}, error) {
		return nil, errors.New("test error")
	}

	// Act - trigger failures to open circuit
	for i := 0; i < 3; i++ {
		_, err := cb.Execute(ctx, failFunc)
		assert.Error(t, err)
	}

	// Assert - circuit should be open now
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	// Further calls should fail immediately with ErrCircuitOpen
	_, err := cb.Execute(ctx, failFunc)
	assert.Equal(t, circuitbreaker.ErrCircuitOpen, err)
}

func TestCircuitBreaker_HalfOpen_Recovery(t *testing.T) {
	// Arrange
	cb := circuitbreaker.NewCircuitBreaker("test", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    time.Second * 10,
		Timeout:     time.Millisecond * 500,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	ctx := context.Background()
	failFunc := func() (interface{}, error) {
		return nil, errors.New("test error")
	}
	successFunc := func() (interface{}, error) {
		return "success", nil
	}

	// Act - open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, failFunc)
	}
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(time.Millisecond * 600)

	// Try successful requests in half-open state
	_, err := cb.Execute(ctx, successFunc)
	assert.NoError(t, err)

	_, err = cb.Execute(ctx, successFunc)
	assert.NoError(t, err)

	// Assert - circuit should be closed now
	assert.Equal(t, circuitbreaker.StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpen_FailureReOpens(t *testing.T) {
	// Arrange
	cb := circuitbreaker.NewCircuitBreaker("test", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    time.Second * 10,
		Timeout:     time.Millisecond * 500,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	ctx := context.Background()
	failFunc := func() (interface{}, error) {
		return nil, errors.New("test error")
	}

	// Open circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, failFunc)
	}
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(time.Millisecond * 600)

	// Fail in half-open state should immediately reopen
	_, err := cb.Execute(ctx, failFunc)
	assert.Error(t, err)
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())
}

func TestCircuitBreaker_Counts(t *testing.T) {
	// Arrange
	cb := circuitbreaker.NewCircuitBreaker("test", circuitbreaker.Config{
		MaxRequests: 10,
		Interval:    time.Second * 10,
		Timeout:     time.Second * 30,
	})

	ctx := context.Background()
	successFunc := func() (interface{}, error) {
		return "success", nil
	}
	failFunc := func() (interface{}, error) {
		return nil, errors.New("test error")
	}

	// Act
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, successFunc)
	}
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, failFunc)
	}

	// Assert
	counts := cb.Counts()
	assert.Equal(t, uint32(5), counts.Requests)
	assert.Equal(t, uint32(3), counts.TotalSuccesses)
	assert.Equal(t, uint32(2), counts.TotalFailures)
	assert.Equal(t, uint32(2), counts.ConsecutiveFailures)
}
