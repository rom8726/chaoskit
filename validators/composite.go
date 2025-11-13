package validators

import (
	"context"
	"fmt"
	"log/slog"

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
			chaoskit.GetLogger(ctx).Error("composite validator failed",
				slog.String("composite_validator", c.name),
				slog.String("failed_validator", val.Name()),
				slog.String("error", err.Error()))

			return fmt.Errorf("validator %s failed: %w", val.Name(), err)
		}
	}

	chaoskit.GetLogger(ctx).Debug("composite validator passed",
		slog.String("composite_validator", c.name),
		slog.Int("validator_count", len(c.validators)))

	return nil
}
