package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// DelayInjector introduces random delays
type DelayInjector struct {
	name     string
	minDelay time.Duration
	maxDelay time.Duration
	mu       sync.Mutex
	stopCh   chan struct{}
	stopped  bool
}

// RandomDelay creates a delay injector with random delays
func RandomDelay(min, max time.Duration) *DelayInjector {
	return &DelayInjector{
		name:     fmt.Sprintf("delay_injector_%v_%v", min, max),
		minDelay: min,
		maxDelay: max,
		stopCh:   make(chan struct{}),
	}
}

func (d *DelayInjector) Name() string {
	return d.name
}

func (d *DelayInjector) Inject(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return fmt.Errorf("injector already stopped")
	}

	fmt.Printf("[CHAOS] Delay injector started (range: %v-%v)\n", d.minDelay, d.maxDelay)

	return nil
}

func (d *DelayInjector) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.stopped {
		close(d.stopCh)
		d.stopped = true
	}

	return nil
}

// InjectDelay returns a random delay within the configured range
func (d *DelayInjector) InjectDelay() time.Duration {
	if d.stopped {
		return 0
	}

	delta := d.maxDelay - d.minDelay
	delay := d.minDelay + time.Duration(rand.Int63n(int64(delta)))

	return delay
}
