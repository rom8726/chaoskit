package validators

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/rom8726/chaoskit"
)

// PanicRecoveryValidator ensures panics are recovered
type PanicRecoveryValidator struct {
	name       string
	panicCount int
	maxPanics  int
	mu         sync.Mutex
}

// NoPanics creates a panic recovery validator
func NoPanics(maxPanics int) *PanicRecoveryValidator {
	return &PanicRecoveryValidator{
		name:      fmt.Sprintf("no_panics_%d", maxPanics),
		maxPanics: maxPanics,
	}
}

func (p *PanicRecoveryValidator) Name() string {
	return p.name
}

func (p *PanicRecoveryValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Warn if approaching limit (80% threshold)
	if p.panicCount > int(float64(p.maxPanics)*0.8) {
		slog.Warn("panic count approaching limit",
			slog.String("validator", p.name),
			slog.Int("current", p.panicCount),
			slog.Int("limit", p.maxPanics))
	}

	if p.panicCount > p.maxPanics {
		err := fmt.Errorf("too many panics: %d (limit: %d)", p.panicCount, p.maxPanics)
		slog.Error("panic recovery validator failed",
			slog.String("validator", p.name),
			slog.Int("panic_count", p.panicCount),
			slog.Int("limit", p.maxPanics),
			slog.String("error", err.Error()))

		return err
	}

	slog.Debug("panic recovery validator passed",
		slog.String("validator", p.name),
		slog.Int("panic_count", p.panicCount),
		slog.Int("limit", p.maxPanics))

	return nil
}

// RecordPanic records a panic occurrence
func (p *PanicRecoveryValidator) RecordPanic() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.panicCount++
	slog.Debug("panic recorded",
		slog.String("validator", p.name),
		slog.Int("total_panics", p.panicCount))
}
