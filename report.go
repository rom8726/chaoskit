package chaoskit

import "time"

// Report contains comprehensive test results with verdict
type Report struct {
	// Verdict is the overall test outcome
	Verdict Verdict `json:"verdict"`

	// Summary is human-readable verdict explanation
	Summary string `json:"summary"`

	// ScenarioName is the name of tested scenario
	ScenarioName string `json:"scenario_name"`

	// ExecutionTime is when test was executed
	ExecutionTime time.Time `json:"execution_time"`

	// Duration is total test duration
	Duration time.Duration `json:"duration"`

	// Statistics
	TotalIterations int           `json:"total_iterations"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	SuccessRate     float64       `json:"success_rate"`
	AvgDuration     time.Duration `json:"avg_duration"`

	// Failures categorized by severity
	CriticalFailures []ValidationFailure `json:"critical_failures"`
	Warnings         []ValidationFailure `json:"warnings"`
	InfoMessages     []ValidationFailure `json:"info_messages,omitempty"`

	// Failure analysis
	Analysis *FailureAnalysis `json:"analysis,omitempty"`

	// Thresholds used for evaluation
	Thresholds *SuccessThresholds `json:"thresholds,omitempty"`
}

// ValidationFailure represents a validator failure
type ValidationFailure struct {
	ValidatorName string             `json:"validator_name"`
	Severity      ValidationSeverity `json:"severity"`
	Message       string             `json:"message"`
	Occurrences   int                `json:"occurrences"`
	FirstSeen     time.Time          `json:"first_seen"`
	LastSeen      time.Time          `json:"last_seen"`
	Details       map[string]any     `json:"details,omitempty"`
}

// FailureAnalysis provides detailed failure breakdown
type FailureAnalysis struct {
	// ByValidator counts failures per validator
	ByValidator map[string]int `json:"by_validator"`

	// ByType groups failures by error type
	ByType map[string]int `json:"by_type"`

	// TopErrors lists most common errors
	TopErrors []ErrorSummary `json:"top_errors"`

	// FailureRate over time (if duration-based test)
	FailureRateOverTime []TimeWindow `json:"failure_rate_over_time,omitempty"`
}

// ErrorSummary represents a common error pattern
type ErrorSummary struct {
	ErrorPattern string             `json:"error_pattern"`
	Count        int                `json:"count"`
	Severity     ValidationSeverity `json:"severity"`
	Examples     []string           `json:"examples,omitempty"`
}

// TimeWindow represents the failure rate in a time period
type TimeWindow struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Iterations  int       `json:"iterations"`
	Failures    int       `json:"failures"`
	FailureRate float64   `json:"failure_rate"`
}
