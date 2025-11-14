package validators

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

type SlowIterationValidator struct {
	name          string
	timeout       time.Duration
	lastCheckTime time.Time
	mu            sync.Mutex
}

// NoSlowIteration creates a slow iteration validator
func NoSlowIteration(timeout time.Duration) *SlowIterationValidator {
	return &SlowIterationValidator{
		name:          fmt.Sprintf("slow_iteration_%v", timeout),
		timeout:       timeout,
		lastCheckTime: time.Now(),
	}
}

func (i *SlowIterationValidator) Name() string {
	return i.name
}

func (i *SlowIterationValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityCritical
}

func (i *SlowIterationValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if the validator itself is stuck
	elapsed := time.Since(i.lastCheckTime)

	// Warn if approaching timeout (80% threshold)
	if elapsed > i.timeout*9/10 {
		chaoskit.GetLogger(ctx).Warn("possible long-processing loop approaching timeout",
			slog.String("validator", i.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("timeout", i.timeout))
	}

	if elapsed > i.timeout {
		err := fmt.Errorf("possible long-processing loop detected: no progress for %v", elapsed)
		chaoskit.GetLogger(ctx).Error("long-processing loop validator failed",
			slog.String("validator", i.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("timeout", i.timeout),
			slog.String("error", err.Error()))

		return err
	}

	chaoskit.GetLogger(ctx).Debug("long-processing loop validator passed",
		slog.String("validator", i.name),
		slog.Duration("elapsed", elapsed))

	i.lastCheckTime = time.Now()

	return nil
}
