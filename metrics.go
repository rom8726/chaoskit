package chaoskit

import (
	"sync"
	"time"
)

// MetricsCollector collects execution metrics
type MetricsCollector struct {
	mu              sync.RWMutex
	totalExecutions int
	successCount    int
	failureCount    int
	totalDuration   time.Duration
	injectorMetrics map[string]map[string]interface{} // injector name -> metrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		injectorMetrics: make(map[string]map[string]interface{}),
	}
}

// RecordExecution records an execution result
func (m *MetricsCollector) RecordExecution(result ExecutionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalExecutions++
	m.totalDuration += result.Duration

	if result.Success {
		m.successCount++
	} else {
		m.failureCount++
	}
}

// Stats returns current statistics
func (m *MetricsCollector) Stats() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgDuration := time.Duration(0)
	if m.totalExecutions > 0 {
		avgDuration = m.totalDuration / time.Duration(m.totalExecutions)
	}

	return map[string]any{
		"total_executions": m.totalExecutions,
		"success_count":    m.successCount,
		"failure_count":    m.failureCount,
		"avg_duration_ms":  avgDuration.Milliseconds(),
		"injector_metrics": m.injectorMetrics,
	}
}

// RecordInjectorMetrics records metrics from an injector
func (m *MetricsCollector) RecordInjectorMetrics(injectorName string, metrics map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.injectorMetrics[injectorName] = metrics
}

// GetInjectorMetrics returns metrics for a specific injector
func (m *MetricsCollector) GetInjectorMetrics(injectorName string) (map[string]interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, ok := m.injectorMetrics[injectorName]

	return metrics, ok
}
