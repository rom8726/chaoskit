//go:build !disable_monkey_patching

package injectors

import (
	"context"
	"testing"

	"github.com/rom8726/chaoskit"
)

// Test functions
var (
	testCorruptionFunc = func() float64 {
		return 100.0
	}

	testCorruptionFuncMultiple = func() (int, string) {
		return 42, "hello"
	}
)

func TestMonkeyPatchValueCorruptionInjector_Creation(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 10
			},
			Probability: 0.5,
		},
	})

	if injector == nil {
		t.Fatal("MonkeyPatchValueCorruption() returned nil")
	}

	if injector.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestMonkeyPatchValueCorruptionInjector_Validation(t *testing.T) {
	tests := []struct {
		name      string
		targets   []ValueCorruptionPatchTarget
		wantError bool
	}{
		{
			name: "valid target",
			targets: []ValueCorruptionPatchTarget{
				{
					Func:        &testCorruptionFunc,
					CorruptFunc: func(orig float64) float64 { return orig * 2 },
					Probability: 0.5,
				},
			},
			wantError: false,
		},
		{
			name: "nil CorruptFunc",
			targets: []ValueCorruptionPatchTarget{
				{
					Func:        &testCorruptionFunc,
					CorruptFunc: nil,
					Probability: 0.5,
				},
			},
			wantError: true,
		},
		{
			name: "invalid probability",
			targets: []ValueCorruptionPatchTarget{
				{
					Func:        &testCorruptionFunc,
					CorruptFunc: func(orig float64) float64 { return orig * 2 },
					Probability: 1.5,
				},
			},
			wantError: true,
		},
		{
			name: "mismatched corrupt function signature",
			targets: []ValueCorruptionPatchTarget{
				{
					Func:        &testCorruptionFunc,
					CorruptFunc: func(orig int) int { return orig * 2 }, // Wrong type
					Probability: 0.5,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := MonkeyPatchValueCorruption(tt.targets)
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

func TestMonkeyPatchValueCorruptionInjector_ValueCorruption(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 10 // Corrupt: multiply by 10
			},
			Probability: 1.0, // Always corrupt
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result := testCorruptionFunc()
	if result != 1000.0 {
		t.Errorf("testCorruptionFunc() = %f, want 1000.0 (corrupted)", result)
	}

	injector.Stop(ctx)

	// After restore, should return original value
	result = testCorruptionFunc()
	if result != 100.0 {
		t.Errorf("testCorruptionFunc() after restore = %f, want 100.0", result)
	}
}

func TestMonkeyPatchValueCorruptionInjector_NoCorruptionWithZeroProbability(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 10
			},
			Probability: 0.0, // Never corrupt
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result := testCorruptionFunc()
	if result != 100.0 {
		t.Errorf("testCorruptionFunc() = %f, want 100.0 (not corrupted)", result)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchValueCorruptionInjector_MultipleReturnValues(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFuncMultiple,
			CorruptFunc: func(origInt int, origStr string) (int, string) {
				return origInt * 2, origStr + "_corrupted"
			},
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	resultInt, resultStr := testCorruptionFuncMultiple()
	if resultInt != 84 {
		t.Errorf("testCorruptionFuncMultiple() int = %d, want 84", resultInt)
	}
	if resultStr != "hello_corrupted" {
		t.Errorf("testCorruptionFuncMultiple() string = %s, want 'hello_corrupted'", resultStr)
	}

	injector.Stop(ctx)

	// After restore
	resultInt, resultStr = testCorruptionFuncMultiple()
	if resultInt != 42 || resultStr != "hello" {
		t.Errorf("After restore: got (%d, %s), want (42, hello)", resultInt, resultStr)
	}
}

func TestMonkeyPatchValueCorruptionInjector_CorruptionCount(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 2
			},
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Call multiple times
	for i := 0; i < 5; i++ {
		_ = testCorruptionFunc()
	}

	count := injector.GetCorruptionCount()
	if count != 5 {
		t.Errorf("GetCorruptionCount() = %d, want 5", count)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchValueCorruptionInjector_Metrics(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 2
			},
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

	injector.Stop(ctx)
}

func TestMonkeyPatchValueCorruptionInjector_Type(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 2
			},
			Probability: 0.5,
		},
	})

	if injector.Type() != chaoskit.InjectorTypeHybrid {
		t.Errorf("Type() = %v, want InjectorTypeHybrid", injector.Type())
	}
}

func TestMonkeyPatchValueCorruptionInjector_MultipleTargets(t *testing.T) {
	injector := MonkeyPatchValueCorruption([]ValueCorruptionPatchTarget{
		{
			Func: &testCorruptionFunc,
			CorruptFunc: func(orig float64) float64 {
				return orig * 10
			},
			Probability: 1.0,
		},
		{
			Func: &testCorruptionFuncMultiple,
			CorruptFunc: func(origInt int, origStr string) (int, string) {
				return origInt + 10, origStr + "_test"
			},
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result1 := testCorruptionFunc()
	if result1 != 1000.0 {
		t.Errorf("testCorruptionFunc() = %f, want 1000.0", result1)
	}

	result2Int, result2Str := testCorruptionFuncMultiple()
	if result2Int != 52 || result2Str != "hello_test" {
		t.Errorf("testCorruptionFuncMultiple() = (%d, %s), want (52, hello_test)", result2Int, result2Str)
	}

	injector.Stop(ctx)
}
