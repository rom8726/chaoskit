package injectors

import (
	"context"
	"fmt"

	"github.com/rom8726/chaoskit"
)

// CompositeInjector combines multiple injectors
type CompositeInjector struct {
	name      string
	injectors []chaoskit.Injector
}

// Composite creates a composite injector
func Composite(name string, injectors ...chaoskit.Injector) *CompositeInjector {
	return &CompositeInjector{
		name:      name,
		injectors: injectors,
	}
}

func (c *CompositeInjector) Name() string {
	return c.name
}

func (c *CompositeInjector) Inject(ctx context.Context) error {
	for _, inj := range c.injectors {
		if err := inj.Inject(ctx); err != nil {
			return fmt.Errorf("injector %s failed: %w", inj.Name(), err)
		}
	}

	return nil
}

func (c *CompositeInjector) Stop(ctx context.Context) error {
	var errs []error
	for _, inj := range c.injectors {
		if err := inj.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping injectors: %v", errs)
	}

	return nil
}
