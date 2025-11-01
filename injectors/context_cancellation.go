package injectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// ContextCancellationInjector injects context cancellation with a given probability
// It creates child contexts with cancel functions and randomly cancels them
type ContextCancellationInjector struct {
	name          string
	probability   float64
	mu            sync.Mutex
	stopped       bool
	cancelCount   int64
	cancellations map[context.Context]context.CancelFunc // track active cancellations
}

// NewContextCancellationInjector creates a new context cancellation injector
func NewContextCancellationInjector(probability float64) *ContextCancellationInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	name := fmt.Sprintf("context_cancellation_%.2f", probability)

	return &ContextCancellationInjector{
		name:          name,
		probability:   probability,
		cancellations: make(map[context.Context]context.CancelFunc),
	}
}

func (c *ContextCancellationInjector) Name() string {
	return c.name
}

func (c *ContextCancellationInjector) Inject(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("injector already stopped")
	}

	fmt.Printf("[CHAOS] Context cancellation injector started (probability: %.2f)\n", c.probability)

	return nil
}

func (c *ContextCancellationInjector) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.stopped {
		// Cancel all tracked contexts
		for ctx, cancel := range c.cancellations {
			if ctx.Err() == nil {
				cancel()
			}
		}
		c.cancellations = make(map[context.Context]context.CancelFunc)

		c.stopped = true
		fmt.Printf("[CHAOS] Context cancellation injector stopped (total cancellations: %d)\n", c.cancelCount)
	}

	return nil
}

// GetChaosContext creates a child context with cancellation support
// Returns the child context and a cancel function
// If probability triggers, the context will be cancelled
func (c *ContextCancellationInjector) GetChaosContext(parent context.Context) (context.Context, context.CancelFunc) {
	c.mu.Lock()
	stopped := c.stopped
	probability := c.probability
	c.mu.Unlock()

	if stopped {
		// If stopped, just return parent context with no-op cancel
		return parent, func() {}
	}

	// Create child context with cancel
	childCtx, cancel := context.WithCancel(parent)

	// Track this cancellation
	c.mu.Lock()
	c.cancellations[childCtx] = cancel
	c.mu.Unlock()

	// Check probability for immediate cancellation
	if rand.Float64() < probability {
		c.mu.Lock()
		c.cancelCount++
		c.mu.Unlock()

		// Cancel immediately
		go func() {
			// Cancel after a small delay to allow context to be used
			time.Sleep(10 * time.Millisecond)
			cancel()

			fmt.Printf("[CHAOS] Context cancellation triggered (probability: %.2f)\n", probability)

			// Remove from tracking after cancellation
			c.mu.Lock()
			delete(c.cancellations, childCtx)
			c.mu.Unlock()
		}()
	}

	return childCtx, func() {
		cancel()

		// Remove from tracking
		c.mu.Lock()
		delete(c.cancellations, childCtx)
		c.mu.Unlock()
	}
}

// GetCancellationProbability returns the current cancellation probability
func (c *ContextCancellationInjector) GetCancellationProbability() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.probability
}

// GetCancelCount returns the number of cancellations applied
func (c *ContextCancellationInjector) GetCancelCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.cancelCount
}

// Type implements CategorizedInjector
func (c *ContextCancellationInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext
}

// GetMetrics implements MetricsProvider
func (c *ContextCancellationInjector) GetMetrics() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return map[string]interface{}{
		"probability":          c.probability,
		"total_cancellations":  c.cancelCount,
		"active_cancellations": len(c.cancellations),
		"stopped":              c.stopped,
	}
}
