package validators

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/rom8726/chaoskit"
)

// RecursionDepthValidator checks recursion depth
type RecursionDepthValidator struct {
	name     string
	maxDepth int
	mu       sync.RWMutex
	depths   []int
}

// RecursionDepthLimit creates a recursion depth validator
func RecursionDepthLimit(maxDepth int) *RecursionDepthValidator {
	return &RecursionDepthValidator{
		name:     fmt.Sprintf("recursion_depth_limit_%d", maxDepth),
		maxDepth: maxDepth,
		depths:   make([]int, 0),
	}
}

func (r *RecursionDepthValidator) Name() string {
	return r.name
}

func (r *RecursionDepthValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	maxObserved := 0
	for _, depth := range r.depths {
		if depth > maxObserved {
			maxObserved = depth
		}
		// Warn if approaching limit (80% threshold)
		if depth > int(float64(r.maxDepth)*0.8) {
			chaoskit.GetLogger(ctx).Warn("recursion depth approaching limit",
				slog.String("validator", r.name),
				slog.Int("current_depth", depth),
				slog.Int("limit", r.maxDepth))
		}
		if depth > r.maxDepth {
			err := fmt.Errorf("recursion depth exceeded: %d (limit: %d)", depth, r.maxDepth)
			chaoskit.GetLogger(ctx).Error("recursion depth validator failed",
				slog.String("validator", r.name),
				slog.Int("depth", depth),
				slog.Int("limit", r.maxDepth),
				slog.String("error", err.Error()))

			return err
		}
	}

	if len(r.depths) > 0 {
		chaoskit.GetLogger(ctx).Debug("recursion depth validator passed",
			slog.String("validator", r.name),
			slog.Int("max_observed_depth", maxObserved),
			slog.Int("limit", r.maxDepth),
			slog.Int("total_measurements", len(r.depths)))
	}

	return nil
}

// RecordRecursion records a recursion depth
func (r *RecursionDepthValidator) RecordRecursion(depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.depths = append(r.depths, depth)
}
