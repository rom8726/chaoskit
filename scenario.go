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
	scopes     []*Scope
	validators []Validator
	repeat     int
	duration   time.Duration
	seed       *int64 // Optional seed for deterministic randomness (nil = random)
}

// Scope groups injectors logically (e.g., "db", "api", "cache")
type Scope struct {
	name      string
	injectors []Injector
}

// ScopeBuilder builds a scope fluently
type ScopeBuilder struct {
	scope *Scope
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

// WithSeed sets the random seed for deterministic experiments
func (b *ScenarioBuilder) WithSeed(seed int64) *ScenarioBuilder {
	b.scenario.seed = &seed

	return b
}

// Scope adds a scope for grouping injectors
func (b *ScenarioBuilder) Scope(name string, fn func(*ScopeBuilder)) *ScenarioBuilder {
	sb := &ScopeBuilder{
		scope: &Scope{
			name:      name,
			injectors: make([]Injector, 0),
		},
	}
	fn(sb)
	b.scenario.scopes = append(b.scenario.scopes, sb.scope)

	return b
}

// Inject adds a fault injector to the scope
func (sb *ScopeBuilder) Inject(name string, injector Injector) *ScopeBuilder {
	sb.scope.injectors = append(sb.scope.injectors, injector)

	return sb
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
