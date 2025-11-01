package validators

import (
	"context"
	"fmt"
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

func (g *GoroutineLeakValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	current := runtime.NumGoroutine()

	if !g.initialized {
		g.baselineGCount = current
		g.initialized = true

		return nil
	}

	if current > g.maxGoroutines {
		return fmt.Errorf("goroutine leak detected: %d goroutines (limit: %d, baseline: %d)",
			current, g.maxGoroutines, g.baselineGCount)
	}

	return nil
}
