package testing

import (
	"context"

	"github.com/rom8726/chaoskit"
)

// TestingT is an interface that matches testing.T and similar types
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
}

// WithChaos creates a wrapper function that injects chaos into a test function.
// It automatically calls Inject() on all provided injectors before the test
// and Stop() on all injectors after the test (even if the test panics).
//
// Usage:
//
//	func TestCheckoutFlow(t *testing.T) {
//	    chaos := chaoskit.WithChaos(t,
//	        injectors.RandomDelay(50*time.Millisecond, 300*time.Millisecond),
//	        injectors.PanicProbability(0.05),
//	    )
//	    chaos(func() {
//	        err := CheckoutOrder()
//	        require.NoError(t, err)
//	    })
//	}
func WithChaos(t TestingT, injectors ...chaoskit.Injector) func(fn func()) {
	return func(fn func()) {
		ctx := context.Background()

		// Inject all injectors
		for _, inj := range injectors {
			if err := inj.Inject(ctx); err != nil {
				t.Errorf("failed to inject %s: %v", inj.Name(), err)
				t.FailNow()

				return
			}
		}

		// Defer stop for all injectors (will be called even if test panics)
		defer func() {
			for _, inj := range injectors {
				if err := inj.Stop(ctx); err != nil {
					// Log error but don't fail test (cleanup phase)
					if logger, ok := t.(interface{ Logf(string, ...interface{}) }); ok {
						logger.Logf("warning: failed to stop injector %s: %v", inj.Name(), err)
					}
				}
			}
		}()

		// Execute the test function
		fn()
	}
}
