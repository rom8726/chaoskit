package validators

import (
	"context"
	"fmt"
	"runtime"

	"github.com/rom8726/chaoskit"
)

// MemoryLimitValidator checks memory usage
type MemoryLimitValidator struct {
	name       string
	limitBytes uint64
}

// MemoryUnderLimit creates a memory limit validator
func MemoryUnderLimit(limitBytes uint64) *MemoryLimitValidator {
	return &MemoryLimitValidator{
		name:       fmt.Sprintf("memory_under_%dMB", limitBytes/(1024*1024)),
		limitBytes: limitBytes,
	}
}

func (m *MemoryLimitValidator) Name() string {
	return m.name
}

func (m *MemoryLimitValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	if memStats.Alloc > m.limitBytes {
		return fmt.Errorf("memory limit exceeded: %d bytes (limit: %d bytes)",
			memStats.Alloc, m.limitBytes)
	}

	return nil
}
