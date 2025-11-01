package validators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

type InfiniteLoopValidator struct {
	name           string
	timeout        time.Duration
	lastCheckTime  time.Time
	lastCheckValue any
	mu             sync.Mutex
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

func (i *InfiniteLoopValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check if the validator itself is stuck
	elapsed := time.Since(i.lastCheckTime)
	if elapsed > i.timeout {
		return fmt.Errorf("possible infinite loop detected: no progress for %v", elapsed)
	}

	i.lastCheckTime = time.Now()

	return nil
}
