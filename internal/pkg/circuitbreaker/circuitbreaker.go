package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen은 circuit이 열려있을 때 발생합니다
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrTooManyRequests는 half-open 상태에서 요청이 많을 때 발생합니다
	ErrTooManyRequests = errors.New("too many requests")
)

// State는 circuit breaker의 상태입니다
type State int

const (
	// StateClosed는 정상 상태입니다
	StateClosed State = iota
	// StateHalfOpen은 복구 시도 중인 상태입니다
	StateHalfOpen
	// StateOpen은 차단된 상태입니다
	StateOpen
)

// Config는 circuit breaker 설정입니다
type Config struct {
	MaxRequests  uint32        // half-open 상태에서 허용할 최대 요청 수
	Interval     time.Duration // 통계를 리셋할 간격
	Timeout      time.Duration // open 상태를 유지할 시간
	ReadyToTrip  func(counts Counts) bool
	OnStateChange func(name string, from State, to State)
}

// Counts는 circuit breaker의 통계입니다
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreaker는 circuit breaker 구현입니다
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	counts     Counts
	expiry     time.Time
}

// NewCircuitBreaker는 새로운 circuit breaker를 생성합니다
func NewCircuitBreaker(name string, cfg Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   cfg.MaxRequests,
		interval:      cfg.Interval,
		timeout:       cfg.Timeout,
		readyToTrip:   cfg.ReadyToTrip,
		onStateChange: cfg.OnStateChange,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}
	if cb.interval <= 0 {
		cb.interval = time.Duration(0)
	}
	if cb.timeout <= 0 {
		cb.timeout = 60 * time.Second
	}
	if cb.readyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	}

	cb.toNewGeneration(time.Now())
	return cb
}

// Execute는 함수를 circuit breaker로 감싸서 실행합니다
func (cb *CircuitBreaker) Execute(ctx context.Context, req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)

	return result, err
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrCircuitOpen
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.Requests++
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, currentGeneration := cb.currentState(now)
	if generation != currentGeneration {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
	case StateHalfOpen:
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// State는 현재 상태를 반환합니다
func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts는 현재 통계를 반환합니다
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}
