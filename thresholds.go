package chaoskit

import (
	"fmt"
	"time"
)

// SuccessThresholds defines criteria for test success
type SuccessThresholds struct {
	// MinSuccessRate is minimum acceptable success rate (0.0-1.0)
	// Example: 0.95 = 95% of iterations must succeed
	MinSuccessRate float64 `json:"min_success_rate" yaml:"min_success_rate"`

	// CriticalValidators lists validators that MUST pass (block release if fail)
	// Example: ["goroutine-limit", "infinite-loop"]
	CriticalValidators []string `json:"critical_validators" yaml:"critical_validators"`

	// WarningValidators lists validators that produce warnings (don't block)
	// Example: ["execution-time", "memory-pressure"]
	WarningValidators []string `json:"warning_validators,omitempty" yaml:"warning_validators,omitempty"`

	// MaxFailedIterations is maximum number of failed iterations allowed
	// If exceeded, test fails regardless of success rate
	MaxFailedIterations int `json:"max_failed_iterations,omitempty" yaml:"max_failed_iterations,omitempty"`

	// MaxAvgDuration is maximum acceptable average execution duration
	// Exceeding this produces a warning
	MaxAvgDuration time.Duration `json:"max_avg_duration,omitempty" yaml:"max_avg_duration,omitempty"`

	// RequireAllValidatorsPassing requires ALL validators to pass
	// If true, any validator failure = FAIL
	//nolint:lll
	RequireAllValidatorsPassing bool `json:"require_all_validators_passing,omitempty" yaml:"require_all_validators_passing,omitempty"`
}

// DefaultThresholds returns sensible defaults for most systems
func DefaultThresholds() *SuccessThresholds {
	return &SuccessThresholds{
		MinSuccessRate: 0.95, // 95%
		CriticalValidators: []string{
			"goroutine-limit",
			"recursion-depth",
			"infinite-loop",
			"memory-limit",
		},
		MaxFailedIterations: 0, // 0 = use MinSuccessRate
	}
}

// StrictThresholds returns strict thresholds for critical systems
func StrictThresholds() *SuccessThresholds {
	return &SuccessThresholds{
		MinSuccessRate:              1.0, // 100%
		RequireAllValidatorsPassing: true,
		CriticalValidators: []string{
			"goroutine-limit",
			"recursion-depth",
			"infinite-loop",
			"memory-limit",
			"panic-recovery",
		},
	}
}

// RelaxedThresholds returns relaxed thresholds for experimental features
func RelaxedThresholds() *SuccessThresholds {
	return &SuccessThresholds{
		MinSuccessRate:      0.80, // 80%
		CriticalValidators:  []string{"goroutine-limit"},
		MaxFailedIterations: 200,
	}
}

// Validate checks if thresholds are valid
func (t *SuccessThresholds) Validate() error {
	if t.MinSuccessRate < 0.0 || t.MinSuccessRate > 1.0 {
		return fmt.Errorf("min_success_rate must be between 0.0 and 1.0")
	}
	if t.MaxFailedIterations < 0 {
		return fmt.Errorf("max_failed_iterations must be >= 0")
	}

	return nil
}
