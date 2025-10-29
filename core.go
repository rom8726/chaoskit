package chaoskit

import (
	"context"
	"fmt"
)

// Target represents the system under test
type Target interface {
	Name() string
	Setup(ctx context.Context) error
	Teardown(ctx context.Context) error
}

// Step represents a single step in a scenario
type Step interface {
	Name() string
	Execute(ctx context.Context, target Target) error
}

// Injector introduces faults into the system
type Injector interface {
	Name() string
	Inject(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Validator checks system invariants
type Validator interface {
	Name() string
	Validate(ctx context.Context, target Target) error
}

func Run(ctx context.Context, scenario *Scenario) error {
	executor := NewExecutor()
	if err := executor.Run(ctx, scenario); err != nil {
		return err
	}

	fmt.Println(executor.Reporter().GenerateReport())

	return nil
}
