//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchValueCorruptionInjector is a no-op placeholder compiled without the chaos tag.
type MonkeyPatchValueCorruptionInjector struct {
	name    string
	targets []ValueCorruptionPatchTarget
}

// ValueCorruptionPatchTarget mirrors the real configuration structure.
type ValueCorruptionPatchTarget struct {
	Func        interface{}
	CorruptFunc interface{}
	Probability float64
	FuncName    string
}

// MonkeyPatchValueCorruption returns a stub injector with the same naming semantics.
func MonkeyPatchValueCorruption(targets []ValueCorruptionPatchTarget) *MonkeyPatchValueCorruptionInjector {
	if targets == nil {
		targets = []ValueCorruptionPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_value_corruption_%d_targets", len(targets))

	return &MonkeyPatchValueCorruptionInjector{
		name:    name,
		targets: targets,
	}
}

// Name returns the configured name.
func (m *MonkeyPatchValueCorruptionInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MonkeyPatchValueCorruptionInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MonkeyPatchValueCorruptionInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MonkeyPatchValueCorruptionInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (m *MonkeyPatchValueCorruptionInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_targets":     len(m.targets),
		"active_patches":    0,
		"total_corruptions": int64(0),
		"stopped":           false,
	}
}

// GetCorruptionCount returns zero in stub builds.
func (*MonkeyPatchValueCorruptionInjector) GetCorruptionCount() int64 {
	return 0
}
