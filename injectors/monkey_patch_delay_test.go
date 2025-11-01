//go:build !disable_monkey_patching

package injectors

import (
	"context"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
)

// Test functions
var (
	testDelayFunc = func() int {
		return 42
	}

	testDelayFuncWithArgs = func(a, b int) int {
		return a + b
	}
)

func TestMonkeyPatchDelayInjector_Creation(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 0.5,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})

	if injector == nil {
		t.Fatal("MonkeyPatchDelay() returned nil")
	}

	if injector.Name() == "" {
		t.Error("Name() should not be empty")
	}
}

func TestMonkeyPatchDelayInjector_Validation(t *testing.T) {
	tests := []struct {
		name      string
		targets   []DelayPatchTarget
		wantError bool
	}{
		{
			name: "valid target",
			targets: []DelayPatchTarget{
				{
					Func:        &testDelayFunc,
					Probability: 0.5,
					MinDelay:    10 * time.Millisecond,
					MaxDelay:    50 * time.Millisecond,
				},
			},
			wantError: false,
		},
		{
			name: "invalid probability",
			targets: []DelayPatchTarget{
				{
					Func:        &testDelayFunc,
					Probability: 1.5,
					MinDelay:    10 * time.Millisecond,
					MaxDelay:    50 * time.Millisecond,
				},
			},
			wantError: true,
		},
		{
			name: "negative min delay",
			targets: []DelayPatchTarget{
				{
					Func:        &testDelayFunc,
					Probability: 0.5,
					MinDelay:    -1 * time.Millisecond,
					MaxDelay:    50 * time.Millisecond,
				},
			},
			wantError: true,
		},
		{
			name: "max delay less than min delay",
			targets: []DelayPatchTarget{
				{
					Func:        &testDelayFunc,
					Probability: 0.5,
					MinDelay:    50 * time.Millisecond,
					MaxDelay:    10 * time.Millisecond,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := MonkeyPatchDelay(tt.targets)
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

func TestMonkeyPatchDelayInjector_DelayBefore(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 1.0, // Always delay
			MinDelay:    50 * time.Millisecond,
			MaxDelay:    100 * time.Millisecond,
			DelayBefore: true,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	start := time.Now()
	result := testDelayFunc()
	duration := time.Since(start)

	if result != 42 {
		t.Errorf("testDelayFunc() = %d, want 42", result)
	}

	if duration < 50*time.Millisecond {
		t.Errorf("Delay was not applied, duration = %v", duration)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchDelayInjector_DelayAfter(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 1.0,
			MinDelay:    50 * time.Millisecond,
			MaxDelay:    100 * time.Millisecond,
			DelayBefore: false,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	start := time.Now()
	result := testDelayFunc()
	duration := time.Since(start)

	if result != 42 {
		t.Errorf("testDelayFunc() = %d, want 42", result)
	}

	if duration < 50*time.Millisecond {
		t.Errorf("Delay was not applied, duration = %v", duration)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchDelayInjector_NoDelayWithZeroProbability(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 0.0, // Never delay
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	start := time.Now()
	result := testDelayFunc()
	duration := time.Since(start)

	if result != 42 {
		t.Errorf("testDelayFunc() = %d, want 42", result)
	}

	// Should complete quickly without delay
	if duration > 10*time.Millisecond {
		t.Errorf("No delay should be applied, but duration = %v", duration)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchDelayInjector_DelayCount(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 1.0,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    20 * time.Millisecond,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	// Call multiple times
	for i := 0; i < 5; i++ {
		testDelayFunc()
	}

	count := injector.GetDelayCount()
	if count != 5 {
		t.Errorf("GetDelayCount() = %d, want 5", count)
	}

	countForTarget, found := injector.GetDelayCountForTarget(&testDelayFunc)
	if !found {
		t.Error("GetDelayCountForTarget() should find target")
	}
	if countForTarget != 5 {
		t.Errorf("GetDelayCountForTarget() = %d, want 5", countForTarget)
	}

	injector.Stop(ctx)
}

func TestMonkeyPatchDelayInjector_Metrics(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 0.5,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
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

func TestMonkeyPatchDelayInjector_Type(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 0.5,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})

	if injector.Type() != chaoskit.InjectorTypeHybrid {
		t.Errorf("Type() = %v, want InjectorTypeHybrid", injector.Type())
	}
}

func TestMonkeyPatchDelayInjector_MultipleTargets(t *testing.T) {
	injector := MonkeyPatchDelay([]DelayPatchTarget{
		{
			Func:        &testDelayFunc,
			Probability: 0.0,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
		{
			Func:        &testDelayFuncWithArgs,
			Probability: 0.0,
			MinDelay:    10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})

	ctx := context.Background()
	if err := injector.Inject(ctx); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	result := testDelayFuncWithArgs(2, 3)
	if result != 5 {
		t.Errorf("testDelayFuncWithArgs(2, 3) = %d, want 5", result)
	}

	injector.Stop(ctx)
}
