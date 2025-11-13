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

// MonkeyPatchDelayInjector uses monkey patching to intercept function calls
// and inject delays with a given probability
//
// WARNING: Monkey patching is inherently unsafe and should only be used in testing.
// It modifies executable code at runtime and may cause:
// - Instability in production
// - Security issues
// - Race conditions
// - Unexpected behavior
//
// IMPORTANT: Requires building with -gcflags=all=-l to disable inlining:
//
//	go test -gcflags=all=-l ./...
//
// Usage:
//
//	targetFunc := func() { ... }
//	injector := injectors.MonkeyPatchDelay([]injectors.DelayPatchTarget{
//	    {Func: &targetFunc, Probability: 0.3, MinDelay: 10*time.Millisecond, MaxDelay: 50*time.Millisecond},
//	})
type MonkeyPatchDelayInjector struct {
	name         string
	targets      []DelayPatchTarget
	patchManager *PatchManager
	delayCounts  map[interface{}]*int64 // Map from function pointer to delay count
	mu           sync.Mutex
	stopped      bool
}

// DelayPatchTarget defines a function to patch and delay parameters
type DelayPatchTarget struct {
	// Func is the function to patch (must be a pointer to function)
	// Example: var myFunc = func() { ... }
	//          target := DelayPatchTarget{Func: &myFunc, ...}
	Func interface{}

	// Probability of delay on each call (0.0 to 1.0)
	Probability float64

	// MinDelay is the minimum delay to apply
	MinDelay time.Duration

	// MaxDelay is the maximum delay to apply
	MaxDelay time.Duration

	// DelayBefore controls whether delay happens before or after function call
	// true = delay before, false = delay after (default: true)
	DelayBefore bool

	// FuncName is optional name for logging (defaults to reflect.TypeOf)
	FuncName string
}

// MonkeyPatchDelay creates a new monkey patch delay injector
func MonkeyPatchDelay(targets []DelayPatchTarget) *MonkeyPatchDelayInjector {
	if targets == nil {
		targets = []DelayPatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_delay_%d_targets", len(targets))

	return &MonkeyPatchDelayInjector{
		name:         name,
		targets:      targets,
		patchManager: NewPatchManager(),
		delayCounts:  make(map[interface{}]*int64),
	}
}

func (m *MonkeyPatchDelayInjector) Name() string {
	return m.name
}

func (m *MonkeyPatchDelayInjector) Inject(ctx context.Context) error {
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

		if target.MinDelay < 0 {
			return fmt.Errorf("invalid target %d: min delay must be non-negative, got %v", i, target.MinDelay)
		}

		if target.MaxDelay < target.MinDelay {
			return fmt.Errorf("invalid target %d: max delay (%v) must be >= min delay (%v)", i, target.MaxDelay, target.MinDelay)
		}

		if err := ValidateFunction(target.Func); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		handle, err := CreatePatch(target.Func)
		if err != nil {
			return fmt.Errorf("failed to create patch for target %d: %w", i, err)
		}

		// Initialize delay counter
		var count int64
		m.delayCounts[target.Func] = &count

		// Apply patch with delay logic
		funcName := GetFuncName(target.Func, target.FuncName)
		probability := target.Probability
		minDelay := target.MinDelay
		maxDelay := target.MaxDelay
		delayBefore := target.DelayBefore
		originalCopy := handle.Original
		rng := chaoskit.GetRand(ctx) // Get deterministic generator from context
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}

		if err := ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
			if rng.Float64() < probability {
				delay := m.calculateDelay(minDelay, maxDelay, rng)

				slog.Debug("monkey patch delay triggered",
					slog.String("injector", m.name),
					slog.String("function", funcName),
					slog.Duration("delay", delay),
					slog.Float64("probability", probability))

				if delayBefore {
					time.Sleep(delay)
				}

				results := originalCopy.Call(args)

				if !delayBefore {
					time.Sleep(delay)
				}

				// Increment counter
				m.mu.Lock()
				*m.delayCounts[target.Func]++
				m.mu.Unlock()

				return results
			}

			return originalCopy.Call(args)
		}); err != nil {
			m.patchManager.RollbackPatches(i)
			delete(m.delayCounts, target.Func)

			return fmt.Errorf("failed to apply patch %d: %w", i, err)
		}

		m.patchManager.AddPatch(handle)
		slog.Info("monkey patch applied",
			slog.String("injector", m.name),
			slog.String("function", funcName),
			slog.Float64("probability", probability),
			slog.Duration("min_delay", minDelay),
			slog.Duration("max_delay", maxDelay),
			slog.Bool("delay_before", delayBefore))
	}

	slog.Info("monkey patch delay injector started",
		slog.String("injector", m.name),
		slog.Int("targets_patched", len(m.targets)))
	slog.Warn("monkey patching requires -gcflags=all=-l for correct operation",
		slog.String("injector", m.name))

	return nil
}

func (m *MonkeyPatchDelayInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.patchManager.RestoreAllPatches(func(handle PatchHandle) string {
			for _, target := range m.targets {
				if target.Func == handle.Func {
					delayCount := int64(0)
					if countPtr, ok := m.delayCounts[target.Func]; ok {
						delayCount = *countPtr
					}
					name := GetFuncName(target.Func, target.FuncName)
					slog.Debug("monkey patch restored",
						slog.String("injector", m.name),
						slog.String("function", name),
						slog.Int64("delays_applied", delayCount))

					return name
				}
			}

			return ""
		})
		m.stopped = true
		slog.Info("monkey patch delay injector stopped",
			slog.String("injector", m.name),
			slog.String("status", "patches restored"))
	}

	return nil
}

// calculateDelay calculates a random delay between min and max
func (m *MonkeyPatchDelayInjector) calculateDelay(min, max time.Duration, rng *rand.Rand) time.Duration {
	if max <= min {
		return min
	}

	delta := max - min
	delay := min + time.Duration(rng.Int63n(int64(delta)))

	return delay
}

// Type implements CategorizedInjector
func (m *MonkeyPatchDelayInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid
}

// GetMetrics implements MetricsProvider
func (m *MonkeyPatchDelayInjector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalDelays := int64(0)
	for _, countPtr := range m.delayCounts {
		totalDelays += *countPtr
	}

	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": m.patchManager.GetActivePatchCount(),
		"total_delays":   totalDelays,
		"stopped":        m.stopped,
	}
}

// GetDelayCount returns the total number of delays applied across all patches
func (m *MonkeyPatchDelayInjector) GetDelayCount() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := int64(0)
	for _, countPtr := range m.delayCounts {
		total += *countPtr
	}

	return total
}

// GetDelayCountForTarget returns delay count for a specific target
func (m *MonkeyPatchDelayInjector) GetDelayCountForTarget(targetFunc interface{}) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if countPtr, ok := m.delayCounts[targetFunc]; ok {
		return *countPtr, true
	}

	return 0, false
}
