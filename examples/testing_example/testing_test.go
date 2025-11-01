package main

import (
	"context"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	chaostest "github.com/rom8726/chaoskit/testing"

	"github.com/stretchr/testify/require"
)

// Mock functions for demonstration
var (
	processPaymentHandler = func() error {
		// Simulate payment processing
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	calculateTax = func() (float64, error) {
		// Simulate tax calculation
		time.Sleep(5 * time.Millisecond)
		return 100.0, nil
	}
)

// Example test function
func TestCheckoutFlow(t *testing.T) {
	// Create chaos wrapper with injectors
	chaos := chaostest.WithChaos(t,
		injectors.RandomDelay(50*time.Millisecond, 300*time.Millisecond),
		injectors.PanicProbability(0.05),
	)

	// Run test under chaos
	chaos(func() {
		err := processPaymentHandler()
		require.NoError(t, err)

		tax, err := calculateTax()
		require.NoError(t, err)
		require.Greater(t, tax, 0.0)
	})
}

// Example with monkey patching injectors
func TestMonkeyPatchChaos(t *testing.T) {
	// Note: Monkey patching requires -gcflags=all=-l
	chaos := chaostest.WithChaos(t,
		injectors.MonkeyPatchDelay([]injectors.DelayPatchTarget{
			{
				Func:        &processPaymentHandler,
				Probability: 0.3,
				MinDelay:    50 * time.Millisecond,
				MaxDelay:    200 * time.Millisecond,
			},
		}),
		injectors.MonkeyPatchPanic([]injectors.PatchTarget{
			{
				Func:         &calculateTax,
				Probability:  0.05,
				PanicMessage: "chaos: tax calculation failed",
			},
		}),
	)

	chaos(func() {
		err := processPaymentHandler()
		require.NoError(t, err)

		// Handle potential panic
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Panic caught (expected): %v", r)
			}
		}()

		_, _ = calculateTax()
	})
}

// Example with context-based chaos
func TestContextChaos(t *testing.T) {
	chaos := chaostest.WithChaos(t,
		injectors.RandomDelayWithProbability(10*time.Millisecond, 50*time.Millisecond, 0.5),
		injectors.PanicProbability(0.1),
	)

	chaos(func() {
		ctx := context.Background()

		// Use chaos context functions in test code
		for i := 0; i < 5; i++ {
			chaoskit.MaybeDelay(ctx)
			chaoskit.MaybePanic(ctx)

			// Your test logic here
			err := processPaymentHandler()
			require.NoError(t, err)
		}
	})
}
