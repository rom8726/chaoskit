package validators

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// CompositeValidator combines multiple validators
type CompositeValidator struct {
	name       string
	validators []chaoskit.Validator
}

// Composite creates a composite validator
func Composite(name string, validators ...chaoskit.Validator) *CompositeValidator {
	return &CompositeValidator{
		name:       name,
		validators: validators,
	}
}

func (c *CompositeValidator) Name() string {
	return c.name
}

func (c *CompositeValidator) Validate(ctx context.Context, target chaoskit.Target) error {
	for _, val := range c.validators {
		if err := val.Validate(ctx, target); err != nil {
			return fmt.Errorf("validator %s failed: %w", val.Name(), err)
		}
	}

	return nil
}
