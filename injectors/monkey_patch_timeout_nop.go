//go:build !chaos

package injectors

import (
	"context"
	"fmt"
	"time"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchTimeoutInjector is a no-op placeholder compiled without the chaos tag.
type MonkeyPatchTimeoutInjector struct {
	name    string
	targets []TimeoutPatchTarget
}

// TimeoutPatchTarget mirrors the real configuration structure.
type TimeoutPatchTarget struct {
	Func        interface{}
	Timeout     time.Duration
	Probability float64
	FuncName    string
	ReturnError error
}

// MonkeyPatchTimeout returns a stub injector with the same naming semantics.
func MonkeyPatchTimeout(targets []TimeoutPatchTarget) *MonkeyPatchTimeoutInjector {
	if targets == nil {
		targets = []TimeoutPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_timeout_%d_targets", len(targets))

	return &MonkeyPatchTimeoutInjector{
		name:    name,
		targets: targets,
	}
}

// Name returns the configured name.
func (m *MonkeyPatchTimeoutInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MonkeyPatchTimeoutInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MonkeyPatchTimeoutInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MonkeyPatchTimeoutInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (m *MonkeyPatchTimeoutInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": 0,
		"total_timeouts": int64(0),
		"stopped":        false,
	}
}

// GetTimeoutCount returns zero in stub builds.
func (*MonkeyPatchTimeoutInjector) GetTimeoutCount() int64 {
	return 0
}
