package injectors

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchTimeoutInjector uses monkey patching to intercept function calls
// and inject timeouts with a given probability
//
// WARNING: Monkey patching is inherently unsafe and should only be used in testing.
//
// IMPORTANT: Requires building with -gcflags=all=-l to disable inlining:
//
//	go test -gcflags=all=-l ./...
//
// Usage:
//
//	targetFunc := func() error { ... }
//	injector := injectors.MonkeyPatchTimeout([]injectors.TimeoutPatchTarget{
//	    {Func: &targetFunc, Timeout: 200*time.Millisecond, Probability: 0.3},
//	})
type MonkeyPatchTimeoutInjector struct {
	name          string
	targets       []TimeoutPatchTarget
	patchManager  *PatchManager
	timeoutCounts map[interface{}]*int64 // Map from function pointer to timeout count
	mu            sync.Mutex
	stopped       bool
}

// TimeoutPatchTarget defines a function to patch and timeout parameters
type TimeoutPatchTarget struct {
	// Func is the function to patch (must be a pointer to function)
	// Function must accept context.Context as first parameter
	// Example: var myFunc = func(ctx context.Context) error { ... }
	//          target := TimeoutPatchTarget{Func: &myFunc, ...}
	Func interface{}

	// Timeout is the duration after which function execution should be interrupted
	Timeout time.Duration

	// Probability of timeout on each call (0.0 to 1.0)
	Probability float64

	// FuncName is optional name for logging (defaults to reflect.TypeOf)
	FuncName string

	// ReturnError controls what error to return on timeout
	// If nil, returns context.DeadlineExceeded
	ReturnError error
}

// MonkeyPatchTimeout creates a new monkey patch timeout injector
func MonkeyPatchTimeout(targets []TimeoutPatchTarget) *MonkeyPatchTimeoutInjector {
	if targets == nil {
		targets = []TimeoutPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_timeout_%d_targets", len(targets))

	return &MonkeyPatchTimeoutInjector{
		name:          name,
		targets:       targets,
		patchManager:  NewPatchManager(),
		timeoutCounts: make(map[interface{}]*int64),
	}
}

func (m *MonkeyPatchTimeoutInjector) Name() string {
	return m.name
}

func (m *MonkeyPatchTimeoutInjector) Inject(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return fmt.Errorf("injector already stopped")
	}

	// Validate and prepare patches
	for i, target := range m.targets {
		if err := ValidateProbability(target.Probability); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		if target.Timeout <= 0 {
			return fmt.Errorf("invalid target %d: timeout must be positive, got %v", i, target.Timeout)
		}

		if err := ValidateFunction(target.Func); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		// Check that function accepts context.Context as first parameter
		funcType := reflect.TypeOf(target.Func).Elem()
		if funcType.NumIn() == 0 {
			return fmt.Errorf("invalid target %d: function must accept context.Context as first parameter", i)
		}

		firstParam := funcType.In(0)
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !firstParam.Implements(contextType) {
			return fmt.Errorf("invalid target %d: function first parameter must be context.Context, got %v", i, firstParam)
		}

		handle, err := CreatePatch(target.Func)
		if err != nil {
			return fmt.Errorf("failed to create patch for target %d: %w", i, err)
		}

		// Initialize timeout counter
		var count int64
		m.timeoutCounts[target.Func] = &count

		// Apply patch with timeout logic
		funcName := GetFuncName(target.Func, target.FuncName)
		probability := target.Probability
		timeout := target.Timeout
		returnError := target.ReturnError
		if returnError == nil {
			returnError = context.DeadlineExceeded
		}
		originalCopy := handle.Original
		originalType := reflect.TypeOf(originalCopy.Interface())
		originalNumOut := originalType.NumOut()

		// Build outTypes for error replacement
		outTypes := make([]reflect.Type, originalNumOut)
		for j := 0; j < originalNumOut; j++ {
			outTypes[j] = originalType.Out(j)
		}

		rng := chaoskit.GetRand(ctx) // Get deterministic generator from context
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}

		if err := ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
			if rng.Float64() < probability {
				// Apply timeout: wrap context with timeout
				ctx := args[0].Interface().(context.Context)
				timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				// Replace context in args
				newArgs := make([]reflect.Value, len(args))
				copy(newArgs, args)
				newArgs[0] = reflect.ValueOf(timeoutCtx)

				slog.Debug("monkey patch timeout triggered",
					slog.String("injector", m.name),
					slog.String("function", funcName),
					slog.Duration("timeout", timeout),
					slog.Float64("probability", probability))

				// Call original function with timeout context in a goroutine
				done := make(chan []reflect.Value, 1)

				go func() {
					done <- originalCopy.Call(newArgs)
				}()

				// Wait for either completion or timeout
				var results []reflect.Value
				select {
				case results = <-done:
					// Function completed normally
					// Check if context was cancelled during execution
					if timeoutCtx.Err() == context.DeadlineExceeded {
						// Function exceeded timeout, inject error
						m.mu.Lock()
						*m.timeoutCounts[target.Func]++
						m.mu.Unlock()

						// Replace error in results if function returns error
						for j := range results {
							if results[j].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
								// Replace error with timeout error
								errVal := reflect.ValueOf(returnError)
								results[j] = errVal

								break
							}
						}

						slog.Debug("timeout injected",
							slog.String("injector", m.name),
							slog.String("function", funcName),
							slog.String("error", returnError.Error()))
					}
				case <-timeoutCtx.Done():
					// Timeout occurred before function completed
					m.mu.Lock()
					*m.timeoutCounts[target.Func]++
					m.mu.Unlock()

					slog.Debug("timeout occurred",
						slog.String("injector", m.name),
						slog.String("function", funcName),
						slog.String("error", returnError.Error()))

					// Build return values: return zero values for all except error
					results = make([]reflect.Value, originalNumOut)
					for j := 0; j < originalNumOut-1; j++ {
						results[j] = reflect.Zero(outTypes[j])
					}
					// Set timeout error as last return value (error)
					results[originalNumOut-1] = reflect.ValueOf(returnError)
				}

				return results
			}

			// No timeout, call original function directly
			return originalCopy.Call(args)
		}); err != nil {
			m.patchManager.RollbackPatches(i)
			delete(m.timeoutCounts, target.Func)

			return fmt.Errorf("failed to apply patch %d: %w", i, err)
		}

		m.patchManager.AddPatch(handle)
		slog.Debug("monkey patch applied",
			slog.String("injector", m.name),
			slog.String("function", funcName),
			slog.Duration("timeout", timeout),
			slog.Float64("probability", probability))
	}

	slog.Info("monkey patch timeout injector started",
		slog.String("injector", m.name),
		slog.Int("targets_patched", len(m.targets)))
	slog.Warn("monkey patching requires -gcflags=all=-l for correct operation",
		slog.String("injector", m.name))

	return nil
}

func (m *MonkeyPatchTimeoutInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.patchManager.RestoreAllPatches(func(handle PatchHandle) string {
			for _, target := range m.targets {
				if target.Func == handle.Func {
					timeoutCount := int64(0)
					if countPtr, ok := m.timeoutCounts[target.Func]; ok {
						timeoutCount = *countPtr
					}
					name := GetFuncName(target.Func, target.FuncName)
					slog.Debug("monkey patch restored",
						slog.String("injector", m.name),
						slog.String("function", name),
						slog.Int64("timeouts_applied", timeoutCount))

					return name
				}
			}

			return ""
		})
		m.stopped = true
		slog.Info("monkey patch timeout injector stopped",
			slog.String("injector", m.name),
			slog.String("status", "patches restored"))
	}

	return nil
}

// Type implements CategorizedInjector
func (m *MonkeyPatchTimeoutInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics implements MetricsProvider
func (m *MonkeyPatchTimeoutInjector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalTimeouts := int64(0)
	for _, countPtr := range m.timeoutCounts {
		totalTimeouts += *countPtr
	}

	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": m.patchManager.GetActivePatchCount(),
		"total_timeouts": totalTimeouts,
		"stopped":        m.stopped,
	}
}

// GetTimeoutCount returns the total number of timeouts applied across all patches
func (m *MonkeyPatchTimeoutInjector) GetTimeoutCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := int64(0)
	for _, countPtr := range m.timeoutCounts {
		total += *countPtr
	}

	return total
}
