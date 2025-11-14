package exporters

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rom8726/chaoskit"
)

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	mu               sync.RWMutex
	namespace        string
	subsystem        string
	executions       map[string]*executionMetrics
	injectorMetrics  map[string]map[string]any
	validatorMetrics map[string]*validatorMetrics
	startTime        time.Time
}

type executionMetrics struct {
	total           int64
	success         int64
	failure         int64
	durationBuckets map[float64]int64 // duration in seconds -> count
	totalDuration   time.Duration
}

type validatorMetrics struct {
	validations int64
	failures    int64
	warnings    int64
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(namespace, subsystem string) *PrometheusExporter {
	return &PrometheusExporter{
		namespace:        namespace,
		subsystem:        subsystem,
		executions:       make(map[string]*executionMetrics),
		injectorMetrics:  make(map[string]map[string]any),
		validatorMetrics: make(map[string]*validatorMetrics),
		startTime:        time.Now(),
	}
}

// RecordExecution records an execution result
func (p *PrometheusExporter) RecordExecution(result chaoskit.ExecutionResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	scenario := result.ScenarioName
	if scenario == "" {
		scenario = "unknown"
	}

	metrics, ok := p.executions[scenario]
	if !ok {
		metrics = &executionMetrics{
			durationBuckets: make(map[float64]int64),
		}
		p.executions[scenario] = metrics
	}

	metrics.total++
	metrics.totalDuration += result.Duration

	if result.Success {
		metrics.success++
	} else {
		metrics.failure++
	}

	// Record duration in histogram buckets
	durationSec := result.Duration.Seconds()
	buckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
	for _, bucket := range buckets {
		if durationSec <= bucket {
			metrics.durationBuckets[bucket]++
		}
	}
}

// RecordInjectorMetrics records metrics from an injector
func (p *PrometheusExporter) RecordInjectorMetrics(injectorName string, metrics map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.injectorMetrics[injectorName] = metrics
}

// RecordValidatorMetrics records validator execution
func (p *PrometheusExporter) RecordValidatorMetrics(validatorName string, failed bool, warning bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metrics, ok := p.validatorMetrics[validatorName]
	if !ok {
		metrics = &validatorMetrics{}
		p.validatorMetrics[validatorName] = metrics
	}

	metrics.validations++
	if failed {
		metrics.failures++
	}
	if warning {
		metrics.warnings++
	}
}

// Export generates Prometheus metrics format
func (p *PrometheusExporter) Export() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var sb strings.Builder

	// Add timestamp comment
	sb.WriteString(fmt.Sprintf("# ChaosKit Metrics Export - %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("# Uptime: %s\n\n", time.Since(p.startTime).Round(time.Second)))

	// Execution metrics
	p.writeExecutionMetrics(&sb)

	// Injector metrics
	p.writeInjectorMetrics(&sb)

	// Validator metrics
	p.writeValidatorMetrics(&sb)

	// System info
	p.writeSystemMetrics(&sb)

	return sb.String()
}

func (p *PrometheusExporter) writeExecutionMetrics(sb *strings.Builder) {
	// Sort scenario names for consistent output
	scenarios := make([]string, 0, len(p.executions))
	for scenario := range p.executions {
		scenarios = append(scenarios, scenario)
	}
	sort.Strings(scenarios)

	// Total executions counter
	sb.WriteString("# HELP chaoskit_executions_total Total number of scenario executions\n")
	sb.WriteString("# TYPE chaoskit_executions_total counter\n")
	for _, scenario := range scenarios {
		metrics := p.executions[scenario]
		_, _ = fmt.Fprintf(sb, "chaoskit_executions_total{scenario=\"%s\",result=\"success\"} %d\n",
			scenario, metrics.success)
		_, _ = fmt.Fprintf(sb, "chaoskit_executions_total{scenario=\"%s\",result=\"failure\"} %d\n",
			scenario, metrics.failure)
	}
	sb.WriteString("\n")

	// Success rate gauge
	sb.WriteString("# HELP chaoskit_success_rate Success rate of scenario executions (0-1)\n")
	sb.WriteString("# TYPE chaoskit_success_rate gauge\n")
	for _, scenario := range scenarios {
		metrics := p.executions[scenario]
		successRate := 0.0
		if metrics.total > 0 {
			successRate = float64(metrics.success) / float64(metrics.total)
		}
		_, _ = fmt.Fprintf(sb, "chaoskit_success_rate{scenario=\"%s\"} %.4f\n",
			scenario, successRate)
	}
	sb.WriteString("\n")

	// Duration histogram
	sb.WriteString("# HELP chaoskit_execution_duration_seconds Duration of scenario executions\n")
	sb.WriteString("# TYPE chaoskit_execution_duration_seconds histogram\n")
	for _, scenario := range scenarios {
		metrics := p.executions[scenario]

		// Write buckets
		buckets := []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}
		cumulativeCount := int64(0)
		for _, bucket := range buckets {
			cumulativeCount += metrics.durationBuckets[bucket]
			_, _ = fmt.Fprintf(sb, "chaoskit_execution_duration_seconds_bucket{scenario=\"%s\",le=\"%.3f\"} %d\n",
				scenario, bucket, cumulativeCount)
		}

		// +Inf bucket
		_, _ = fmt.Fprintf(sb, "chaoskit_execution_duration_seconds_bucket{scenario=\"%s\",le=\"+Inf\"} %d\n",
			scenario, metrics.total)

		// Sum and count
		_, _ = fmt.Fprintf(sb, "chaoskit_execution_duration_seconds_sum{scenario=\"%s\"} %.6f\n",
			scenario, metrics.totalDuration.Seconds())
		_, _ = fmt.Fprintf(sb, "chaoskit_execution_duration_seconds_count{scenario=\"%s\"} %d\n",
			scenario, metrics.total)
	}
	sb.WriteString("\n")

	// Average duration gauge
	sb.WriteString("# HELP chaoskit_execution_duration_avg_seconds Average duration of scenario executions\n")
	sb.WriteString("# TYPE chaoskit_execution_duration_avg_seconds gauge\n")
	for _, scenario := range scenarios {
		metrics := p.executions[scenario]
		avgDuration := 0.0
		if metrics.total > 0 {
			avgDuration = metrics.totalDuration.Seconds() / float64(metrics.total)
		}
		_, _ = fmt.Fprintf(sb, "chaoskit_execution_duration_avg_seconds{scenario=\"%s\"} %.6f\n",
			scenario, avgDuration)
	}
	sb.WriteString("\n")
}

func (p *PrometheusExporter) writeInjectorMetrics(sb *strings.Builder) {
	if len(p.injectorMetrics) == 0 {
		return
	}

	// Sort injector names
	injectors := make([]string, 0, len(p.injectorMetrics))
	for injector := range p.injectorMetrics {
		injectors = append(injectors, injector)
	}
	sort.Strings(injectors)

	sb.WriteString("# HELP chaoskit_injector_active Whether the injector is currently active\n")
	sb.WriteString("# TYPE chaoskit_injector_active gauge\n")
	for _, injector := range injectors {
		metrics := p.injectorMetrics[injector]

		// Check if injector is stopped
		stopped := 0
		if stoppedVal, ok := metrics["stopped"].(bool); ok && stoppedVal {
			stopped = 1
		}
		active := 1 - stopped

		_, _ = fmt.Fprintf(sb, "chaoskit_injector_active{injector=\"%s\"} %d\n",
			sanitizeLabel(injector), active)
	}
	sb.WriteString("\n")

	// Injector-specific metrics
	sb.WriteString("# HELP chaoskit_injector_operations_total Total operations by injector\n")
	sb.WriteString("# TYPE chaoskit_injector_operations_total counter\n")
	for _, injector := range injectors {
		metrics := p.injectorMetrics[injector]

		// Extract common counter fields
		if count, ok := extractInt64(metrics, "delay_count", "cancel_count", "panic_count", "count"); ok {
			_, _ = fmt.Fprintf(sb, "chaoskit_injector_operations_total{injector=\"%s\"} %d\n",
				sanitizeLabel(injector), count)
		}
	}
	sb.WriteString("\n")

	// Probability gauge for injectors
	sb.WriteString("# HELP chaoskit_injector_probability Configured probability for the injector\n")
	sb.WriteString("# TYPE chaoskit_injector_probability gauge\n")
	for _, injector := range injectors {
		metrics := p.injectorMetrics[injector]

		if prob, ok := extractFloat64(metrics, "probability"); ok {
			_, _ = fmt.Fprintf(sb, "chaoskit_injector_probability{injector=\"%s\"} %.4f\n",
				sanitizeLabel(injector), prob)
		}
	}
	sb.WriteString("\n")
}

func (p *PrometheusExporter) writeValidatorMetrics(sb *strings.Builder) {
	if len(p.validatorMetrics) == 0 {
		return
	}

	// Sort validator names
	validators := make([]string, 0, len(p.validatorMetrics))
	for validator := range p.validatorMetrics {
		validators = append(validators, validator)
	}
	sort.Strings(validators)

	sb.WriteString("# HELP chaoskit_validator_checks_total Total number of validator checks\n")
	sb.WriteString("# TYPE chaoskit_validator_checks_total counter\n")
	for _, validator := range validators {
		metrics := p.validatorMetrics[validator]
		_, _ = fmt.Fprintf(sb, "chaoskit_validator_checks_total{validator=\"%s\"} %d\n",
			sanitizeLabel(validator), metrics.validations)
	}
	sb.WriteString("\n")

	sb.WriteString("# HELP chaoskit_validator_failures_total Total number of validator failures\n")
	sb.WriteString("# TYPE chaoskit_validator_failures_total counter\n")
	for _, validator := range validators {
		metrics := p.validatorMetrics[validator]
		_, _ = fmt.Fprintf(sb, "chaoskit_validator_failures_total{validator=\"%s\"} %d\n",
			sanitizeLabel(validator), metrics.failures)
	}
	sb.WriteString("\n")

	sb.WriteString("# HELP chaoskit_validator_warnings_total Total number of validator warnings\n")
	sb.WriteString("# TYPE chaoskit_validator_warnings_total counter\n")
	for _, validator := range validators {
		metrics := p.validatorMetrics[validator]
		_, _ = fmt.Fprintf(sb, "chaoskit_validator_warnings_total{validator=\"%s\"} %d\n",
			sanitizeLabel(validator), metrics.warnings)
	}
	sb.WriteString("\n")
}

func (p *PrometheusExporter) writeSystemMetrics(sb *strings.Builder) {
	sb.WriteString("# HELP chaoskit_uptime_seconds Time since exporter started\n")
	sb.WriteString("# TYPE chaoskit_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(sb, "chaoskit_uptime_seconds %.0f\n", time.Since(p.startTime).Seconds())
	sb.WriteString("\n")

	sb.WriteString("# HELP chaoskit_scenarios_total Total number of scenarios tracked\n")
	sb.WriteString("# TYPE chaoskit_scenarios_total gauge\n")
	_, _ = fmt.Fprintf(sb, "chaoskit_scenarios_total %d\n", len(p.executions))
	sb.WriteString("\n")

	sb.WriteString("# HELP chaoskit_injectors_total Total number of injectors tracked\n")
	sb.WriteString("# TYPE chaoskit_injectors_total gauge\n")
	_, _ = fmt.Fprintf(sb, "chaoskit_injectors_total %d\n", len(p.injectorMetrics))
	sb.WriteString("\n")

	sb.WriteString("# HELP chaoskit_validators_total Total number of validators tracked\n")
	sb.WriteString("# TYPE chaoskit_validators_total gauge\n")
	_, _ = fmt.Fprintf(sb, "chaoskit_validators_total %d\n", len(p.validatorMetrics))
	sb.WriteString("\n")
}

// Helper functions

func sanitizeLabel(label string) string {
	// Replace invalid characters for Prometheus labels
	label = strings.ReplaceAll(label, "\"", "\\\"")
	label = strings.ReplaceAll(label, "\n", "\\n")

	return label
}

func extractInt64(m map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case int:
				return int64(v), true
			case int64:
				return v, true
			case float64:
				return int64(v), true
			}
		}
	}

	return 0, false
}

func extractFloat64(m map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case float64:
				return v, true
			case int:
				return float64(v), true
			case int64:
				return float64(v), true
			}
		}
	}

	return 0, false
}
