//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchPanicInjector is a no-op placeholder compiled without the chaos tag.
type MonkeyPatchPanicInjector struct {
	name    string
	targets []PatchTarget
}

// PatchTarget mirrors the real configuration structure.
type PatchTarget struct {
	Func         interface{}
	Probability  float64
	PanicMessage string
	FuncName     string
}

// MonkeyPatchPanic returns a stub injector with the same naming semantics.
func MonkeyPatchPanic(targets []PatchTarget) *MonkeyPatchPanicInjector {
	if targets == nil {
		targets = []PatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_panic_%d_targets", len(targets))

	return &MonkeyPatchPanicInjector{
		name:    name,
		targets: targets,
	}
}

// Name returns the configured name.
func (m *MonkeyPatchPanicInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MonkeyPatchPanicInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MonkeyPatchPanicInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MonkeyPatchPanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (m *MonkeyPatchPanicInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": 0,
		"stopped":        false,
	}
}
