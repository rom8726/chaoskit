//go:build chaos

package chaoskit

import (
	"context"
	"sync"
)

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

func NewChaosContext() *ChaosContext {
	return &ChaosContext{
		providers: make(map[string]ChaosProvider),
	}
}

func (ctx *ChaosContext) SetDelayFunc(fn func() bool) {
	ctx.delayFunc = fn
}

func (ctx *ChaosContext) SetErrorFunc(fn func() error) {
	ctx.errorFunc = fn
}

func (ctx *ChaosContext) SetPanicFunc(fn func() bool) {
	ctx.panicFunc = fn
}

func (ctx *ChaosContext) SetNetworkFunc(fn func(host string, port int) bool) {
	ctx.networkFunc = fn
}

func (ctx *ChaosContext) SetCancellationFunc(fn func(context.Context) (context.Context, context.CancelFunc)) {
	ctx.cancellationFunc = fn
}

// RegisterProvider registers a universal chaos provider
func (ctx *ChaosContext) RegisterProvider(provider ChaosProvider) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.providers == nil {
		ctx.providers = make(map[string]ChaosProvider)
	}
	ctx.providers[provider.Name()] = provider
}

// GetProvider returns a registered provider by name
func (ctx *ChaosContext) GetProvider(name string) (ChaosProvider, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	provider, ok := ctx.providers[name]

	return provider, ok
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
