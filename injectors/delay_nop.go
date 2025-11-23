//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	"github.com/rom8726/chaoskit"
)

// DelayMode defines how delays would be injected in real builds.
type DelayMode int

const (
	// ProbabilityMode mirrors the probability-based delay behaviour.
	ProbabilityMode DelayMode = iota
	// IntervalMode mirrors the interval-based delay behaviour.
	IntervalMode
)

// DelayInjector is a no-op placeholder used without the chaos tag.
type DelayInjector struct {
	name        string
	mode        DelayMode
	minDelay    time.Duration
	maxDelay    time.Duration
	interval    time.Duration
	probability float64
}

// RandomDelay creates a stubbed probability-based delay injector.
func RandomDelay(min, max time.Duration) *DelayInjector {
	return &DelayInjector{
		name:     fmt.Sprintf("delay_injector_prob_%v_%v", min, max),
		minDelay: min,
		maxDelay: max,
		mode:     ProbabilityMode,
	}
}

// RandomDelayWithProbability creates a stubbed probability-based delay injector.
func RandomDelayWithProbability(min, max time.Duration, probability float64) *DelayInjector {
	if probability < 0 {
		probability = 0
	}
	if probability > 1 {
		probability = 1
	}

	return &DelayInjector{
		name:        fmt.Sprintf("delay_injector_prob_%v_%v_%.2f", min, max, probability),
		minDelay:    min,
		maxDelay:    max,
		probability: probability,
		mode:        ProbabilityMode,
	}
}

// RandomDelayWithInterval creates an interval-mode stub injector.
func RandomDelayWithInterval(min, max, interval time.Duration) *DelayInjector {
	return &DelayInjector{
		name:     fmt.Sprintf("delay_injector_interval_%v_%v_%v", min, max, interval),
		minDelay: min,
		maxDelay: max,
		interval: interval,
		mode:     IntervalMode,
	}
}

// Name returns the configured name.
func (d *DelayInjector) Name() string {
	return d.name
}

// Inject is a no-op.
func (*DelayInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*DelayInjector) Stop(context.Context) error {
	return nil
}

// GetDelayCount always reports zero in stub builds.
func (*DelayInjector) GetDelayCount() int64 {
	return 0
}

// BeforeStep is a no-op.
func (*DelayInjector) BeforeStep(context.Context) error {
	return nil
}

// AfterStep is a no-op.
func (*DelayInjector) AfterStep(context.Context, error) error {
	return nil
}

// GetChaosDelay reports that no delay should be applied.
func (*DelayInjector) GetChaosDelay(context.Context) (time.Duration, bool) {
	return 0, false
}

// Type reports the injector classification.
func (d *DelayInjector) Type() chaoskit.InjectorType {
	if d.mode == IntervalMode {
		return chaoskit.InjectorTypeHybrid
	}

	return chaoskit.InjectorTypeContext
}

// GetMetrics returns stub metrics that mirror the real structure.
func (d *DelayInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"mode":        d.mode.String(),
		"min_delay":   d.minDelay.String(),
		"max_delay":   d.maxDelay.String(),
		"probability": d.probability,
		"interval":    d.interval.String(),
		"delay_count": int64(0),
		"stopped":     false,
	}
}

// String returns a descriptive representation of the delay mode.
func (m DelayMode) String() string {
	switch m {
	case ProbabilityMode:
		return "probability"
	case IntervalMode:
		return "interval"
	default:
		return "unknown"
	}
}
