//go:build !chaos

package injectors

import (
	"context"

	"github.com/rom8726/chaoskit"
)

// ErrorInjector is a no-op placeholder compiled without the chaos tag.
type ErrorInjector struct {
	name        string
	probability float64
	errorMsg    string
}

// ErrorWithProbability returns a stub that matches the real constructor.
func ErrorWithProbability(errorMsg string, probability float64) *ErrorInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return &ErrorInjector{
		name:        "error_injector",
		probability: probability,
		errorMsg:    errorMsg,
	}
}

// Name returns the configured name.
func (e *ErrorInjector) Name() string {
	return e.name
}

// Inject is a no-op.
func (*ErrorInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*ErrorInjector) Stop(context.Context) error {
	return nil
}

// BeforeStep is a no-op.
func (*ErrorInjector) BeforeStep(context.Context) error {
	return nil
}

// AfterStep is a no-op.
func (*ErrorInjector) AfterStep(context.Context, error) error {
	return nil
}

// Type reports the injector classification.
func (*ErrorInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeContext
}

// GetMetrics returns stub metrics to mirror the real structure.
func (e *ErrorInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"probability": e.probability,
		"error_count": int64(0),
		"stopped":     false,
	}
}

// ShouldReturnError never triggers in stub builds.
func (*ErrorInjector) ShouldReturnError() error {
	return nil
}
