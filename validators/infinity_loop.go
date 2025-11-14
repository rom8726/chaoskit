package validators

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// InfiniteLoopValidator detects infinite loops by applying a timeout to step execution
type InfiniteLoopValidator struct {
	name            string
	timeout         time.Duration
	mu              sync.Mutex
	detectionsCount int64
}

// NoInfiniteLoop creates an infinite loop detector validator
func NoInfiniteLoop(timeout time.Duration) *InfiniteLoopValidator {
	return &InfiniteLoopValidator{
		name:    fmt.Sprintf("no_infinite_loop_%v", timeout),
		timeout: timeout,
	}
}

func (v *InfiniteLoopValidator) Name() string {
	return v.name
}

func (v *InfiniteLoopValidator) Severity() chaoskit.ValidationSeverity {
	return chaoskit.SeverityCritical
}

// Validate is called after each iteration - no validation here,
// as actual detection happens at executor level through WrapStep
func (v *InfiniteLoopValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	return nil
}

// WrapStep wraps a step in a goroutine with timeout
// This allows detecting hung steps that exceed the timeout
func (v *InfiniteLoopValidator) WrapStep(
	step chaoskit.Step,
) func(ctx context.Context, target chaoskit.Target) error {
	return func(ctx context.Context, target chaoskit.Target) error {
		// Channel for result
		done := make(chan error, 1)

		// Context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, v.timeout)
		defer cancel()

		// Launch step in a separate goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					select {
					case done <- fmt.Errorf("panic in step: %v", r):
					case <-timeoutCtx.Done():
						// Timeout already occurred, goroutine will be abandoned
					}
				}
			}()

			// Execute step with timeout context
			err := step.Execute(timeoutCtx, target)
			select {
			case done <- err:
			case <-timeoutCtx.Done():
				// Timeout already occurred, result is no longer needed
			}
		}()

		// Wait for either completion or timeout
		select {
		case err := <-done:
			// Step completed successfully within timeout
			return err

		case <-timeoutCtx.Done():
			// Timeout expired, step hasn't completed yet
			// Check if it was actually a timeout (not context cancellation)
			if !errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
				// Context was cancelled, not a timeout
				return timeoutCtx.Err()
			}

			v.mu.Lock()
			v.detectionsCount++
			detections := v.detectionsCount
			v.mu.Unlock()

			chaoskit.GetLogger(ctx).Error("infinite loop detected",
				slog.String("validator", v.name),
				slog.String("step", step.Name()),
				slog.Duration("timeout", v.timeout),
				slog.Int64("total_detections", detections))

			// Wait a bit to see if goroutine completes (graceful shutdown)
			select {
			case <-done:
				// Goroutine completed after timeout - still consider it a detection
				chaoskit.GetLogger(ctx).Debug("step completed after timeout",
					slog.String("validator", v.name),
					slog.String("step", step.Name()))
			case <-time.After(100 * time.Millisecond):
				// Goroutine is still hanging - it will remain as a zombie goroutine
				chaoskit.GetLogger(ctx).Warn("step goroutine still running after timeout",
					slog.String("validator", v.name),
					slog.String("step", step.Name()))
			}

			return fmt.Errorf("infinite loop detected in step %s: exceeded timeout %v",
				step.Name(), v.timeout)
		}
	}
}

// GetDetectionsCount returns the number of detected infinite loops
func (v *InfiniteLoopValidator) GetDetectionsCount() int64 {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.detectionsCount
}
