//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchErrorInjector is a no-op placeholder compiled without the chaos tag.
type MonkeyPatchErrorInjector struct {
	name    string
	targets []ErrorPatchTarget
}

// ErrorPatchTarget mirrors the real configuration structure.
type ErrorPatchTarget struct {
	Func        interface{}
	Error       error
	ErrorFunc   func() error
	Probability float64
	FuncName    string
}

// MonkeyPatchError returns a stub injector with the same naming semantics.
func MonkeyPatchError(targets []ErrorPatchTarget) *MonkeyPatchErrorInjector {
	if targets == nil {
		targets = []ErrorPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_error_%d_targets", len(targets))

	return &MonkeyPatchErrorInjector{
		name:    name,
		targets: targets,
	}
}

// Name returns the configured name.
func (m *MonkeyPatchErrorInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MonkeyPatchErrorInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MonkeyPatchErrorInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MonkeyPatchErrorInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics returns stub metrics matching the real structure.
func (m *MonkeyPatchErrorInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": 0,
		"total_errors":   int64(0),
		"stopped":        false,
	}
}

// GetErrorCount returns zero in stub builds.
func (*MonkeyPatchErrorInjector) GetErrorCount() int64 {
	return 0
}

// GetErrorCountForTarget reports zero and false in stub builds.
func (*MonkeyPatchErrorInjector) GetErrorCountForTarget(interface{}) (int64, bool) {
	return 0, false
}
