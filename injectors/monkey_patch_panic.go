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

// MonkeyPatchPanicInjector uses monkey patching to intercept function calls
// and inject panics with a given probability
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
//	injector := injectors.MonkeyPatchPanic([]injectors.PatchTarget{
//	    {Func: &targetFunc, Probability: 0.1},
//	})
type MonkeyPatchPanicInjector struct {
	name         string
	targets      []PatchTarget
	patchManager *PatchManager
	mu           sync.Mutex
	stopped      bool
}

// PatchTarget defines a function to patch and panic probability
type PatchTarget struct {
	// Func is the function to patch (must be a pointer to function)
	// Example: var myFunc = func() { ... }
	//          target := PatchTarget{Func: &myFunc, ...}
	Func interface{}

	// Probability of panic on each call (0.0 to 1.0)
	Probability float64

	// PanicMessage is the message to panic with
	// If empty, uses default: "chaos: monkey patch panic"
	PanicMessage string

	// FuncName is optional name for logging (defaults to reflect.TypeOf)
	FuncName string
}

// MonkeyPatchPanic creates a new monkey patch panic injector
func MonkeyPatchPanic(targets []PatchTarget) *MonkeyPatchPanicInjector {
	if targets == nil {
		targets = []PatchTarget{}
	}

	name := fmt.Sprintf("monkey_patch_panic_%d_targets", len(targets))

	return &MonkeyPatchPanicInjector{
		name:         name,
		targets:      targets,
		patchManager: NewPatchManager(),
	}
}

func (m *MonkeyPatchPanicInjector) Name() string {
	return m.name
}

func (m *MonkeyPatchPanicInjector) Inject(ctx context.Context) error {
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

		if err := ValidateFunction(target.Func); err != nil {
			return fmt.Errorf("invalid target %d: %w", i, err)
		}

		handle, err := CreatePatch(target.Func)
		if err != nil {
			return fmt.Errorf("failed to create patch for target %d: %w", i, err)
		}

		// Apply patch with panic logic
		funcName := GetFuncName(target.Func, target.FuncName)
		panicMsg := m.getPanicMessage(target)
		probability := target.Probability
		originalCopy := handle.Original
		rng := chaoskit.GetRand(ctx) // Get deterministic generator from context
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}

		if err := ApplyPatch(&handle, func(args []reflect.Value) []reflect.Value {
			if rng.Float64() < probability {
				slog.Debug("monkey patch panic triggered",
					slog.String("injector", m.name),
					slog.String("function", funcName),
					slog.Float64("probability", probability))
				panic(panicMsg)
			}

			return originalCopy.Call(args)
		}); err != nil {
			// Rollback already applied patches
			m.patchManager.RollbackPatches(i)

			return fmt.Errorf("failed to apply patch %d: %w", i, err)
		}

		m.patchManager.AddPatch(handle)
		slog.Info("monkey patch applied",
			slog.String("injector", m.name),
			slog.String("function", funcName),
			slog.Float64("probability", probability))
	}

	slog.Info("monkey patch panic injector started",
		slog.String("injector", m.name),
		slog.Int("targets_patched", len(m.targets)))
	slog.Warn("monkey patching requires -gcflags=all=-l for correct operation",
		slog.String("injector", m.name))

	return nil
}

func (m *MonkeyPatchPanicInjector) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.patchManager.RestoreAllPatches(func(handle PatchHandle) string {
			for _, target := range m.targets {
				if target.Func == handle.Func {
					return GetFuncName(target.Func, target.FuncName)
				}
			}

			return ""
		})
		m.stopped = true
		slog.Info("monkey patch panic injector stopped",
			slog.String("injector", m.name),
			slog.String("status", "patches restored"))
	}

	return nil
}

func (m *MonkeyPatchPanicInjector) getPanicMessage(target PatchTarget) string {
	if target.PanicMessage != "" {
		return target.PanicMessage
	}

	return "chaos: monkey patch panic"
}

// Type implements CategorizedInjector
func (m *MonkeyPatchPanicInjector) Type() chaoskit.InjectorType {
	return chaoskit.InjectorTypeHybrid // Works both globally (patches) and can be context-aware
}

// GetMetrics implements MetricsProvider
func (m *MonkeyPatchPanicInjector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	return map[string]interface{}{
		"total_targets":  len(m.targets),
		"active_patches": m.patchManager.GetActivePatchCount(),
		"stopped":        m.stopped,
	}
}
