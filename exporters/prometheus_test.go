package exporters

import (
	"strings"
	"testing"
	"time"

	"github.com/rom8726/chaoskit"
)

func TestPrometheusExporter_RecordExecution(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	// Record successful execution
	exporter.RecordExecution(chaoskit.ExecutionResult{
		ScenarioName: "test-scenario",
		Success:      true,
		Duration:     100 * time.Millisecond,
		Timestamp:    time.Now(),
	})

	// Record failed execution
	exporter.RecordExecution(chaoskit.ExecutionResult{
		ScenarioName: "test-scenario",
		Success:      false,
		Duration:     200 * time.Millisecond,
		Timestamp:    time.Now(),
	})

	metrics := exporter.Export()

	// Check that metrics contain expected data
	if !strings.Contains(metrics, "chaoskit_executions_total") {
		t.Error("Expected chaoskit_executions_total metric")
	}

	if !strings.Contains(metrics, "test-scenario") {
		t.Error("Expected test-scenario label")
	}

	if !strings.Contains(metrics, "result=\"success\"") {
		t.Error("Expected success result label")
	}

	if !strings.Contains(metrics, "result=\"failure\"") {
		t.Error("Expected failure result label")
	}
}

func TestPrometheusExporter_SuccessRate(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	// Record 7 successes and 3 failures
	for i := 0; i < 7; i++ {
		exporter.RecordExecution(chaoskit.ExecutionResult{
			ScenarioName: "test",
			Success:      true,
			Duration:     10 * time.Millisecond,
			Timestamp:    time.Now(),
		})
	}

	for i := 0; i < 3; i++ {
		exporter.RecordExecution(chaoskit.ExecutionResult{
			ScenarioName: "test",
			Success:      false,
			Duration:     10 * time.Millisecond,
			Timestamp:    time.Now(),
		})
	}

	metrics := exporter.Export()

	// Success rate should be 0.7 (7/10)
	if !strings.Contains(metrics, "chaoskit_success_rate") {
		t.Error("Expected chaoskit_success_rate metric")
	}

	// Check approximate success rate (0.70)
	if !strings.Contains(metrics, "0.7") {
		t.Error("Expected success rate around 0.7")
	}
}

func TestPrometheusExporter_DurationHistogram(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	durations := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, d := range durations {
		exporter.RecordExecution(chaoskit.ExecutionResult{
			ScenarioName: "perf-test",
			Success:      true,
			Duration:     d,
			Timestamp:    time.Now(),
		})
	}

	metrics := exporter.Export()

	// Check histogram metrics
	if !strings.Contains(metrics, "chaoskit_execution_duration_seconds_bucket") {
		t.Error("Expected histogram bucket metric")
	}

	if !strings.Contains(metrics, "chaoskit_execution_duration_seconds_sum") {
		t.Error("Expected histogram sum metric")
	}

	if !strings.Contains(metrics, "chaoskit_execution_duration_seconds_count") {
		t.Error("Expected histogram count metric")
	}

	// Check bucket labels
	if !strings.Contains(metrics, "le=\"0.001\"") {
		t.Error("Expected 1ms bucket")
	}

	if !strings.Contains(metrics, "le=\"+Inf\"") {
		t.Error("Expected +Inf bucket")
	}
}

func TestPrometheusExporter_InjectorMetrics(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	exporter.RecordInjectorMetrics("delay_injector", map[string]interface{}{
		"delay_count": 42,
		"probability": 0.25,
		"stopped":     false,
	})

	exporter.RecordInjectorMetrics("panic_injector", map[string]interface{}{
		"panic_count": 5,
		"probability": 0.1,
		"stopped":     true,
	})

	metrics := exporter.Export()

	// Check injector metrics
	if !strings.Contains(metrics, "chaoskit_injector_active") {
		t.Error("Expected injector active metric")
	}

	if !strings.Contains(metrics, "chaoskit_injector_operations_total") {
		t.Error("Expected injector operations metric")
	}

	if !strings.Contains(metrics, "chaoskit_injector_probability") {
		t.Error("Expected injector probability metric")
	}

	// Check specific values
	if !strings.Contains(metrics, "delay_injector") {
		t.Error("Expected delay_injector label")
	}

	if !strings.Contains(metrics, "panic_injector") {
		t.Error("Expected panic_injector label")
	}
}

func TestPrometheusExporter_ValidatorMetrics(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	// Record validator executions
	exporter.RecordValidatorMetrics("goroutine_validator", false, false)
	exporter.RecordValidatorMetrics("goroutine_validator", false, false)
	exporter.RecordValidatorMetrics("goroutine_validator", true, false) // failure
	exporter.RecordValidatorMetrics("memory_validator", false, true)    // warning

	metrics := exporter.Export()

	// Check validator metrics
	if !strings.Contains(metrics, "chaoskit_validator_checks_total") {
		t.Error("Expected validator checks metric")
	}

	if !strings.Contains(metrics, "chaoskit_validator_failures_total") {
		t.Error("Expected validator failures metric")
	}

	if !strings.Contains(metrics, "chaoskit_validator_warnings_total") {
		t.Error("Expected validator warnings metric")
	}

	if !strings.Contains(metrics, "goroutine_validator") {
		t.Error("Expected goroutine_validator label")
	}

	if !strings.Contains(metrics, "memory_validator") {
		t.Error("Expected memory_validator label")
	}
}

func TestPrometheusExporter_SystemMetrics(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	// Record some data
	exporter.RecordExecution(chaoskit.ExecutionResult{
		ScenarioName: "scenario1",
		Success:      true,
		Duration:     10 * time.Millisecond,
	})

	exporter.RecordExecution(chaoskit.ExecutionResult{
		ScenarioName: "scenario2",
		Success:      true,
		Duration:     20 * time.Millisecond,
	})

	exporter.RecordInjectorMetrics("injector1", map[string]interface{}{})

	metrics := exporter.Export()

	// Check system metrics
	if !strings.Contains(metrics, "chaoskit_uptime_seconds") {
		t.Error("Expected uptime metric")
	}

	if !strings.Contains(metrics, "chaoskit_scenarios_total") {
		t.Error("Expected scenarios total metric")
	}

	if !strings.Contains(metrics, "chaoskit_injectors_total") {
		t.Error("Expected injectors total metric")
	}

	// Check values
	if !strings.Contains(metrics, "chaoskit_scenarios_total 2") {
		t.Error("Expected 2 scenarios")
	}

	if !strings.Contains(metrics, "chaoskit_injectors_total 1") {
		t.Error("Expected 1 injector")
	}
}

func TestPrometheusExporter_EmptyScenarioName(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	exporter.RecordExecution(chaoskit.ExecutionResult{
		ScenarioName: "", // Empty name
		Success:      true,
		Duration:     10 * time.Millisecond,
	})

	metrics := exporter.Export()

	// Should use "unknown" as default
	if !strings.Contains(metrics, "scenario=\"unknown\"") {
		t.Error("Expected 'unknown' as default scenario name")
	}
}

func TestPrometheusExporter_MultipleScenarios(t *testing.T) {
	exporter := NewPrometheusExporter("chaoskit", "test")

	scenarios := []string{"scenario-a", "scenario-b", "scenario-c"}

	for _, scenario := range scenarios {
		exporter.RecordExecution(chaoskit.ExecutionResult{
			ScenarioName: scenario,
			Success:      true,
			Duration:     10 * time.Millisecond,
		})
	}

	metrics := exporter.Export()

	// All scenarios should be present
	for _, scenario := range scenarios {
		if !strings.Contains(metrics, scenario) {
			t.Errorf("Expected scenario %s in metrics", scenario)
		}
	}
}
