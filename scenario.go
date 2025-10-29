package chaoskit

import (
	"context"
	"time"
)

// Scenario describes a chaos experiment
type Scenario struct {
	name       string
	target     Target
	steps      []Step
	injectors  []Injector
	validators []Validator
	repeat     int
	duration   time.Duration
}

// ScenarioBuilder builds scenarios fluently
type ScenarioBuilder struct {
	scenario *Scenario
}

// NewScenario creates a new scenario builder
func NewScenario(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: &Scenario{
			name:   name,
			repeat: 1,
		},
	}
}

// WithTarget sets the target system
func (b *ScenarioBuilder) WithTarget(target Target) *ScenarioBuilder {
	b.scenario.target = target
	return b
}

// Step adds a step to the scenario
func (b *ScenarioBuilder) Step(name string, fn func(context.Context, Target) error) *ScenarioBuilder {
	b.scenario.steps = append(b.scenario.steps, &funcStep{name: name, fn: fn})
	return b
}

// Inject adds a fault injector
func (b *ScenarioBuilder) Inject(name string, injector Injector) *ScenarioBuilder {
	b.scenario.injectors = append(b.scenario.injectors, injector)
	return b
}

// Assert adds a validator
func (b *ScenarioBuilder) Assert(name string, validator Validator) *ScenarioBuilder {
	b.scenario.validators = append(b.scenario.validators, validator)
	return b
}

// Repeat sets the number of times to repeat the scenario
func (b *ScenarioBuilder) Repeat(n int) *ScenarioBuilder {
	b.scenario.repeat = n
	return b
}

// RunFor sets the duration to run the scenario
func (b *ScenarioBuilder) RunFor(duration time.Duration) *ScenarioBuilder {
	b.scenario.duration = duration
	return b
}

// Build returns the built scenario
func (b *ScenarioBuilder) Build() *Scenario {
	return b.scenario
}

// funcStep implements Step interface
type funcStep struct {
	name string
	fn   func(context.Context, Target) error
}

func (s *funcStep) Name() string {
	return s.name
}

func (s *funcStep) Execute(ctx context.Context, target Target) error {
	return s.fn(ctx, target)
}
