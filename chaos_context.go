package chaoskit

import (
	"context"
	"math/rand"
	"sync"
)

// randKey is a private type for context key
type randKey struct{}

// chaosKey is a private type for context key
type chaosKey struct{}

// ChaosContext provides chaos injection capabilities to user code
type ChaosContext struct {
	mu               sync.RWMutex
	delayFunc        func() bool
	errorFunc        func() error
	panicFunc        func() bool
	networkFunc      func(host string, port int) bool
	cancellationFunc func(context.Context) (context.Context, context.CancelFunc)
	providers        map[string]ChaosProvider
}

// AttachChaos attaches chaos capabilities to context
func AttachChaos(ctx context.Context, chaos *ChaosContext) context.Context {
	return context.WithValue(ctx, chaosKey{}, chaos)
}

// GetChaos retrieves chaos context
func GetChaos(ctx context.Context) *ChaosContext {
	if v := ctx.Value(chaosKey{}); v != nil {
		if chaos, ok := v.(*ChaosContext); ok {
			return chaos
		}
	}

	return nil
}

func MaybeError(ctx context.Context) error {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return nil
	}

	chaos.mu.RLock()
	errorFunc := chaos.errorFunc
	chaos.mu.RUnlock()

	if errorFunc != nil {
		return errorFunc()
	}

	return nil
}

// MaybePanic triggers a panic based on configured probability
// User code should call this at critical points in their logic
func MaybePanic(ctx context.Context) {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return
	}

	chaos.mu.RLock()
	panicFunc := chaos.panicFunc
	chaos.mu.RUnlock()

	if panicFunc != nil && panicFunc() {
		panic("chaos: injected panic")
	}
}

// MaybeDelay applies a delay based on configured injector
// User code can call this at critical points
func MaybeDelay(ctx context.Context) {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return
	}

	chaos.mu.RLock()
	delayFunc := chaos.delayFunc
	chaos.mu.RUnlock()

	if delayFunc != nil {
		delayFunc()
	}
}

// MaybeNetworkChaos applies network chaos (latency, drops) based on configured injector
// User code should call this before network operations
func MaybeNetworkChaos(ctx context.Context, host string, port int) {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return
	}

	chaos.mu.RLock()
	networkFunc := chaos.networkFunc
	chaos.mu.RUnlock()

	if networkFunc != nil && networkFunc(host, port) {
		// Network chaos was applied (latency injected, connection dropped, etc.)
		return
	}
}

// ApplyChaos applies a chaos provider by name
func ApplyChaos(ctx context.Context, providerName string) bool {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return false
	}

	chaos.mu.RLock()
	provider, ok := chaos.providers[providerName]
	chaos.mu.RUnlock()

	if !ok {
		return false
	}

	return provider.Apply(ctx)
}

// RegisterChaosProvider registers a universal chaos provider
func (c *ChaosContext) RegisterProvider(provider ChaosProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.providers == nil {
		c.providers = make(map[string]ChaosProvider)
	}
	c.providers[provider.Name()] = provider
}

// GetProvider returns a registered provider by name
func (c *ChaosContext) GetProvider(name string) (ChaosProvider, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	provider, ok := c.providers[name]

	return provider, ok
}

// AttachRand attaches a deterministic random number generator to context
func AttachRand(ctx context.Context, rng *rand.Rand) context.Context {
	return context.WithValue(ctx, randKey{}, rng)
}

// GetRand retrieves the random number generator from context, or creates a new one if not found
// If seed was set in scenario, the generator will be deterministic
func GetRand(ctx context.Context) *rand.Rand {
	if v := ctx.Value(randKey{}); v != nil {
		if rng, ok := v.(*rand.Rand); ok {
			return rng
		}
	}

	// Fallback to global random if no generator in context
	return rand.New(rand.NewSource(rand.Int63()))
}

// ShouldFail returns true with given probability
// User code can use this to simulate failures
func ShouldFail(ctx context.Context, probability float64) bool {
	chaos := GetChaos(ctx)
	if chaos == nil {
		return false
	}

	rng := GetRand(ctx)

	return rng.Float64() < probability
}

// MaybeCancelContext creates a child context with possible cancellation
// User code should use this to wrap contexts that should be subject to cancellation chaos
func MaybeCancelContext(ctx context.Context) (context.Context, context.CancelFunc) {
	chaos := GetChaos(ctx)
	if chaos == nil {
		// No chaos context, just return parent context with no-op cancel
		return ctx, func() {}
	}

	chaos.mu.RLock()
	cancellationFunc := chaos.cancellationFunc
	chaos.mu.RUnlock()

	if cancellationFunc != nil {
		return cancellationFunc(ctx)
	}

	// No cancellation provider, return parent context
	return ctx, func() {}
}
