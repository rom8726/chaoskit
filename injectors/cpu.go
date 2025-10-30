package injectors

import (
	"context"
	"fmt"
	"sync"
)

// CPUStressInjector creates CPU load
type CPUStressInjector struct {
	name    string
	workers int
	mu      sync.Mutex
	stopCh  chan struct{}
	stopped bool
	wg      sync.WaitGroup
}

// CPUStress creates a CPU stress injector
func CPUStress(workers int) *CPUStressInjector {
	return &CPUStressInjector{
		name:    fmt.Sprintf("cpu_stress_%d", workers),
		workers: workers,
		stopCh:  make(chan struct{}),
	}
}

func (c *CPUStressInjector) Name() string {
	return c.name
}

func (c *CPUStressInjector) Inject(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Start CPU stress workers
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func(id int) {
			defer c.wg.Done()
			c.stressWorker(id)
		}(i)
	}

	fmt.Printf("[CHAOS] CPU stress started with %d workers\n", c.workers)

	return nil
}

func (c *CPUStressInjector) stressWorker(id int) {
	for {
		select {
		case <-c.stopCh:
			return
		default:
			// Busy loop to create CPU load
			_ = 0
		}
	}
}

func (c *CPUStressInjector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
		c.wg.Wait()
	}

	return nil
}
