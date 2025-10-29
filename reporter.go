package chaoskit

import (
	"fmt"
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

// GenerateReport generates a summary report
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
