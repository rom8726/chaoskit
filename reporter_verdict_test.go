package chaoskit

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReporter_GetVerdict_Pass(t *testing.T) {
	reporter := NewReporter()

	// Add successful results
	for i := 0; i < 100; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Duration:     10 * time.Millisecond,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	thresholds := DefaultThresholds()
	report, err := reporter.GetVerdict(thresholds)

	assert.NoError(t, err)
	assert.Equal(t, VerdictPass, report.Verdict)
	assert.Equal(t, 1.0, report.SuccessRate)
	assert.Empty(t, report.CriticalFailures)
	assert.Equal(t, 100, report.TotalIterations)
	assert.Equal(t, 100, report.SuccessCount)
	assert.Equal(t, 0, report.FailureCount)
}

func TestReporter_GetVerdict_Fail_CriticalValidator(t *testing.T) {
	reporter := NewReporter()

	// Add one critical failure
	reporter.AddResult(ExecutionResult{
		Success:      false,
		Error:        fmt.Errorf("validator goroutine_limit_100 failed: exceeded limit"),
		Timestamp:    time.Now(),
		ScenarioName: "test-scenario",
	})

	// Add 99 successes
	for i := 0; i < 99; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	thresholds := DefaultThresholds()
	report, err := reporter.GetVerdict(thresholds)

	assert.NoError(t, err)
	assert.Equal(t, VerdictFail, report.Verdict)
	assert.Len(t, report.CriticalFailures, 1)
	assert.Contains(t, report.CriticalFailures[0].ValidatorName, "goroutine")
}

func TestReporter_GetVerdict_Fail_LowSuccessRate(t *testing.T) {
	reporter := NewReporter()

	// Add 50 failures and 50 successes (50% success rate)
	for i := 0; i < 50; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      false,
			Error:        fmt.Errorf("some error"),
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}
	for i := 0; i < 50; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	thresholds := DefaultThresholds()
	report, err := reporter.GetVerdict(thresholds)

	assert.NoError(t, err)
	assert.Equal(t, VerdictFail, report.Verdict)
	assert.Equal(t, 0.5, report.SuccessRate)
}

func TestReporter_GetVerdict_Unstable_Warnings(t *testing.T) {
	reporter := NewReporter()

	// Add 100 successes
	for i := 0; i < 100; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	// Add one warning failure
	reporter.AddResult(ExecutionResult{
		Success:      false,
		Error:        fmt.Errorf("validator execution_time_10ms_100ms failed: too slow"),
		Timestamp:    time.Now(),
		ScenarioName: "test-scenario",
	})

	thresholds := DefaultThresholds()
	thresholds.WarningValidators = []string{"execution-time"}
	report, err := reporter.GetVerdict(thresholds)

	assert.NoError(t, err)
	assert.Equal(t, VerdictUnstable, report.Verdict)
	assert.Len(t, report.Warnings, 1)
}

func TestReporter_GenerateTextReport(t *testing.T) {
	reporter := NewReporter()

	// Add some results
	for i := 0; i < 10; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Duration:     10 * time.Millisecond,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	thresholds := DefaultThresholds()
	report, err := reporter.GetVerdict(thresholds)
	assert.NoError(t, err)

	textReport := reporter.GenerateTextReport(report)
	assert.Contains(t, textReport, "ChaosKit Test Report")
	assert.Contains(t, textReport, "test-scenario")
	assert.Contains(t, textReport, "VERDICT")
}

func TestReporter_GenerateJUnitXML(t *testing.T) {
	reporter := NewReporter()

	// Add some results
	for i := 0; i < 10; i++ {
		reporter.AddResult(ExecutionResult{
			Success:      true,
			Duration:     10 * time.Millisecond,
			Timestamp:    time.Now(),
			ScenarioName: "test-scenario",
		})
	}

	thresholds := DefaultThresholds()
	report, err := reporter.GetVerdict(thresholds)
	assert.NoError(t, err)

	xmlStr, err := reporter.GenerateJUnitXML(report)
	assert.NoError(t, err)
	assert.Contains(t, xmlStr, "<?xml version")
	assert.Contains(t, xmlStr, "<testsuite")
	assert.Contains(t, xmlStr, "test-scenario")
}
