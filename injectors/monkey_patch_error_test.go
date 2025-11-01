//go:build !disable_monkey_patching

package injectors

import (
	"context"
	"errors"
	"testing"

	"github.com/rom8726/chaoskit"
)

// Test functions
var (
	testErrorFunc = func() error {
		return nil
	}

	testErrorFuncWithResult = func() (int, error) {
		return 42, nil
	}
)

func TestMonkeyPatchErrorInjector_Creation(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       errors.New("test error"),
			Probability: 0.5,
		},
	})

	if injector == nil {
		t.Fatal("MonkeyPatchError() returned nil")
	}

	if injector.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestMonkeyPatchErrorInjector_Validation(t *testing.T) {
	tests := []struct {
		name      string
		targets   []ErrorPatchTarget
		wantError bool
	}{
		{
			name: "valid target with Error",
			targets: []ErrorPatchTarget{
				{
					Func:        &testErrorFunc,
					Error:       errors.New("test"),
					Probability: 0.5,
				},
			},
			wantError: false,
		},
		{
			name: "valid target with ErrorFunc",
			targets: []ErrorPatchTarget{
				{
					Func:        &testErrorFunc,
					ErrorFunc:   func() error { return errors.New("dynamic") },
					Probability: 0.5,
				},
			},
			wantError: false,
		},
		{
			name: "invalid - no Error or ErrorFunc",
			targets: []ErrorPatchTarget{
				{
					Func:        &testErrorFunc,
					Probability: 0.5,
				},
			},
			wantError: true,
		},
		{
			name: "invalid - both Error and ErrorFunc",
			targets: []ErrorPatchTarget{
				{
					Func:        &testErrorFunc,
					Error:       errors.New("static"),
					ErrorFunc:   func() error { return errors.New("dynamic") },
					Probability: 0.5,
				},
			},
			wantError: true,
		},
		{
			name: "invalid - function doesn't return error",
			targets: []ErrorPatchTarget{
				{
					Func:        func() int { return 42 },
					Error:       errors.New("test"),
					Probability: 0.5,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := MonkeyPatchError(tt.targets)
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

func TestMonkeyPatchErrorInjector_ErrorInjection(t *testing.T) {
	injectedErr := errors.New("injected error")
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       injectedErr,
			Probability: 1.0, // Always inject
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	err := testErrorFunc()
	if err == nil {
		t.Error("Expected error, but got nil")
	}
	if err != nil && err.Error() != injectedErr.Error() {
		t.Errorf("Error = %v, want %v", err, injectedErr)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchErrorInjector_DynamicErrorFunc(t *testing.T) {
	callCount := 0
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func: &testErrorFunc,
			ErrorFunc: func() error {
				callCount++
				return errors.New("dynamic error")
			},
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	err := testErrorFunc()
	if err == nil {
		t.Error("Expected error, but got nil")
	}

	if callCount != 1 {
		t.Errorf("ErrorFunc called %d times, want 1", callCount)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchErrorInjector_NoErrorWithZeroProbability(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       errors.New("should not appear"),
			Probability: 0.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	err := testErrorFunc()
	if err != nil {
		t.Errorf("testErrorFunc() error = %v, want nil", err)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchErrorInjector_ErrorCount(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       errors.New("test"),
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Call multiple times
	for i := 0; i < 5; i++ {
		_ = testErrorFunc()
	}

	count := injector.GetErrorCount()
	if count != 5 {
		t.Errorf("GetErrorCount() = %d, want 5", count)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchErrorInjector_WithReturnValue(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFuncWithResult,
			Error:       errors.New("injected"),
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result, err := testErrorFuncWithResult()
	if err == nil {
		t.Error("Expected error, but got nil")
	}
	if result != 0 {
		t.Errorf("Result should be zero value, got %d", result)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchErrorInjector_Metrics(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       errors.New("test"),
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

func TestMonkeyPatchErrorInjector_Type(t *testing.T) {
	injector := MonkeyPatchError([]ErrorPatchTarget{
		{
			Func:        &testErrorFunc,
			Error:       errors.New("test"),
			Probability: 0.5,
		},
	})

	if injector.Type() != chaoskit.InjectorTypeHybrid {
		t.Errorf("Type() = %v, want InjectorTypeHybrid", injector.Type())
	}
}
