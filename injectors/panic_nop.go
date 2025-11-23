//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// PanicInjector is a no-op placeholder compiled without the chaos tag.
type PanicInjector struct {
	name        string
	probability float64
}

// PanicProbability returns a stub panic injector.
func PanicProbability(probability float64) *PanicInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return &PanicInjector{
		name:        fmt.Sprintf("panic_injector_%.2f", probability),
		probability: probability,
	}
}

// Name returns the configured name.
func (p *PanicInjector) Name() string {
	return p.name
}

// Inject is a no-op.
func (*PanicInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*PanicInjector) Stop(context.Context) error {
	return nil
}

// ShouldChaosPanic always returns false in stub builds.
func (*PanicInjector) ShouldChaosPanic() bool {
	return false
}

// GetPanicProbability exposes the configured probability.
func (p *PanicInjector) GetPanicProbability() float64 {
	return p.probability
}

// Type reports the injector classification.
func (*PanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext
}

// GetMetrics returns stub metrics matching the real structure.
func (p *PanicInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"probability": p.probability,
		"stopped":     false,
	}
}
