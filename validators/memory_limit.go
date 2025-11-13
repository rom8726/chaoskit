package validators

import (
	"context"
	"fmt"
	"log/slog"
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

	// Warn if approaching limit (80% threshold)
	if memStats.Alloc > m.limitBytes*9/10 {
		chaoskit.GetLogger(ctx).Warn("memory usage approaching limit",
			slog.String("validator", m.name),
			slog.Uint64("allocated_bytes", memStats.Alloc),
			slog.Uint64("limit_bytes", m.limitBytes))
	}

	if memStats.Alloc > m.limitBytes {
		err := fmt.Errorf("memory limit exceeded: %d bytes (limit: %d bytes)",
			memStats.Alloc, m.limitBytes)
		chaoskit.GetLogger(ctx).Error("memory limit validator failed",
			slog.String("validator", m.name),
			slog.Uint64("allocated_bytes", memStats.Alloc),
			slog.Uint64("limit_bytes", m.limitBytes),
			slog.String("error", err.Error()))

		return err
	}

	chaoskit.GetLogger(ctx).Debug("memory limit validator passed",
		slog.String("validator", m.name),
		slog.Uint64("allocated_bytes", memStats.Alloc),
		slog.Uint64("limit_bytes", m.limitBytes))

	return nil
}
