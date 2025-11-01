package validators

import (
	"context"

	"github.com/rom8726/chaoskit"
)

// StateConsistencyValidator validates state consistency
type StateConsistencyValidator struct {
	name      string
	checkFunc func(ctx context.Context, target chaoskit.Target) error
}

// StateConsistency creates a custom state consistency validator
func StateConsistency(
	name string,
	checkFunc func(ctx context.Context, target chaoskit.Target) error,
) *StateConsistencyValidator {
	return &StateConsistencyValidator{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (s *StateConsistencyValidator) Name() string {
	return s.name
}

func (s *StateConsistencyValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	return s.checkFunc(ctx, target)
}
