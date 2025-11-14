package chaoskit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Reporter generates execution reports
type Reporter struct {
	mu      sync.Mutex
	results []ExecutionResult
}

// NewReporter creates a new reporter
func NewReporter() *Reporter {
	return &Reporter{
		results: make([]ExecutionResult, 0),
	}
}

// AddResult adds an execution result
func (r *Reporter) AddResult(result ExecutionResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, result)
}

// Results returns a copy of accumulated results
func (r *Reporter) Results() []ExecutionResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]ExecutionResult, len(r.results))
	copy(out, r.results)

	return out
}

// GenerateReport generates a human-readable summary report
func (r *Reporter) GenerateReport() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.results) == 0 {
		return "No executions recorded"
	}

	success := 0
	failed := 0
	var totalDuration time.Duration

	for _, result := range r.results {
		if result.Success {
			success++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	avgDuration := totalDuration / time.Duration(len(r.results))

	return fmt.Sprintf(
		"ChaosKit Execution Report\n"+
			"========================\n"+
			"Total Executions: %d\n"+
			"Success: %d\n"+
			"Failed: %d\n"+
			"Success Rate: %.2f%%\n"+
			"Average Duration: %v\n",
		len(r.results),
		success,
		failed,
		float64(success)/float64(len(r.results))*100,
		avgDuration,
	)
}

// GenerateJSON returns a JSON report with aggregate stats and executions
func (r *Reporter) GenerateJSON() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	type jsonResult struct {
		Scenario   string    `json:"scenario"`
		Success    bool      `json:"success"`
		Error      string    `json:"error,omitempty"`
		DurationMs int64     `json:"duration_ms"`
		Steps      int       `json:"steps_executed"`
		Timestamp  time.Time `json:"timestamp"`
	}

	stats := struct {
		Total       int          `json:"total_executions"`
		Success     int          `json:"success_count"`
		Failed      int          `json:"failure_count"`
		AvgDuration int64        `json:"avg_duration_ms"`
		Executions  []jsonResult `json:"executions"`
	}{
		Executions: make([]jsonResult, 0, len(r.results)),
	}

	var totalDuration time.Duration
	for _, res := range r.results {
		jr := jsonResult{
			Scenario:   res.ScenarioName,
			Success:    res.Success,
			DurationMs: res.Duration.Milliseconds(),
			Steps:      res.StepsExecuted,
			Timestamp:  res.Timestamp,
		}
		if res.Error != nil {
			jr.Error = res.Error.Error()
		}
		stats.Executions = append(stats.Executions, jr)
		if res.Success {
			stats.Success++
		} else {
			stats.Failed++
		}
		totalDuration += res.Duration
	}
	stats.Total = len(r.results)
	if stats.Total > 0 {
		stats.AvgDuration = (totalDuration / time.Duration(stats.Total)).Milliseconds()
	}

	b, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// SaveJSON writes the JSON report to a file
func (r *Reporter) SaveJSON(path string) error {
	jsonStr, err := r.GenerateJSON()
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(jsonStr), 0644)
}

// GetVerdict calculates verdict based on thresholds
func (r *Reporter) GetVerdict(thresholds *SuccessThresholds) (*Report, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.results) == 0 {
		return nil, fmt.Errorf("no execution results available")
	}

	if err := thresholds.Validate(); err != nil {
		return nil, fmt.Errorf("invalid thresholds: %w", err)
	}

	report := &Report{
		ExecutionTime:   time.Now(),
		TotalIterations: len(r.results),
		Thresholds:      thresholds,
	}

	// Extract scenario name from the first result
	if len(r.results) > 0 {
		report.ScenarioName = r.results[0].ScenarioName
	}

	// Calculate statistics
	var totalDuration time.Duration
	for _, result := range r.results {
		if result.Success {
			report.SuccessCount++
		} else {
			report.FailureCount++
		}
		totalDuration += result.Duration
	}

	report.SuccessRate = float64(report.SuccessCount) / float64(report.TotalIterations)
	if report.TotalIterations > 0 {
		report.AvgDuration = totalDuration / time.Duration(report.TotalIterations)
	}
	report.Duration = totalDuration

	// Analyze failures and categorize by severity
	report.Analysis = r.analyzeFailures()
	report.CriticalFailures = r.categorizeFailures(SeverityCritical, thresholds)
	report.Warnings = r.categorizeFailures(SeverityWarning, thresholds)
	report.InfoMessages = r.categorizeFailures(SeverityInfo, thresholds)

	// Determine verdict
	report.Verdict = r.determineVerdict(report, thresholds)
	report.Summary = r.generateSummary(report)

	return report, nil
}

// determineVerdict applies thresholds to determine verdict
func (r *Reporter) determineVerdict(report *Report, thresholds *SuccessThresholds) Verdict {
	// Check critical failures
	if len(report.CriticalFailures) > 0 {
		return VerdictFail
	}

	// Check the success rate
	if report.SuccessRate < thresholds.MinSuccessRate {
		return VerdictFail
	}

	// Check max failed iterations (if set)
	if thresholds.MaxFailedIterations > 0 && report.FailureCount > thresholds.MaxFailedIterations {
		return VerdictFail
	}

	// Check if all validators must pass
	if thresholds.RequireAllValidatorsPassing && report.FailureCount > 0 {
		return VerdictFail
	}

	// Check warnings
	if len(report.Warnings) > 0 {
		return VerdictUnstable
	}

	return VerdictPass
}

// generateSummary creates human-readable summary
func (r *Reporter) generateSummary(report *Report) string {
	switch report.Verdict {
	case VerdictPass:
		return fmt.Sprintf("All tests passed. Success rate: %.2f%% (%d/%d iterations)",
			report.SuccessRate*100, report.SuccessCount, report.TotalIterations)

	case VerdictUnstable:
		return fmt.Sprintf("Tests completed with warnings. Success rate: %.2f%% (%d/%d). %d warnings detected.",
			report.SuccessRate*100, report.SuccessCount, report.TotalIterations, len(report.Warnings))

	case VerdictFail:
		reasons := []string{}
		if len(report.CriticalFailures) > 0 {
			reasons = append(reasons, fmt.Sprintf("%d critical validator(s) failed", len(report.CriticalFailures)))
		}
		if report.SuccessRate < report.Thresholds.MinSuccessRate {
			reasons = append(reasons, fmt.Sprintf("success rate %.2f%% below threshold %.2f%%",
				report.SuccessRate*100, report.Thresholds.MinSuccessRate*100))
		}

		return fmt.Sprintf("Tests failed: %s", strings.Join(reasons, ", "))

	default:
		return "Unknown verdict"
	}
}

// analyzeFailures performs detailed failure analysis
func (r *Reporter) analyzeFailures() *FailureAnalysis {
	analysis := &FailureAnalysis{
		ByValidator: make(map[string]int),
		ByType:      make(map[string]int),
		TopErrors:   make([]ErrorSummary, 0),
	}

	errorCounts := make(map[string]int)

	for _, result := range r.results {
		if result.Error != nil {
			// Extract validator name from error
			validatorName := extractValidatorName(result.Error)
			analysis.ByValidator[validatorName]++

			// Extract error type
			errorType := classifyError(result.Error)
			analysis.ByType[errorType]++

			// Count error patterns
			errorPattern := normalizeError(result.Error)
			errorCounts[errorPattern]++
		}
	}

	// Build top errors list
	type errorCount struct {
		pattern string
		count   int
	}
	var errors []errorCount
	for pattern, count := range errorCounts {
		errors = append(errors, errorCount{pattern, count})
	}
	sort.Slice(errors, func(i, j int) bool {
		return errors[i].count > errors[j].count
	})

	// Take top 5
	for i := 0; i < len(errors) && i < 5; i++ {
		analysis.TopErrors = append(analysis.TopErrors, ErrorSummary{
			ErrorPattern: errors[i].pattern,
			Count:        errors[i].count,
		})
	}

	return analysis
}

// categorizeFailures groups failures by severity
func (r *Reporter) categorizeFailures(severity ValidationSeverity, thresholds *SuccessThresholds) []ValidationFailure {
	failures := make(map[string]*ValidationFailure)

	for _, result := range r.results {
		if result.Error == nil {
			continue
		}

		validatorName := extractValidatorName(result.Error)

		// Determine severity based on thresholds
		failureSeverity := r.getValidatorSeverity(validatorName, thresholds)

		if failureSeverity != severity {
			continue
		}

		if existing, ok := failures[validatorName]; ok {
			existing.Occurrences++
			if result.Timestamp.After(existing.LastSeen) {
				existing.LastSeen = result.Timestamp
			}
			if result.Timestamp.Before(existing.FirstSeen) {
				existing.FirstSeen = result.Timestamp
			}
		} else {
			failures[validatorName] = &ValidationFailure{
				ValidatorName: validatorName,
				Severity:      failureSeverity,
				Message:       result.Error.Error(),
				Occurrences:   1,
				FirstSeen:     result.Timestamp,
				LastSeen:      result.Timestamp,
			}
		}
	}

	// Convert map to slice
	var result []ValidationFailure
	for _, failure := range failures {
		result = append(result, *failure)
	}

	// Sort by occurrences (most common first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Occurrences > result[j].Occurrences
	})

	return result
}

// getValidatorSeverity determines validator severity from thresholds
func (r *Reporter) getValidatorSeverity(validatorName string, thresholds *SuccessThresholds) ValidationSeverity {
	// Normalize validator name for matching (e.g., "goroutine_limit_100" -> ValidatorGoroutineLimit)
	normalizedName := normalizeValidatorName(validatorName)

	// Check if critical
	for _, critical := range thresholds.CriticalValidators {
		if normalizedName == critical || validatorName == critical {
			return SeverityCritical
		}
	}

	// Check if warning
	for _, warning := range thresholds.WarningValidators {
		if normalizedName == warning || validatorName == warning {
			return SeverityWarning
		}
	}

	// Default to info
	return SeverityInfo
}

// normalizeValidatorName converts validator names to canonical form for matching
// Examples:
//   - "goroutine_limit_100" -> ValidatorGoroutineLimit
//   - "recursion_depth_limit_10" -> ValidatorRecursionDepth
//   - "no_infinite_loop_5s" -> ValidatorInfiniteLoop
//   - "memory_under_100MB" -> ValidatorMemoryLimit
//   - "no_panics_5" -> ValidatorPanicRecovery
func normalizeValidatorName(name string) string {
	name = strings.ToLower(name)

	// Remove numeric suffixes and common prefixes
	re := regexp.MustCompile(`_\d+[a-z]*$`)
	name = re.ReplaceAllString(name, "")

	// Remove common prefixes
	prefixes := []string{"no_", "no-"}
	for _, prefix := range prefixes {
		name = strings.TrimPrefix(name, prefix)
	}

	// Replace underscores with hyphens
	name = strings.ReplaceAll(name, "_", "-")

	// Handle specific mappings
	mappings := map[string]string{
		ValidatorGoroutineLimit:      ValidatorGoroutineLimit,
		ValidatorRecursionDepthLimit: ValidatorRecursionDepth,
		ValidatorInfiniteLoop:        ValidatorInfiniteLoop,
		ValidatorMemoryUnder:         ValidatorMemoryLimit,
		ValidatorPanics:              ValidatorPanicRecovery,
		ValidatorExecutionTime:       ValidatorExecutionTime,
	}

	// Check if name matches any mapping key
	for key, value := range mappings {
		if strings.Contains(name, key) {
			return value
		}
	}

	// If name contains key parts, map them
	if strings.Contains(name, "goroutine") {
		return ValidatorGoroutineLimit
	}
	if strings.Contains(name, "recursion") {
		return ValidatorRecursionDepth
	}
	if strings.Contains(name, "infinite") || strings.Contains(name, "loop") {
		return ValidatorInfiniteLoop
	}
	if strings.Contains(name, "memory") {
		return ValidatorMemoryLimit
	}
	if strings.Contains(name, "panic") {
		return ValidatorPanicRecovery
	}
	if strings.Contains(name, "execution") || strings.Contains(name, "time") {
		return ValidatorExecutionTime
	}

	return name
}

// Helper functions

func extractValidatorName(err error) string {
	// Extract validator name from error message
	// Format: "validator X failed: ..."
	msg := err.Error()
	if idx := strings.Index(msg, " failed:"); idx > 0 {
		if strings.HasPrefix(msg, "validator ") {
			return strings.TrimSpace(msg[10:idx])
		}
	}

	return ErrorTypeUnknown
}

func classifyError(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "goroutine"):
		return ErrorTypeGoroutineLeak
	case strings.Contains(msg, "panic"):
		return ErrorTypePanic
	case strings.Contains(msg, "recursion"):
		return ErrorTypeRecursion
	case strings.Contains(msg, "timeout"):
		return ErrorTypeTimeout
	case strings.Contains(msg, "memory"):
		return ErrorTypeMemory
	default:
		return ErrorTypeOther
	}
}

func normalizeError(err error) string {
	// Remove numbers and specific values to create pattern
	msg := err.Error()
	// Replace numbers with placeholder
	re := regexp.MustCompile(`\d+`)

	return re.ReplaceAllString(msg, "N")
}

// GenerateTextReport generates enhanced human-readable report
func (r *Reporter) GenerateTextReport(report *Report) string {
	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "=== ChaosKit Test Report ===\n")
	_, _ = fmt.Fprintf(&buf, "Scenario: %s\n", report.ScenarioName)
	_, _ = fmt.Fprintf(&buf, "Executed: %s\n", report.ExecutionTime.Format(time.RFC3339))
	_, _ = fmt.Fprintf(&buf, "Duration: %s\n\n", report.Duration)

	// Verdict
	icon := "✅"
	switch report.Verdict {
	case VerdictFail:
		icon = "❌"
	case VerdictUnstable:
		icon = "⚠️"
	}
	_, _ = fmt.Fprintf(&buf, "%s VERDICT: %s\n", icon, report.Verdict)
	_, _ = fmt.Fprintf(&buf, "%s\n\n", report.Summary)

	// Statistics
	_, _ = fmt.Fprintf(&buf, "Statistics:\n")
	_, _ = fmt.Fprintf(&buf, "  Total Iterations: %d\n", report.TotalIterations)
	_, _ = fmt.Fprintf(&buf, "  Success: %d (%.2f%%)\n", report.SuccessCount, report.SuccessRate*100)
	_, _ = fmt.Fprintf(&buf, "  Failures: %d\n", report.FailureCount)
	_, _ = fmt.Fprintf(&buf, "  Avg Duration: %s\n\n", report.AvgDuration)

	// Critical failures
	if len(report.CriticalFailures) > 0 {
		_, _ = fmt.Fprintf(&buf, "🔴 Critical Failures:\n")
		for _, failure := range report.CriticalFailures {
			_, _ = fmt.Fprintf(&buf, "  - %s: %s (occurred %d times)\n",
				failure.ValidatorName, failure.Message, failure.Occurrences)
		}
		_, _ = fmt.Fprintf(&buf, "\n")
	}

	// Warnings
	if len(report.Warnings) > 0 {
		_, _ = fmt.Fprintf(&buf, "⚠️  Warnings:\n")
		for _, warning := range report.Warnings {
			_, _ = fmt.Fprintf(&buf, "  - %s: %s (occurred %d times)\n",
				warning.ValidatorName, warning.Message, warning.Occurrences)
		}
		_, _ = fmt.Fprintf(&buf, "\n")
	}

	// Failure analysis
	if report.Analysis != nil && len(report.Analysis.TopErrors) > 0 {
		_, _ = fmt.Fprintf(&buf, "Top Errors:\n")
		for i, err := range report.Analysis.TopErrors {
			_, _ = fmt.Fprintf(&buf, "  %d. %s (%d occurrences)\n", i+1, err.ErrorPattern, err.Count)
		}
		_, _ = fmt.Fprintf(&buf, "\n")
	}

	// Action items
	switch report.Verdict {
	case VerdictFail:
		_, _ = fmt.Fprintf(&buf, "Action Required:\n")
		_, _ = fmt.Fprintf(&buf, "  🚫 DO NOT DEPLOY - Fix critical issues before release\n")
		for _, failure := range report.CriticalFailures {
			_, _ = fmt.Fprintf(&buf, "  - Fix: %s\n", failure.ValidatorName)
		}
	case VerdictUnstable:
		_, _ = fmt.Fprintf(&buf, "Action Recommended:\n")
		_, _ = fmt.Fprintf(&buf, "  ⚠️  Review warnings before deployment\n")
		for _, warning := range report.Warnings {
			_, _ = fmt.Fprintf(&buf, "  - Review: %s\n", warning.ValidatorName)
		}
	}

	return buf.String()
}
