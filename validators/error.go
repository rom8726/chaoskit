package validators

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/rom8726/chaoskit"
)

// ErrorValidator ensures errors don't exceed the limit
type ErrorValidator struct {
	name       string
	errorCount int
	maxErrors  int
	mu         sync.Mutex
}

// MaxErrors creates an error validator
func MaxErrors(maxErrors int) *ErrorValidator {
	return &ErrorValidator{
		name:      fmt.Sprintf("max_errors_%d", maxErrors),
		maxErrors: maxErrors,
	}
}

func (e *ErrorValidator) Name() string {
	return e.name
}

func (e *ErrorValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityWarning
}

func (e *ErrorValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Warn if approaching limit (80% threshold)
	if e.errorCount > int(float64(e.maxErrors)*0.8) {
		chaoskit.GetLogger(ctx).Warn("error count approaching limit",
			slog.String("validator", e.name),
			slog.Int("current", e.errorCount),
			slog.Int("limit", e.maxErrors))
	}

	if e.errorCount > e.maxErrors {
		err := fmt.Errorf("too many errors: %d (limit: %d)", e.errorCount, e.maxErrors)
		chaoskit.GetLogger(ctx).Error("error validator failed",
			slog.String("validator", e.name),
			slog.Int("error_count", e.errorCount),
			slog.Int("limit", e.maxErrors),
			slog.String("error", err.Error()))

		return err
	}

	chaoskit.GetLogger(ctx).Debug("error validator passed",
		slog.String("validator", e.name),
		slog.Int("error_count", e.errorCount),
		slog.Int("limit", e.maxErrors))

	return nil
}

// RecordError records an error occurrence
func (e *ErrorValidator) RecordError(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errorCount++
	chaoskit.GetLogger(ctx).Debug("error recorded",
		slog.String("validator", e.name),
		slog.Int("total_errors", e.errorCount))
}
