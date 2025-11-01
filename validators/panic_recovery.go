package validators

import (
	"context"
	"fmt"
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

	if p.panicCount > p.maxPanics {
		return fmt.Errorf("too many panics: %d (limit: %d)", p.panicCount, p.maxPanics)
	}

	return nil
}

// RecordPanic records a panic occurrence
func (p *PanicRecoveryValidator) RecordPanic() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.panicCount++
}
