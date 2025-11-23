//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// ContextCancellationInjector is a no-op implementation used without the chaos tag.
type ContextCancellationInjector struct {
	name        string
	probability float64
}

// NewContextCancellationInjector returns a stub that matches the real API.
func NewContextCancellationInjector(probability float64) *ContextCancellationInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return &ContextCancellationInjector{
		name:        fmt.Sprintf("context_cancellation_%.2f", probability),
		probability: probability,
	}
}

// Name returns the configured name.
func (c *ContextCancellationInjector) Name() string {
	return c.name
}

// Inject is a no-op.
func (*ContextCancellationInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ContextCancellationInjector) Stop(context.Context) error {
	return nil
}

// GetChaosContext returns the parent context unchanged and a no-op cancel function.
func (*ContextCancellationInjector) GetChaosContext(parent context.Context) (context.Context, context.CancelFunc) {
	return parent, func() {}
}

// GetCancellationProbability exposes the configured probability.
func (c *ContextCancellationInjector) GetCancellationProbability() float64 {
	return c.probability
}

// GetCancelCount always returns zero for the stub implementation.
func (*ContextCancellationInjector) GetCancelCount() int64 {
	return 0
}

// Type reports the injector category.
func (*ContextCancellationInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext
}

// GetMetrics reports stub metrics for parity with the real injector.
func (c *ContextCancellationInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"probability":          c.probability,
		"total_cancellations":  int64(0),
		"active_cancellations": 0,
		"stopped":              false,
	}
}
