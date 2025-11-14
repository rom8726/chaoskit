package validators

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"github.com/rom8726/chaoskit"
)

// GoroutineLeakValidator checks for goroutine leaks
type GoroutineLeakValidator struct {
	name           string
	baselineGCount int
	maxGoroutines  int
	mu             sync.Mutex
	initialized    bool
}

// NoGoroutineLeak creates a validator that checks for goroutine leaks
func NoGoroutineLeak() *GoroutineLeakValidator {
	return &GoroutineLeakValidator{
		name:          "no_goroutine_leak",
		maxGoroutines: 1000, // Default threshold
	}
}

// GoroutineLimit creates a validator with a specific limit
func GoroutineLimit(max int) *GoroutineLeakValidator {
	return &GoroutineLeakValidator{
		name:          fmt.Sprintf("goroutine_limit_%d", max),
		maxGoroutines: max,
	}
}

func (g *GoroutineLeakValidator) Name() string {
	return g.name
}

func (g *GoroutineLeakValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityCritical
}

func (g *GoroutineLeakValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	current := runtime.NumGoroutine()

	if !g.initialized {
		g.baselineGCount = current
		g.initialized = true
		chaoskit.GetLogger(ctx).Debug("goroutine validator initialized",
			slog.String("validator", g.name),
			slog.Int("baseline_goroutines", current),
			slog.Int("max_goroutines", g.maxGoroutines))

		return nil
	}

	// Warn if approaching limit (80% threshold)
	if current > int(float64(g.maxGoroutines)*0.8) {
		chaoskit.GetLogger(ctx).Warn("goroutine count approaching limit",
			slog.String("validator", g.name),
			slog.Int("current", current),
			slog.Int("limit", g.maxGoroutines),
			slog.Int("baseline", g.baselineGCount))
	}

	if current > g.maxGoroutines {
		err := fmt.Errorf("goroutine leak detected: %d goroutines (limit: %d, baseline: %d)",
			current, g.maxGoroutines, g.baselineGCount)
		chaoskit.GetLogger(ctx).Error("goroutine validator failed",
			slog.String("validator", g.name),
			slog.Int("current", current),
			slog.Int("limit", g.maxGoroutines),
			slog.Int("baseline", g.baselineGCount),
			slog.String("error", err.Error()))

		return err
	}

	chaoskit.GetLogger(ctx).Debug("goroutine validator passed",
		slog.String("validator", g.name),
		slog.Int("current", current),
		slog.Int("limit", g.maxGoroutines))

	return nil
}
