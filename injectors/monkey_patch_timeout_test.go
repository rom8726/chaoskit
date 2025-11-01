//go:build !disable_monkey_patching

package injectors

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
)

// Test functions
var (
	testTimeoutFunc = func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	testTimeoutFuncFast = func(ctx context.Context) error {
		// Fast function
		return nil
	}
)

func TestMonkeyPatchTimeoutInjector_Creation(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     50 * time.Millisecond,
			Probability: 0.5,
		},
	})

	if injector == nil {
		t.Fatal("MonkeyPatchTimeout() returned nil")
	}

	if injector.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestMonkeyPatchTimeoutInjector_Validation(t *testing.T) {
	tests := []struct {
		name      string
		targets   []TimeoutPatchTarget
		wantError bool
	}{
		{
			name: "valid target",
			targets: []TimeoutPatchTarget{
				{
					Func:        &testTimeoutFunc,
					Timeout:     50 * time.Millisecond,
					Probability: 0.5,
				},
			},
			wantError: false,
		},
		{
			name: "invalid probability",
			targets: []TimeoutPatchTarget{
				{
					Func:        &testTimeoutFunc,
					Timeout:     50 * time.Millisecond,
					Probability: 1.5,
				},
			},
			wantError: true,
		},
		{
			name: "zero timeout",
			targets: []TimeoutPatchTarget{
				{
					Func:        &testTimeoutFunc,
					Timeout:     0,
					Probability: 0.5,
				},
			},
			wantError: true,
		},
		{
			name: "function without context",
			targets: []TimeoutPatchTarget{
				{
					Func:        func() error { return nil },
					Timeout:     50 * time.Millisecond,
					Probability: 0.5,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := MonkeyPatchTimeout(tt.targets)
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

func TestMonkeyPatchTimeoutInjector_TimeoutInjection(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     50 * time.Millisecond, // Shorter than function execution
			Probability: 1.0,                   // Always timeout
			ReturnError: errors.New("timeout error"),
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Function should timeout
	err := testTimeoutFunc(ctx)
	if err == nil {
		t.Error("Expected timeout error, but got nil")
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchTimeoutInjector_NoTimeoutWithZeroProbability(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFuncFast,
			Timeout:     50 * time.Millisecond,
			Probability: 0.0, // Never timeout
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	err := testTimeoutFuncFast(ctx)
	if err != nil {
		t.Errorf("testTimeoutFuncFast() error = %v, want nil", err)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchTimeoutInjector_TimeoutCount(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     10 * time.Millisecond,
			Probability: 1.0,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Call multiple times (will timeout)
	for i := 0; i < 3; i++ {
		_ = testTimeoutFunc(ctx)
	}

	count := injector.GetTimeoutCount()
	if count == 0 {
		t.Error("GetTimeoutCount() should be > 0 after timeouts")
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchTimeoutInjector_Metrics(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     50 * time.Millisecond,
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

func TestMonkeyPatchTimeoutInjector_CustomReturnError(t *testing.T) {
	customErr := errors.New("custom timeout")
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     10 * time.Millisecond,
			Probability: 1.0,
			ReturnError: customErr,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	err := testTimeoutFunc(ctx)
	if err == nil {
		t.Error("Expected error, but got nil")
	}
	if err != nil && err.Error() != customErr.Error() {
		t.Errorf("Error message = %v, want %v", err, customErr)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchTimeoutInjector_Type(t *testing.T) {
	injector := MonkeyPatchTimeout([]TimeoutPatchTarget{
		{
			Func:        &testTimeoutFunc,
			Timeout:     50 * time.Millisecond,
			Probability: 0.5,
		},
	})

	if injector.Type() != chaoskit.InjectorTypeHybrid {
		t.Errorf("Type() = %v, want InjectorTypeHybrid", injector.Type())
	}
}
