//go:build !disable_monkey_patching

package injectors

import (
	"context"
	"testing"

	"github.com/rom8726/chaoskit"
)

// Test functions
var (
	testPanicFunc = func() {
		// Normal function
	}

	testPanicFuncWithResult = func() string {
		return "success"
	}
)

func TestMonkeyPatchPanicInjector_Creation(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:        &testPanicFunc,
			Probability: 0.5,
			FuncName:    "testPanicFunc",
		},
	})

	if injector == nil {
		t.Fatal("MonkeyPatchPanic() returned nil")
	}

	if injector.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestMonkeyPatchPanicInjector_Validation(t *testing.T) {
	tests := []struct {
		name      string
		targets   []PatchTarget
		wantError bool
	}{
		{
			name: "valid target",
			targets: []PatchTarget{
				{
					Func:        &testPanicFunc,
					Probability: 0.5,
				},
			},
			wantError: false,
		},
		{
			name: "invalid probability",
			targets: []PatchTarget{
				{
					Func:        &testPanicFunc,
					Probability: 1.5, // Invalid
				},
			},
			wantError: true,
		},
		{
			name: "nil function",
			targets: []PatchTarget{
				{
					Func:        nil,
					Probability: 0.5,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := MonkeyPatchPanic(tt.targets)
			ctx := context.Background()
			err := injector.Inject(ctx)

			if (err != nil) != tt.wantError {
				t.Errorf("Inject() error = %v, wantError %v", err, tt.wantError)
			}

			if err == nil {
				injector.Stop(ctx)
			}
		})
	}
}

func TestMonkeyPatchPanicInjector_InjectAndRestore(t *testing.T) {
	t.SkipNow()
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:        &testPanicFunc,
			Probability: 0.0, // Never panic for this test
			FuncName:    "testPanicFunc",
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Function should work normally with 0 probability
	testPanicFunc()

	// Restore
	if err := injector.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Function should still work
	testPanicFunc()
}

func TestMonkeyPatchPanicInjector_PanicInjection(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:         &testPanicFunc,
			Probability:  1.0, // Always panic
			PanicMessage: "test panic",
			FuncName:     "testPanicFunc",
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Function should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic, but function did not panic")
		} else if r != "test panic" {
			t.Errorf("Panic message = %v, want 'test panic'", r)
		}
	}()

	testPanicFunc()
}

func TestMonkeyPatchPanicInjector_Metrics(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:        &testPanicFunc,
			Probability: 0.5,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	metrics := injector.GetMetrics()
	if metrics == nil {
		t.Fatal("GetMetrics() returned nil")
	}

	if totalTargets, ok := metrics["total_targets"].(int); !ok || totalTargets != 1 {
		t.Errorf("GetMetrics() total_targets = %v, want 1", metrics["total_targets"])
	}

	if activePatches, ok := metrics["active_patches"].(int); !ok || activePatches != 1 {
		t.Errorf("GetMetrics() active_patches = %v, want 1", metrics["active_patches"])
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchPanicInjector_Type(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{Func: &testPanicFunc, Probability: 0.5},
	})

	if injector.Type() != chaoskit.InjectorTypeHybrid {
		t.Errorf("Type() = %v, want InjectorTypeHybrid", injector.Type())
	}
}

func TestMonkeyPatchPanicInjector_StopWithoutInject(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{Func: &testPanicFunc, Probability: 0.5},
	})

	ctx := context.Background()
	if err := injector.Stop(ctx); err != nil {
		t.Errorf("Stop() without Inject() should not error, got %v", err)
	}
}

func TestMonkeyPatchPanicInjector_MultipleTargets(t *testing.T) {
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:        &testPanicFunc,
			Probability: 0.0,
		},
		{
			Func:        &testPanicFuncWithResult,
			Probability: 0.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result := testPanicFuncWithResult()
	if result != "success" {
		t.Errorf("testPanicFuncWithResult() = %v, want 'success'", result)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchPanicInjector_CustomPanicMessage(t *testing.T) {
	customMsg := "custom panic message"
	injector := MonkeyPatchPanic([]PatchTarget{
		{
			Func:         &testPanicFunc,
			Probability:  1.0,
			PanicMessage: customMsg,
		},
	})

	ctx := context.Background()
	injector.Inject(ctx)

	defer func() {
		if r := recover(); r != customMsg {
			t.Errorf("Panic message = %v, want %v", r, customMsg)
		}
	}()

	testPanicFunc()
}
