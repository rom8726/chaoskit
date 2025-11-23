package chaoskit

import (
	"context"
	"math/rand"
)

// randKey is a private type for context key
type randKey struct{}

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
