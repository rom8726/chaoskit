package injectors

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"reflect"
	"sync"

	"github.com/rom8726/chaoskit"
)

// MonkeyPatchErrorInjector uses monkey patching to intercept function calls
// and inject errors with a given probability
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
//	injector := injectors.MonkeyPatchError([]injectors.ErrorPatchTarget{
//	    {Func: &targetFunc, Error: errors.New("simulated error"), Probability: 0.1},
//	})
type MonkeyPatchErrorInjector struct {
	name         string
	targets      []ErrorPatchTarget
	patchManager *PatchManager
	errorCounts  map[interface{}]*int64 // Map from function pointer to error count
	mu           sync.Mutex
	stopped      bool
}

// ErrorPatchTarget defines a function to patch and error injection parameters
type ErrorPatchTarget struct {
	// Func is the function to patch (must be a pointer to function)
	// Function must return error as last return value
	// Example: var myFunc = func() error { ... }
	//          target := ErrorPatchTarget{Func: &myFunc, ...}
	Func interface{}

	// Error is the static error to return (mutually exclusive with ErrorFunc)
	Error error

	// ErrorFunc is a function that generates error dynamically
	// Mutually exclusive with Error
	ErrorFunc func() error

	// Probability of error injection on each call (0.0 to 1.0)
	Probability float64

	// FuncName is optional name for logging (defaults to reflect.TypeOf)
	FuncName string
}

// MonkeyPatchError creates a new monkey patch error injector
func MonkeyPatchError(targets []ErrorPatchTarget) *MonkeyPatchErrorInjector {
	if targets == nil {
		targets = []ErrorPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_error_%d_targets", len(targets))

	return &MonkeyPatchErrorInjector{
		name:         name,
		targets:      targets,
		patchManager: NewPatchManager(),
		errorCounts:  make(map[interface{}]*int64),
	}
}

func (m *MonkeyPatchErrorInjector) Name() string {
	return m.name
}

func (m *MonkeyPatchErrorInjector) Inject(ctx context.Context) error {
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

		if target.Error == nil && target.ErrorFunc == nil {
			return fmt.Errorf("invalid target %d: either Error or ErrorFunc must be provided", i)
		}

		if target.Error != nil && target.ErrorFunc != nil {
			return fmt.Errorf("invalid target %d: Error and ErrorFunc are mutually exclusive", i)
		}

		if err := ValidateFunction(target.Func); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		// Check that function returns error as last return value
		funcType := reflect.TypeOf(target.Func).Elem()
		if funcType.NumOut() == 0 {
			return fmt.Errorf("invalid target %d: function must return error as last return value", i)
		}

		lastOut := funcType.Out(funcType.NumOut() - 1)
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		if !lastOut.Implements(errorType) {
			return fmt.Errorf("invalid target %d: function last return value must be error, got %v", i, lastOut)
		}

		handle, err := CreatePatch(target.Func)
		if err != nil {
			return fmt.Errorf("failed to create patch for target %d: %w", i, err)
		}

		// Initialize error counter
		var count int64
		m.errorCounts[target.Func] = &count

		// Apply patch with error logic
		funcName := GetFuncName(target.Func, target.FuncName)
		probability := target.Probability
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
				// Inject error instead of calling original function
				m.mu.Lock()
				*m.errorCounts[target.Func]++
				m.mu.Unlock()

				// Generate error
				var err error
				if target.ErrorFunc != nil {
					err = target.ErrorFunc()
				} else {
					err = target.Error
				}

				slog.Debug("monkey patch error triggered",
					slog.String("injector", m.name),
					slog.String("function", funcName),
					slog.String("error", err.Error()),
					slog.Float64("probability", probability))

				// Build return values: return zero values for all except last (error)
				results := make([]reflect.Value, originalNumOut)
				for j := 0; j < originalNumOut-1; j++ {
					results[j] = reflect.Zero(outTypes[j])
				}
				// Set error as last return value
				results[originalNumOut-1] = reflect.ValueOf(err)

				return results
			}

			// No error injection, call original function
			return originalCopy.Call(args)
		}); err != nil {
			m.patchManager.RollbackPatches(i)
			delete(m.errorCounts, target.Func)

			return fmt.Errorf("failed to apply patch %d: %w", i, err)
		}

		m.patchManager.AddPatch(handle)
		slog.Debug("monkey patch applied",
			slog.String("injector", m.name),
			slog.String("function", funcName),
			slog.String("error", m.getErrorDescription(target)),
			slog.Float64("probability", probability))
	}

	slog.Info("monkey patch error injector started",
		slog.String("injector", m.name),
		slog.Int("targets_patched", len(m.targets)))
	slog.Warn("monkey patching requires -gcflags=all=-l for correct operation",
		slog.String("injector", m.name))

	return nil
}

func (m *MonkeyPatchErrorInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.patchManager.RestoreAllPatches(func(handle PatchHandle) string {
			for _, target := range m.targets {
				if target.Func == handle.Func {
					errorCount := int64(0)
					if countPtr, ok := m.errorCounts[target.Func]; ok {
						errorCount = *countPtr
					}
					name := GetFuncName(target.Func, target.FuncName)
					slog.Debug("monkey patch restored",
						slog.String("injector", m.name),
						slog.String("function", name),
						slog.Int64("errors_injected", errorCount))

					return name
				}
			}

			return ""
		})
		m.stopped = true
		slog.Info("monkey patch error injector stopped",
			slog.String("injector", m.name),
			slog.String("status", "patches restored"))
	}

	return nil
}

func (m *MonkeyPatchErrorInjector) getErrorDescription(target ErrorPatchTarget) string {
	if target.ErrorFunc != nil {
		return "dynamic error"
	}

	return target.Error.Error()
}

// Type implements CategorizedInjector
func (m *MonkeyPatchErrorInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics implements MetricsProvider
func (m *MonkeyPatchErrorInjector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalErrors := int64(0)
	for _, countPtr := range m.errorCounts {
		totalErrors += *countPtr
	}

	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": m.patchManager.GetActivePatchCount(),
		"total_errors":   totalErrors,
		"stopped":        m.stopped,
	}
}

// GetErrorCount returns the total number of errors injected across all patches
func (m *MonkeyPatchErrorInjector) GetErrorCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := int64(0)
	for _, countPtr := range m.errorCounts {
		total += *countPtr
	}

	return total
}
