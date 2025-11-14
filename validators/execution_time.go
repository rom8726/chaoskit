package validators

import (
	"context"
	"fmt"
	"log/slog"
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

func (e *ExecutionTimeValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityWarning
}

func (e *ExecutionTimeValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	elapsed := time.Since(e.startTime)

	if elapsed < e.minDuration {
		err := fmt.Errorf("execution too fast: %v (min: %v)", elapsed, e.minDuration)
		chaoskit.GetLogger(ctx).Error("execution time validator failed",
			slog.String("validator", e.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("min_duration", e.minDuration),
			slog.String("error", err.Error()))

		return err
	}

	if elapsed > e.maxDuration {
		err := fmt.Errorf("execution too slow: %v (max: %v)", elapsed, e.maxDuration)
		chaoskit.GetLogger(ctx).Error("execution time validator failed",
			slog.String("validator", e.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("max_duration", e.maxDuration),
			slog.String("error", err.Error()))

		return err
	}

	// Warn if approaching limits
	if elapsed > e.maxDuration*9/10 {
		chaoskit.GetLogger(ctx).Warn("execution time approaching max limit",
			slog.String("validator", e.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("max_duration", e.maxDuration))
	}

	chaoskit.GetLogger(ctx).Debug("execution time validator passed",
		slog.String("validator", e.name),
		slog.Duration("elapsed", elapsed),
		slog.Duration("min_duration", e.minDuration),
		slog.Duration("max_duration", e.maxDuration))

	return nil
}
