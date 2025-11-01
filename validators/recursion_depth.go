package validators

import (
	"context"
	"fmt"
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

	for _, depth := range r.depths {
		if depth > r.maxDepth {
			return fmt.Errorf("recursion depth exceeded: %d (limit: %d)", depth, r.maxDepth)
		}
	}

	return nil
}

// RecordRecursion records a recursion depth
func (r *RecursionDepthValidator) RecordRecursion(depth int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.depths = append(r.depths, depth)
}
