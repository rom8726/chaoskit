package validators

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

type InfiniteLoopValidator struct {
	name          string
	timeout       time.Duration
	lastCheckTime time.Time
	mu            sync.Mutex
}

// NoInfiniteLoop creates an infinite loop validator
func NoInfiniteLoop(timeout time.Duration) *InfiniteLoopValidator {
	return &InfiniteLoopValidator{
		name:          fmt.Sprintf("no_infinite_loop_%v", timeout),
		timeout:       timeout,
		lastCheckTime: time.Now(),
	}
}

func (i *InfiniteLoopValidator) Name() string {
	return i.name
}

func (i *InfiniteLoopValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityCritical
}

func (i *InfiniteLoopValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if the validator itself is stuck
	elapsed := time.Since(i.lastCheckTime)

	// Warn if approaching timeout (80% threshold)
	if elapsed > i.timeout*9/10 {
		chaoskit.GetLogger(ctx).Warn("possible infinite loop approaching timeout",
			slog.String("validator", i.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("timeout", i.timeout))
	}

	if elapsed > i.timeout {
		err := fmt.Errorf("possible infinite loop detected: no progress for %v", elapsed)
		chaoskit.GetLogger(ctx).Error("infinite loop validator failed",
			slog.String("validator", i.name),
			slog.Duration("elapsed", elapsed),
			slog.Duration("timeout", i.timeout),
			slog.String("error", err.Error()))

		return err
	}

	chaoskit.GetLogger(ctx).Debug("infinite loop validator passed",
		slog.String("validator", i.name),
		slog.Duration("elapsed", elapsed))

	i.lastCheckTime = time.Now()

	return nil
}
