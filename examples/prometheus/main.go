package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/exporters"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
)

// WorkflowEngine implements Target interface
type WorkflowEngine struct{}

func (w *WorkflowEngine) Name() string                       { return "workflow-engine" }
func (w *WorkflowEngine) Setup(ctx context.Context) error    { return nil }
func (w *WorkflowEngine) Teardown(ctx context.Context) error { return nil }

// ExecuteWorkflow is a step that uses chaos context
func ExecuteWorkflow(ctx context.Context, target chaoskit.Target) error {
	// Inject chaos through context
	chaoskit.MaybePanic(ctx)

	chaoskit.MaybeDelay(ctx)

	// Simulate work
	time.Sleep(10 * time.Millisecond)

	return nil
}

func main() {
	// Create Prometheus exporter
	promExporter := exporters.NewPrometheusExporter("chaoskit", "example")

	// Start HTTP server for /metrics endpoint
	go func() {
		http.Handle("/metrics", promExporter.Handler())
		fmt.Println("Prometheus metrics available at http://localhost:9090/metrics")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Create chaos scenario
	engine := &WorkflowEngine{}
	scenario := chaoskit.NewScenario("reliability-test").
		WithTarget(engine).
		Step("execute-workflow", ExecuteWorkflow).
		Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
		Inject("panic", injectors.PanicProbability(0.05)).
		Assert("goroutine-limit", validators.GoroutineLimit(200)).
		Assert("recursion-depth", validators.RecursionDepthLimit(100)).
		Repeat(100).
		Build()

	// Create executor
	executor := chaoskit.NewExecutor(
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)

	// Run scenario and feed metrics to Prometheus exporter
	ctx := context.Background()
	fmt.Println("Running chaos scenario...")

	if err := executor.Run(ctx, scenario); err != nil {
		log.Printf("Scenario failed: %v", err)
	}

	// Export metrics to Prometheus exporter
	for _, result := range executor.Reporter().Results() {
		promExporter.RecordExecution(result)
	}

	// Export injector metrics
	stats := executor.Metrics().Stats()
	if injectorMetrics, ok := stats["injector_metrics"].(map[string]map[string]interface{}); ok {
		for name, metrics := range injectorMetrics {
			promExporter.RecordInjectorMetrics(name, metrics)
		}
	}

	// Print text report
	fmt.Println("\n" + executor.Reporter().GenerateReport())

	// Print sample of Prometheus metrics
	fmt.Println("\n=== Sample Prometheus Metrics ===")
	metrics := promExporter.Export()
	lines := 0
	for _, line := range splitLines(metrics) {
		fmt.Println(line)
		lines++
		if lines >= 30 { // Print first 30 lines
			fmt.Println("... (see http://localhost:9090/metrics for full output)")

			break
		}
	}

	// Keep server running
	fmt.Println("\nPress Ctrl+C to stop...")
	select {}
}

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}

	return result
}
