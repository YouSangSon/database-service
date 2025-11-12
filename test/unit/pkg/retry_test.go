package pkg_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YouSangSon/database-service/internal/pkg/retry"
	"github.com/stretchr/testify/assert"
)

func TestRetry_Success_FirstAttempt(t *testing.T) {
	// Arrange
	ctx := context.Background()
	config := retry.DefaultConfig()
	attemptCount := 0

	fn := func(ctx context.Context) error {
		attemptCount++
		return nil
	}

	// Act
	err := retry.Do(ctx, config, fn)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
}

func TestRetry_Success_AfterRetries(t *testing.T) {
	// Arrange
	ctx := context.Background()
	config := retry.Config{
		MaxAttempts: 5,
		InitialDelay: time.Millisecond * 10,
		MaxDelay: time.Millisecond * 100,
		Multiplier: 2.0,
	}
	attemptCount := 0
	failUntil := 3

	fn := func(ctx context.Context) error {
		attemptCount++
		if attemptCount < failUntil {
			return errors.New("temporary error")
		}
		return nil
	}

	// Act
	err := retry.Do(ctx, config, fn)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, failUntil, attemptCount)
}

func TestRetry_Failure_MaxAttemptsReached(t *testing.T) {
	// Arrange
	ctx := context.Background()
	config := retry.Config{
		MaxAttempts: 3,
		InitialDelay: time.Millisecond * 10,
		MaxDelay: time.Millisecond * 100,
		Multiplier: 2.0,
	}
	attemptCount := 0
	expectedErr := errors.New("persistent error")

	fn := func(ctx context.Context) error {
		attemptCount++
		return expectedErr
	}

	// Act
	err := retry.Do(ctx, config, fn)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 3, attemptCount)
}

func TestRetry_ContextCanceled(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	config := retry.Config{
		MaxAttempts: 10,
		InitialDelay: time.Millisecond * 100,
		MaxDelay: time.Second,
		Multiplier: 2.0,
	}
	attemptCount := 0

	fn := func(ctx context.Context) error {
		attemptCount++
		if attemptCount == 2 {
			cancel() // Cancel context on second attempt
		}
		return errors.New("error")
	}

	// Act
	err := retry.Do(ctx, config, fn)

	// Assert
	assert.Error(t, err)
	assert.True(t, attemptCount <= 3) // Should stop soon after cancellation
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	// Arrange
	ctx := context.Background()
	config := retry.Config{
		MaxAttempts: 4,
		InitialDelay: time.Millisecond * 10,
		MaxDelay: time.Millisecond * 100,
		Multiplier: 2.0,
	}
	attemptTimes := []time.Time{}

	fn := func(ctx context.Context) error {
		attemptTimes = append(attemptTimes, time.Now())
		return errors.New("error")
	}

	// Act
	retry.Do(ctx, config, fn)

	// Assert
	assert.Len(t, attemptTimes, 4)

	// Check delays are increasing exponentially
	// Attempt 1 -> Attempt 2: ~10ms delay
	// Attempt 2 -> Attempt 3: ~20ms delay
	// Attempt 3 -> Attempt 4: ~40ms delay
	if len(attemptTimes) >= 2 {
		delay1 := attemptTimes[1].Sub(attemptTimes[0])
		assert.True(t, delay1 >= time.Millisecond*8) // Allow some variance
	}
	if len(attemptTimes) >= 3 {
		delay2 := attemptTimes[2].Sub(attemptTimes[1])
		assert.True(t, delay2 >= time.Millisecond*15)
	}
}

func TestRetry_DefaultConfig(t *testing.T) {
	// Arrange
	config := retry.DefaultConfig()

	// Assert
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, time.Second, config.InitialDelay)
	assert.Equal(t, time.Second*30, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
}
