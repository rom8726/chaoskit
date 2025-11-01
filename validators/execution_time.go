package validators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// ExecutionTimeValidator validates execution time bounds
type ExecutionTimeValidator struct {
	name        string
	minDuration time.Duration
	maxDuration time.Duration
	startTime   time.Time
	mu          sync.Mutex
}

// ExecutionTime creates an execution time validator
func ExecutionTime(min, max time.Duration) *ExecutionTimeValidator {
	return &ExecutionTimeValidator{
		name:        fmt.Sprintf("execution_time_%v_%v", min, max),
		minDuration: min,
		maxDuration: max,
		startTime:   time.Now(),
	}
}

func (e *ExecutionTimeValidator) Name() string {
	return e.name
}

func (e *ExecutionTimeValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	elapsed := time.Since(e.startTime)

	if elapsed < e.minDuration {
		return fmt.Errorf("execution too fast: %v (min: %v)", elapsed, e.minDuration)
	}

	if elapsed > e.maxDuration {
		return fmt.Errorf("execution too slow: %v (max: %v)", elapsed, e.maxDuration)
	}

	return nil
}
