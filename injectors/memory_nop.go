//go:build !chaos

package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// MemoryPressureInjector is a no-op placeholder compiled without the chaos tag.
type MemoryPressureInjector struct {
	name   string
	sizeMB int
}

// MemoryPressure returns a stub injector matching the real constructor.
func MemoryPressure(sizeMB int) *MemoryPressureInjector {
	return &MemoryPressureInjector{
		name:   fmt.Sprintf("memory_pressure_%dMB", sizeMB),
		sizeMB: sizeMB,
	}
}

// Name returns the configured name.
func (m *MemoryPressureInjector) Name() string {
	return m.name
}

// Inject is a no-op.
func (*MemoryPressureInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*MemoryPressureInjector) Stop(context.Context) error {
	return nil
}

// Type reports the injector classification.
func (*MemoryPressureInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeGlobal
}

// IsGlobal indicates the stub is treated as global.
func (*MemoryPressureInjector) IsGlobal() bool {
	return true
}

// GetMetrics returns stub metrics mirroring the real structure.
func (m *MemoryPressureInjector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"size_mb":  m.sizeMB,
		"stopped":  false,
		"released": true,
	}
}
