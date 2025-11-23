//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchDelayInjector is a no-op placeholder compiled without the chaos tag.
type MonkeyPatchDelayInjector struct {
	name    string
	targets []DelayPatchTarget
}

// DelayPatchTarget mirrors the real configuration structure.
type DelayPatchTarget struct {
	Func        interface{}
	Probability float64
	MinDelay    time.Duration
	MaxDelay    time.Duration
	DelayBefore bool
	FuncName    string
}

// MonkeyPatchDelay returns a stub injector with the same naming semantics.
func MonkeyPatchDelay(targets []DelayPatchTarget) *MonkeyPatchDelayInjector {
	if targets == nil {
		targets = []DelayPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_delay_%d_targets", len(targets))

	return &MonkeyPatchDelayInjector{
		name:    name,
		targets: targets,
	}
}

// Name returns the configured name.
func (m *MonkeyPatchDelayInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MonkeyPatchDelayInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MonkeyPatchDelayInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MonkeyPatchDelayInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (m *MonkeyPatchDelayInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": 0,
		"total_delays":   int64(0),
		"stopped":        false,
	}
}

// GetDelayCount returns zero in stub builds.
func (*MonkeyPatchDelayInjector) GetDelayCount() int64 {
	return 0
}

// GetDelayCountForTarget reports zero and false in stub builds.
func (*MonkeyPatchDelayInjector) GetDelayCountForTarget(interface{}) (int64, bool) {
	return 0, false
}
