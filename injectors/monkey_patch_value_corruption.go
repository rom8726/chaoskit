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

// MonkeyPatchValueCorruptionInjector uses monkey patching to intercept function calls
// and corrupt return values with a given probability
//
// WARNING: Monkey patching is inherently unsafe and should only be used in testing.
//
// IMPORTANT: Requires building with -gcflags=all=-l to disable inlining:
//
//	go test -gcflags=all=-l ./...
//
// Usage:
//
//	targetFunc := func() float64 { return 100.0 }
//	injector := injectors.MonkeyPatchValueCorruption([]injectors.ValueCorruptionPatchTarget{
//	    {Func: &targetFunc, CorruptFunc: func(orig float64) float64 { return orig * 10 }, Probability: 0.05},
//	})
type MonkeyPatchValueCorruptionInjector struct {
	name             string
	targets          []ValueCorruptionPatchTarget
	patchManager     *PatchManager
	corruptionCounts map[interface{}]*int64 // Map from function pointer to corruption count
	mu               sync.Mutex
	stopped          bool
}

// ValueCorruptionPatchTarget defines a function to patch and value corruption parameters
type ValueCorruptionPatchTarget struct {
	// Func is the function to patch (must be a pointer to function)
	// Example: var myFunc = func() float64 { return 100.0 }
	//          target := ValueCorruptionPatchTarget{Func: &myFunc, ...}
	Func interface{}

	// CorruptFunc is a function that takes original return value(s) and returns corrupted value(s)
	// Must match the signature: func(originalType1, originalType2, ...) (corruptedType1, corruptedType2, ...)
	// For single return value: func(originalType) corruptedType
	// For multiple return values: func(originalType1, originalType2) (corruptedType1, corruptedType2)
	CorruptFunc interface{}

	// Probability of corruption on each call (0.0 to 1.0)
	Probability float64

	// FuncName is optional name for logging (defaults to reflect.TypeOf)
	FuncName string
}

// MonkeyPatchValueCorruption creates a new monkey patch value corruption injector
func MonkeyPatchValueCorruption(targets []ValueCorruptionPatchTarget) *MonkeyPatchValueCorruptionInjector {
	if targets == nil {
		targets = []ValueCorruptionPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_value_corruption_%d_targets", len(targets))

	return &MonkeyPatchValueCorruptionInjector{
		name:             name,
		targets:          targets,
		patchManager:     NewPatchManager(),
		corruptionCounts: make(map[interface{}]*int64),
	}
}

func (m *MonkeyPatchValueCorruptionInjector) Name() string {
	return m.name
}

//nolint:lll
func (m *MonkeyPatchValueCorruptionInjector) Inject(ctx context.Context) error {
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

		if target.CorruptFunc == nil {
			return fmt.Errorf("invalid target %d: corrupt function is nil", i)
		}

		if err := ValidateFunction(target.Func); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		// Validate CorruptFunc
		corruptVal := reflect.ValueOf(target.CorruptFunc)
		if corruptVal.Kind() != reflect.Func {
			return fmt.Errorf("invalid target %d: corrupt function must be a function, got %v", i, corruptVal.Kind())
		}

		// Validate signatures match
		funcType := reflect.TypeOf(target.Func).Elem()
		corruptType := corruptVal.Type()

		if funcType.NumOut() == 0 {
			return fmt.Errorf("invalid target %d: target function must return at least one value", i)
		}

		// CorruptFunc should take all return values of original function as inputs
		if corruptType.NumIn() != funcType.NumOut() {
			return fmt.Errorf("invalid target %d: corrupt function must accept %d parameters (matching target function return values), got %d",
				i, funcType.NumOut(), corruptType.NumIn())
		}

		// Check input types match return types
		for j := 0; j < funcType.NumOut(); j++ {
			if !corruptType.In(j).AssignableTo(funcType.Out(j)) {
				return fmt.Errorf("invalid target %d: corrupt function parameter %d type (%v) does not match target return type (%v)",
					i, j, corruptType.In(j), funcType.Out(j))
			}
		}

		// CorruptFunc should return same number and types as original
		if corruptType.NumOut() != funcType.NumOut() {
			return fmt.Errorf("invalid target %d: corrupt function must return %d values, got %d",
				i, funcType.NumOut(), corruptType.NumOut())
		}

		// Check return types match
		for j := 0; j < funcType.NumOut(); j++ {
			if !corruptType.Out(j).AssignableTo(funcType.Out(j)) {
				return fmt.Errorf("invalid target %d: corrupt function return type %d (%v) is not assignable to target return type (%v)",
					i, j, corruptType.Out(j), funcType.Out(j))
			}
		}

		handle, err := CreatePatch(target.Func)
		if err != nil {
			return fmt.Errorf("failed to create patch for target %d: %w", i, err)
		}

		// Initialize corruption counter
		var count int64
		m.corruptionCounts[target.Func] = &count

		// Apply patch with corruption logic
		funcName := GetFuncName(target.Func, target.FuncName)
		probability := target.Probability
		originalCopy := handle.Original

		// Store corruption function value
		corruptCopy := reflect.ValueOf(target.CorruptFunc)

		rng := chaoskit.GetRand(ctx) // Get deterministic generator from context
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}

		if err := ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
			// Call original function first
			originalResults := originalCopy.Call(args)

			// Check probability
			if rng.Float64() < probability {
				// Corrupt return values
				m.mu.Lock()
				*m.corruptionCounts[target.Func]++
				m.mu.Unlock()

				// Call corrupt function with original results
				corruptedResults := corruptCopy.Call(originalResults)

				slog.Debug("monkey patch value corruption triggered",
					slog.String("injector", m.name),
					slog.String("function", funcName),
					slog.Float64("probability", probability))

				return corruptedResults
			}

			// No corruption, return original results
			return originalResults
		}); err != nil {
			m.patchManager.RollbackPatches(i)
			delete(m.corruptionCounts, target.Func)

			return fmt.Errorf("failed to apply patch %d: %w", i, err)
		}

		m.patchManager.AddPatch(handle)
		slog.Debug("monkey patch applied",
			slog.String("injector", m.name),
			slog.String("function", funcName),
			slog.Float64("corruption_probability", probability))
	}

	slog.Info("monkey patch value corruption injector started",
		slog.String("injector", m.name),
		slog.Int("targets_patched", len(m.targets)))
	slog.Warn("monkey patching requires -gcflags=all=-l for correct operation",
		slog.String("injector", m.name))

	return nil
}

func (m *MonkeyPatchValueCorruptionInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.patchManager.RestoreAllPatches(func(handle PatchHandle) string {
			for _, target := range m.targets {
				if target.Func == handle.Func {
					corruptionCount := int64(0)
					if countPtr, ok := m.corruptionCounts[target.Func]; ok {
						corruptionCount = *countPtr
					}
					name := GetFuncName(target.Func, target.FuncName)
					slog.Debug("monkey patch restored",
						slog.String("injector", m.name),
						slog.String("function", name),
						slog.Int64("corruptions_applied", corruptionCount))

					return name
				}
			}

			return ""
		})
		m.stopped = true
		slog.Info("monkey patch value corruption injector stopped",
			slog.String("injector", m.name),
			slog.String("status", "patches restored"))
	}

	return nil
}

// Type implements CategorizedInjector
func (m *MonkeyPatchValueCorruptionInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics implements MetricsProvider
func (m *MonkeyPatchValueCorruptionInjector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalCorruptions := int64(0)
	for _, countPtr := range m.corruptionCounts {
		totalCorruptions += *countPtr
	}

	return map[string]interface{}{
		"total_targets":     len(m.targets),
		"active_patches":    m.patchManager.GetActivePatchCount(),
		"total_corruptions": totalCorruptions,
		"stopped":           m.stopped,
	}
}

// GetCorruptionCount returns the total number of corruptions applied across all patches
func (m *MonkeyPatchValueCorruptionInjector) GetCorruptionCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := int64(0)
	for _, countPtr := range m.corruptionCounts {
		total += *countPtr
	}

	return total
}
