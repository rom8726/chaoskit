//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	"github.com/rom8726/chaoskit"
)

// FailpointPanicInjector is a no-op placeholder compiled without the chaos tag.
type FailpointPanicInjector struct {
	name        string
	failpoints  []string
	probability float64
	interval    time.Duration
	window      time.Duration
}

// FailpointPanic creates a stubbed failpoint panic injector.
func FailpointPanic(names []string, probability float64, window time.Duration) *FailpointPanicInjector {
	return &FailpointPanicInjector{
		name:        fmt.Sprintf("failpoint_panic_%d_pts_p%.2f", len(names), probability),
		failpoints:  append([]string(nil), names...),
		probability: probability,
		interval:    window,
		window:      window,
	}
}

// Name returns the configured name.
func (f *FailpointPanicInjector) Name() string {
	return f.name
}

// Inject is a no-op.
func (*FailpointPanicInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*FailpointPanicInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*FailpointPanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// GetMetrics returns stub metrics mirroring the real structure.
func (f *FailpointPanicInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"failpoints":       f.failpoints,
		"probability":      f.probability,
		"interval":         f.interval.String(),
		"window":           f.window.String(),
		"active_count":     0,
		"total_failpoints": len(f.failpoints),
		"stopped":          false,
	}
}
