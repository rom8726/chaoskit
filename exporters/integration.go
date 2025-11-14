package exporters

import (
	"sync"

	"github.com/rom8726/chaoskit"
)

// ExecutorIntegration provides seamless integration with chaoskit.Executor
type ExecutorIntegration struct {
	exporter *PrometheusExporter
	executor *chaoskit.Executor
	mu       sync.Mutex
}

// NewExecutorIntegration creates integration between Executor and PrometheusExporter
func NewExecutorIntegration(executor *chaoskit.Executor, exporter *PrometheusExporter) *ExecutorIntegration {
	return &ExecutorIntegration{
		exporter: exporter,
		executor: executor,
	}
}

// SyncMetrics syncs all metrics from executor to Prometheus exporter
func (ei *ExecutorIntegration) SyncMetrics() {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	// Sync execution results
	for _, result := range ei.executor.Reporter().Results() {
		ei.exporter.RecordExecution(result)
	}

	// Sync injector metrics
	stats := ei.executor.Metrics().Stats()
	if injectorMetrics, ok := stats["injector_metrics"].(map[string]map[string]interface{}); ok {
		for name, metrics := range injectorMetrics {
			ei.exporter.RecordInjectorMetrics(name, metrics)
		}
	}
}

// Exporter returns the underlying Prometheus exporter
func (ei *ExecutorIntegration) Exporter() *PrometheusExporter {
	return ei.exporter
}
