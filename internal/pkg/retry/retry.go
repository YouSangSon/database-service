package retry

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	// ErrMaxRetriesExceeded는 최대 재시도 횟수를 초과했을 때 발생합니다
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")
)

// Config는 재시도 설정입니다
type Config struct {
	MaxAttempts     int           // 최대 시도 횟수
	InitialInterval time.Duration // 초기 대기 시간
	MaxInterval     time.Duration // 최대 대기 시간
	Multiplier      float64       // 대기 시간 증가 배율
	MaxElapsedTime  time.Duration // 최대 재시도 시간
}

// DefaultConfig는 기본 재시도 설정입니다
func DefaultConfig() Config {
	return Config{
		MaxAttempts:     3,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		MaxElapsedTime:  2 * time.Minute,
	}
}

// RetryableFunc는 재시도 가능한 함수입니다
type RetryableFunc func(ctx context.Context) error

// Do는 함수를 재시도합니다
func Do(ctx context.Context, cfg Config, fn RetryableFunc) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	startTime := time.Now()
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// 컨텍스트가 취소되었는지 확인
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 함수 실행
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// 재시도 가능한 에러인지 확인
		if !IsRetryable(lastErr) {
			return lastErr
		}

		// 최대 시도 횟수에 도달했는지 확인
		if attempt >= cfg.MaxAttempts {
			break
		}

		// 대기 시간 계산 (exponential backoff with jitter)
		waitTime := calculateBackoff(cfg, attempt)

		// 최대 재시도 시간 확인
		if cfg.MaxElapsedTime > 0 {
			elapsed := time.Since(startTime)
			if elapsed+waitTime > cfg.MaxElapsedTime {
				break
			}
		}

		// 대기
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	if lastErr != nil {
		return errors.Join(ErrMaxRetriesExceeded, lastErr)
	}

	return ErrMaxRetriesExceeded
}

// calculateBackoff은 exponential backoff를 계산합니다
func calculateBackoff(cfg Config, attempt int) time.Duration {
	backoff := float64(cfg.InitialInterval) * math.Pow(cfg.Multiplier, float64(attempt-1))

	if backoff > float64(cfg.MaxInterval) {
		backoff = float64(cfg.MaxInterval)
	}

	return time.Duration(backoff)
}

// IsRetryable은 에러가 재시도 가능한지 확인합니다
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 여기에 재시도 가능한 에러 타입을 추가합니다
	// 예: 네트워크 에러, 타임아웃, 일시적인 데이터베이스 에러 등

	switch err.Error() {
	case "connection refused", "connection reset", "timeout":
		return true
	}

	return false
}

// DoWithValue는 값을 반환하는 함수를 재시도합니다
func DoWithValue[T any](ctx context.Context, cfg Config, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	err := Do(ctx, cfg, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		lastErr = err
		return err
	})

	if err != nil {
		return result, lastErr
	}

	return result, nil
}
