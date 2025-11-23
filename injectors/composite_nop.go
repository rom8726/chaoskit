//go:build !chaos

package injectors

import (
	"context"

	"github.com/rom8726/chaoskit"
)

// CompositeInjector is a no-op placeholder compiled without the chaos tag.
type CompositeInjector struct {
	name string
}

// Composite creates a no-op composite injector that preserves the expected API.
func Composite(name string, _ ...chaoskit.Injector) *CompositeInjector {
	return &CompositeInjector{name: name}
}

// Name returns the configured name.
func (c *CompositeInjector) Name() string {
	return c.name
}

// Inject is a no-op.
func (*CompositeInjector) Inject(context.Context) error {
	return nil
}

// Stop is a no-op.
func (*CompositeInjector) Stop(context.Context) error {
	return nil
}
