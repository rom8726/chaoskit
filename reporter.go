package chaoskit

import (
	"encoding/json"
	"fmt"
	"os"
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
