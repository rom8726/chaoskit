//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// CPUStressInjector is a no-op placeholder compiled without the chaos tag.
type CPUStressInjector struct {
	name    string
	workers int
}

// CPUStress returns a stub injector that preserves the expected API surface.
func CPUStress(workers int) *CPUStressInjector {
	return &CPUStressInjector{
		name:    fmt.Sprintf("cpu_stress_%d", workers),
		workers: workers,
	}
}

// Name returns the configured name.
func (c *CPUStressInjector) Name() string {
	return c.name
}

// Inject is a no-op.
func (*CPUStressInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*CPUStressInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector category.
func (*CPUStressInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// IsGlobal indicates the stub is considered global for interface compatibility.
func (*CPUStressInjector) IsGlobal() bool {
	return true
}

// GetMetrics returns minimal stub metrics.
func (c *CPUStressInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"workers": c.workers,
		"stopped": false,
	}
}
