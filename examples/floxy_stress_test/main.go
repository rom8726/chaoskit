package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rom8726/chaoskit"
	"github.com/rom8726/chaoskit/injectors"
	"github.com/rom8726/chaoskit/validators"
	"github.com/rom8726/floxy"
	"github.com/rom8726/floxy/plugins/engine/metrics"

	rolldepth "github.com/rom8726/floxy/plugins/engine/rollback-depth"
)

// failpoint.Inject("payment-handler-panic", func() { panic("gofail: payment handler panic") })

// FloxyStressTarget wraps Floxy engine as a ChaosKit target
type FloxyStressTarget struct {
	pool              *pgxpool.Pool
	engine            *floxy.Engine
	rollbackPlugin    *rolldepth.RollbackDepthPlugin
	metricsCollector  *FloxyMetricsCollector
	workflowInstances atomic.Int64
	successfulRuns    atomic.Int64
	failedRuns        atomic.Int64
	rollbackCount     atomic.Int64
	maxRollbackDepth  atomic.Int32
	mu                sync.RWMutex
	workers           []context.CancelFunc
}

func NewFloxyStressTarget(connString string) (*FloxyStressTarget, error) {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := floxy.RunMigrations(ctx, pool); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	engine := floxy.NewEngine(pool,
		floxy.WithMissingHandlerCooldown(50*time.Millisecond),
		floxy.WithQueueAgingEnabled(true),
		floxy.WithQueueAgingRate(0.5),
	)

	// Register plugins
	rollbackPlugin := rolldepth.New()
	engine.RegisterPlugin(rollbackPlugin)

	metricsCollector := NewFloxyMetricsCollector()
	metricsPlugin := metrics.New(metricsCollector)
	engine.RegisterPlugin(metricsPlugin)

	target := &FloxyStressTarget{
		pool:             pool,
		engine:           engine,
		rollbackPlugin:   rollbackPlugin,
		metricsCollector: metricsCollector,
	}

	// Register handlers
	target.registerHandlers()

	// Register workflows
	if err := target.registerWorkflows(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to register workflows: %w", err)
	}

	return target, nil
}

func (t *FloxyStressTarget) Name() string {
	return "floxy-stress-test"
}

func (t *FloxyStressTarget) Setup(ctx context.Context) error {
	log.Println("[Floxy] Setting up stress test target...")

	// Start worker pool
	workerCount := 5
	t.mu.Lock()
	t.workers = make([]context.CancelFunc, workerCount)
	for i := 0; i < workerCount; i++ {
		workerCtx, cancel := context.WithCancel(ctx)
		t.workers[i] = cancel

		go t.worker(workerCtx, fmt.Sprintf("worker-%d", i))
	}
	t.mu.Unlock()

	log.Printf("[Floxy] Started %d workers", workerCount)

	return nil
}

func (t *FloxyStressTarget) Teardown(ctx context.Context) error {
	log.Println("[Floxy] Tearing down stress test target...")

	// Stop workers
	t.mu.Lock()
	for _, cancel := range t.workers {
		cancel()
	}
	t.mu.Unlock()

	// Wait a bit for workers to finish
	time.Sleep(500 * time.Millisecond)

	// Shutdown engine
	t.engine.Shutdown()

	// Close pool
	t.pool.Close()

	// Print final stats
	stats := t.GetStats()
	log.Printf("[Floxy] Final stats: %+v", stats)

	return nil
}

func (t *FloxyStressTarget) worker(ctx context.Context, workerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			empty, err := t.engine.ExecuteNext(ctx, workerID)
			if err != nil {
				log.Printf("[Floxy] Worker %s error: %v", workerID, err)
				time.Sleep(100 * time.Millisecond)

				continue
			}
			if empty {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}

func (t *FloxyStressTarget) registerHandlers() {
	handlers := []floxy.StepHandler{
		&PaymentHandler{target: t},
		&InventoryHandler{target: t},
		&ShippingHandler{target: t},
		&NotificationHandler{target: t},
		&CompensationHandler{target: t},
		&ValidationHandler{target: t},
	}

	for _, h := range handlers {
		t.engine.RegisterHandler(h)
	}
}

func (t *FloxyStressTarget) registerWorkflows(ctx context.Context) error {
	workflows := []struct {
		name    string
		builder func() (*floxy.WorkflowDefinition, error)
	}{
		{"simple-order", t.buildSimpleOrderWorkflow},
		{"complex-order", t.buildComplexOrderWorkflow},
		{"parallel-processing", t.buildParallelProcessingWorkflow},
		{"nested-workflow", t.buildNestedWorkflow},
	}

	for _, wf := range workflows {
		workflow, err := wf.builder()
		if err != nil {
			return fmt.Errorf("failed to build %s: %w", wf.name, err)
		}
		if err := t.engine.RegisterWorkflow(ctx, workflow); err != nil {
			return fmt.Errorf("failed to register %s: %w", wf.name, err)
		}
	}

	return nil
}

func (t *FloxyStressTarget) buildSimpleOrderWorkflow() (*floxy.WorkflowDefinition, error) {
	return floxy.NewBuilder("simple-order", 1).
		Step("validate", "validation", floxy.WithStepMaxRetries(2)).
		Then("process-payment", "payment", floxy.WithStepMaxRetries(3)).
		OnFailure("refund-payment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "refund"})).
		Then("reserve-inventory", "inventory", floxy.WithStepMaxRetries(2)).
		OnFailure("release-inventory", "compensation", floxy.WithStepMetadata(map[string]any{"action": "release"})).
		Then("ship", "shipping").
		OnFailure("cancel-shipment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "cancel"})).
		Then("notify", "notification").
		Build()
}

func (t *FloxyStressTarget) buildComplexOrderWorkflow() (*floxy.WorkflowDefinition, error) {
	return floxy.NewBuilder("complex-order", 1).
		Step("validate", "validation", floxy.WithStepMaxRetries(2)).
		SavePoint("validation-checkpoint").
		Then("process-payment", "payment", floxy.WithStepMaxRetries(3)).
		OnFailure("refund-payment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "refund"})).
		SavePoint("payment-checkpoint").
		Then("reserve-inventory", "inventory", floxy.WithStepMaxRetries(2)).
		OnFailure("release-inventory", "compensation", floxy.WithStepMetadata(map[string]any{"action": "release"})).
		Then("ship", "shipping").
		OnFailure("cancel-shipment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "cancel"})).
		Then("notify", "notification").
		Build()
}

func (t *FloxyStressTarget) buildParallelProcessingWorkflow() (*floxy.WorkflowDefinition, error) {
	return floxy.NewBuilder("parallel-processing", 1).
		Step("validate", "validation").
		Fork("parallel-tasks",
			func(b *floxy.Builder) {
				b.Step("process-payment", "payment", floxy.WithStepMaxRetries(2)).
					OnFailure("refund-payment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "refund"}))
			},
			func(b *floxy.Builder) {
				b.Step("reserve-inventory", "inventory", floxy.WithStepMaxRetries(2)).
					OnFailure("release-inventory", "compensation", floxy.WithStepMetadata(map[string]any{"action": "release"}))
			},
		).
		Join("join", floxy.JoinStrategyAll).
		Then("ship", "shipping").
		OnFailure("cancel-shipment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "cancel"})).
		Build()
}

func (t *FloxyStressTarget) buildNestedWorkflow() (*floxy.WorkflowDefinition, error) {
	return floxy.NewBuilder("nested-workflow", 1).
		Step("validate", "validation").
		Then("process-payment", "payment", floxy.WithStepMaxRetries(3)).
		OnFailure("refund-payment", "compensation", floxy.WithStepMetadata(map[string]any{"action": "refund"})).
		SavePoint("payment-checkpoint").
		Fork("nested-parallel",
			func(b *floxy.Builder) {
				b.Step("reserve-inventory-1", "inventory").
					OnFailure("release-inventory-1", "compensation", floxy.WithStepMetadata(map[string]any{"action": "release"})).
					Then("ship-1", "shipping").
					OnFailure("cancel-shipment-1", "compensation", floxy.WithStepMetadata(map[string]any{"action": "cancel"}))
			},
			func(b *floxy.Builder) {
				b.Step("reserve-inventory-2", "inventory").
					OnFailure("release-inventory-2", "compensation", floxy.WithStepMetadata(map[string]any{"action": "release"})).
					Then("ship-2", "shipping").
					OnFailure("cancel-shipment-2", "compensation", floxy.WithStepMetadata(map[string]any{"action": "cancel"}))
			},
		).
		Join("join", floxy.JoinStrategyAll).
		Then("notify", "notification").
		Build()
}

func (t *FloxyStressTarget) ExecuteRandomWorkflow(ctx context.Context) error {
	workflows := []string{
		"simple-order-v1",
		"complex-order-v1",
		"parallel-processing-v1",
		"nested-workflow-v1",
	}

	workflowID := workflows[rand.Intn(len(workflows))]

	// Create random order data
	order := map[string]any{
		"order_id": fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		"user_id":  fmt.Sprintf("user-%d", rand.Intn(1000)),
		"amount":   float64(rand.Intn(500) + 50),
		"items":    rand.Intn(10) + 1,
		// Random failure injection
		"should_fail": rand.Float64() < 0.15, // 15% chance of failure
	}

	input, _ := json.Marshal(order)

	instanceID, err := t.engine.Start(ctx, workflowID, input)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	t.workflowInstances.Add(1)

	// Track rollback depth for this instance
	chaoskit.RecordRecursionDepth(ctx, 0)

	// Wait for completion with timeout
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		status, err := t.engine.GetStatus(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		// Check rollback depth
		depth := t.rollbackPlugin.GetMaxDepth(instanceID)
		if depth > 0 {
			chaoskit.RecordRecursionDepth(ctx, depth)
			currentMax := t.maxRollbackDepth.Load()
			if int32(depth) > currentMax {
				t.maxRollbackDepth.Store(int32(depth))
			}
		}

		if status == floxy.StatusCompleted {
			t.successfulRuns.Add(1)
			t.rollbackPlugin.ResetMaxDepth(instanceID)

			return nil
		}

		if status == floxy.StatusFailed {
			t.failedRuns.Add(1)
			if depth > 0 {
				t.rollbackCount.Add(1)
			}
			t.rollbackPlugin.ResetMaxDepth(instanceID)

			return fmt.Errorf("workflow failed")
		}

		if status == floxy.StatusAborted || status == floxy.StatusCancelled {
			t.failedRuns.Add(1)
			t.rollbackPlugin.ResetMaxDepth(instanceID)

			return fmt.Errorf("workflow %s", status)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("workflow timeout")
}

func (t *FloxyStressTarget) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_workflows":    t.workflowInstances.Load(),
		"successful_runs":    t.successfulRuns.Load(),
		"failed_runs":        t.failedRuns.Load(),
		"rollback_count":     t.rollbackCount.Load(),
		"max_rollback_depth": t.maxRollbackDepth.Load(),
		"metrics":            t.metricsCollector.GetStats(),
	}
}

// Step handlers implementation
// These handlers are targets for monkey patching and gofail injection

// PaymentHandler processes payment
// Can be patched with monkey patching injectors
// To enable gofail, uncomment failpoint calls and build with: go build -tags failpoint -gcflags=all=-l
var processPaymentHandler = func(ctx context.Context, order map[string]any) error {
	// Gofail failpoint (requires build with -tags failpoint)
	// Uncomment to enable:
	// import "github.com/pingcap/failpoint"
	// failpoint.Inject("payment-handler-panic", func() { panic("gofail: payment handler panic") })

	// Simulate processing time
	time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(20)))

	// Check for forced failure
	if shouldFail, ok := order["should_fail"].(bool); ok && shouldFail && rand.Float64() < 0.3 {
		return fmt.Errorf("payment processing failed")
	}

	order["payment_status"] = "paid"

	return nil
}

type PaymentHandler struct{ target *FloxyStressTarget }

func (h *PaymentHandler) Name() string { return "payment" }
func (h *PaymentHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	var order map[string]any
	if err := json.Unmarshal(input, &order); err != nil {
		return nil, err
	}

	// Call a processable function (can be monkey patched)
	if err := processPaymentHandler(ctx, order); err != nil {
		return nil, err
	}

	return json.Marshal(order)
}

// InventoryHandler processes inventory reservation
// To enable gofail, uncomment failpoint calls and build with: go build -tags failpoint -gcflags=all=-l
var processInventoryHandler = func(ctx context.Context, order map[string]any) error {
	// Gofail failpoint (requires build with -tags failpoint)
	// Uncomment to enable:
	// import "github.com/pingcap/failpoint"
	// failpoint.Inject("inventory-handler-panic", func() { panic("gofail: inventory handler panic") })

	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(15)))

	if shouldFail, ok := order["should_fail"].(bool); ok && shouldFail && rand.Float64() < 0.2 {
		return fmt.Errorf("inventory reservation failed")
	}

	order["inventory_status"] = "reserved"

	return nil
}

type InventoryHandler struct{ target *FloxyStressTarget }

func (h *InventoryHandler) Name() string { return "inventory" }
func (h *InventoryHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	var order map[string]any
	if err := json.Unmarshal(input, &order); err != nil {
		return nil, err
	}

	if err := processInventoryHandler(ctx, order); err != nil {
		return nil, err
	}

	return json.Marshal(order)
}

// ShippingHandler processes shipping
// To enable gofail, uncomment failpoint calls and build with: go build -tags failpoint -gcflags=all=-l
var processShippingHandler = func(ctx context.Context, order map[string]any) error {
	// Gofail failpoint (requires build with -tags failpoint)
	// Uncomment to enable:
	// import "github.com/pingcap/failpoint"
	// failpoint.Inject("shipping-handler-panic", func() { panic("gofail: shipping handler panic") })

	time.Sleep(time.Millisecond * time.Duration(10+rand.Intn(20)))

	if shouldFail, ok := order["should_fail"].(bool); ok && shouldFail && rand.Float64() < 0.1 {
		return fmt.Errorf("shipping failed")
	}

	order["shipping_status"] = "shipped"

	return nil
}

type ShippingHandler struct{ target *FloxyStressTarget }

func (h *ShippingHandler) Name() string { return "shipping" }
func (h *ShippingHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	var order map[string]any
	if err := json.Unmarshal(input, &order); err != nil {
		return nil, err
	}

	if err := processShippingHandler(ctx, order); err != nil {
		return nil, err
	}

	return json.Marshal(order)
}

type NotificationHandler struct{ target *FloxyStressTarget }

func (h *NotificationHandler) Name() string { return "notification" }
func (h *NotificationHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(10)))

	return input, nil
}

type ValidationHandler struct{ target *FloxyStressTarget }

func (h *ValidationHandler) Name() string { return "validation" }
func (h *ValidationHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(10)))

	return input, nil
}

type CompensationHandler struct{ target *FloxyStressTarget }

func (h *CompensationHandler) Name() string { return "compensation" }
func (h *CompensationHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	action, _ := stepCtx.GetVariableAsString("action")
	time.Sleep(time.Millisecond * time.Duration(5+rand.Intn(10)))

	return json.Marshal(map[string]any{"compensated": action})
}

// Floxy Metrics Collector
type FloxyMetricsCollector struct {
	workflowsStarted      atomic.Int64
	workflowsCompleted    atomic.Int64
	workflowsFailed       atomic.Int64
	stepsStarted          atomic.Int64
	stepsCompleted        atomic.Int64
	stepsFailed           atomic.Int64
	totalStepDuration     atomic.Int64
	totalWorkflowDuration atomic.Int64
}

func NewFloxyMetricsCollector() *FloxyMetricsCollector {
	return &FloxyMetricsCollector{}
}

func (c *FloxyMetricsCollector) RecordWorkflowStarted(instanceID int64, workflowID string) {
	c.workflowsStarted.Add(1)
}

func (c *FloxyMetricsCollector) RecordWorkflowCompleted(
	instanceID int64,
	workflowID string,
	duration time.Duration,
	status floxy.WorkflowStatus,
) {
	c.workflowsCompleted.Add(1)
	c.totalWorkflowDuration.Add(duration.Milliseconds())
}

func (c *FloxyMetricsCollector) RecordWorkflowFailed(instanceID int64, workflowID string, duration time.Duration) {
	c.workflowsFailed.Add(1)
	c.totalWorkflowDuration.Add(duration.Milliseconds())
}

func (c *FloxyMetricsCollector) RecordWorkflowStatus(instanceID int64, workflowID string, status floxy.WorkflowStatus) {
}

func (c *FloxyMetricsCollector) RecordStepStarted(
	instanceID int64,
	workflowID,
	stepName string,
	stepType floxy.StepType,
) {
	c.stepsStarted.Add(1)
}

func (c *FloxyMetricsCollector) RecordStepCompleted(
	instanceID int64,
	workflowID, stepName string,
	stepType floxy.StepType,
	duration time.Duration,
) {
	c.stepsCompleted.Add(1)
	c.totalStepDuration.Add(duration.Milliseconds())
}

func (c *FloxyMetricsCollector) RecordStepFailed(
	instanceID int64,
	workflowID, stepName string,
	stepType floxy.StepType,
	duration time.Duration,
) {
	c.stepsFailed.Add(1)
	c.totalStepDuration.Add(duration.Milliseconds())
}

func (c *FloxyMetricsCollector) RecordStepStatus(
	instanceID int64,
	workflowID, stepName string,
	status floxy.StepStatus,
) {
}

func (c *FloxyMetricsCollector) GetStats() map[string]interface{} {
	avgStepDuration := int64(0)
	if c.stepsCompleted.Load() > 0 {
		avgStepDuration = c.totalStepDuration.Load() / (c.stepsCompleted.Load() + c.stepsFailed.Load())
	}

	avgWorkflowDuration := int64(0)
	if c.workflowsCompleted.Load() > 0 {
		avgWorkflowDuration = c.totalWorkflowDuration.Load() / (c.workflowsCompleted.Load() + c.workflowsFailed.Load())
	}

	return map[string]interface{}{
		"workflows_started":        c.workflowsStarted.Load(),
		"workflows_completed":      c.workflowsCompleted.Load(),
		"workflows_failed":         c.workflowsFailed.Load(),
		"steps_started":            c.stepsStarted.Load(),
		"steps_completed":          c.stepsCompleted.Load(),
		"steps_failed":             c.stepsFailed.Load(),
		"avg_step_duration_ms":     avgStepDuration,
		"avg_workflow_duration_ms": avgWorkflowDuration,
	}
}

// ChaosKit integration
func RunFloxyWorkflow(ctx context.Context, target chaoskit.Target) error {
	floxyTarget, ok := target.(*FloxyStressTarget)
	if !ok {
		return fmt.Errorf("target is not FloxyStressTarget")
	}

	return floxyTarget.ExecuteRandomWorkflow(ctx)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== Floxy Stress Test with ChaosKit ===")
	log.Println("Testing Floxy workflow engine with advanced chaos injection")
	log.Println()
	log.Println("Chaos injection methods:")
	log.Println("  1. Monkey patching - runtime function patching")
	log.Println("  2. Gofail - failpoint-based injection")
	log.Println("  3. ToxiProxy - network chaos for database")
	log.Println()

	// Database connection configuration
	// Can use direct connection or via ToxiProxy
	useToxiProxy := os.Getenv("USE_TOXIPROXY") == "true"
	toxiproxyHost := os.Getenv("TOXIPROXY_HOST")
	if toxiproxyHost == "" {
		toxiproxyHost = "http://localhost:8474"
	}

	// Default connection string
	directConnString := "postgres://floxy:password@localhost:5435/floxy?sslmode=disable"
	connString := directConnString

	// Setup ToxiProxy for database connection chaos
	var toxiproxyClient *injectors.ToxiProxyClient
	var toxiproxyManager *injectors.ToxiProxyManager
	if useToxiProxy {
		log.Println("[Setup] Initializing ToxiProxy for database connection chaos...")
		toxiproxyClient = injectors.NewToxiProxyClient(toxiproxyHost)
		toxiproxyManager = injectors.NewToxiProxyManager(toxiproxyClient)

		// Create proxy for PostgreSQL
		proxyConfig := injectors.ProxyConfig{
			Name:     "postgres-proxy",
			Listen:   "localhost:6432", // Proxy listens here
			Upstream: "localhost:5435", // Actual PostgreSQL
			Enabled:  true,
		}

		if err := toxiproxyManager.CreateProxy(proxyConfig); err != nil {
			log.Printf("[Setup] Note: Proxy might already exist: %v", err)
		} else {
			// Use proxy connection string
			connString = "postgres://floxy:password@localhost:6432/floxy?sslmode=disable"
			log.Println("[Setup] Using ToxiProxy for database connections")
		}
	}

	// Create Floxy target
	log.Println("[Setup] Creating Floxy target...")
	floxyTarget, err := NewFloxyStressTarget(connString)
	if err != nil {
		log.Fatalf("Failed to create Floxy target: %v", err)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Statistics reporter
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := floxyTarget.GetStats()
				log.Printf("[Stats] %+v", stats)
			}
		}
	}()

	// Build injectors
	log.Println("[Setup] Creating chaos injectors...")

	// 1. Monkey patching injectors for handlers
	// Note: Must pass pointer to function for monkey patching
	monkeyPanicTargets := []injectors.PatchTarget{
		{
			Func:        &processPaymentHandler,
			Probability: 0.05, // 5% chance of panic
			FuncName:    "processPaymentHandler",
		},
		{
			Func:        &processInventoryHandler,
			Probability: 0.05,
			FuncName:    "processInventoryHandler",
		},
		{
			Func:        &processShippingHandler,
			Probability: 0.05,
			FuncName:    "processShippingHandler",
		},
	}

	monkeyPanicInjector := injectors.MonkeyPatchPanic(monkeyPanicTargets)

	monkeyDelayTargets := []injectors.DelayPatchTarget{
		{
			Func:        &processPaymentHandler,
			MinDelay:    50 * time.Millisecond,
			MaxDelay:    200 * time.Millisecond,
			Probability: 0.2, // 20% chance of delay
			DelayBefore: true,
			FuncName:    "processPaymentHandler",
		},
		{
			Func:        &processInventoryHandler,
			MinDelay:    30 * time.Millisecond,
			MaxDelay:    150 * time.Millisecond,
			Probability: 0.15,
			DelayBefore: true,
			FuncName:    "processInventoryHandler",
		},
	}

	monkeyDelayInjector := injectors.MonkeyPatchDelay(monkeyDelayTargets)

	// 2. Gofail injector (requires build with -tags failpoint)
	failpointNames := []string{
		"payment-handler-panic",
		"inventory-handler-panic",
		"shipping-handler-panic",
	}
	gofailInjector := injectors.GofailPanic(failpointNames, 0.03, 500*time.Millisecond)

	// 3. ToxiProxy injectors for database network chaos
	var latencyInjector *injectors.ToxiProxyLatencyInjector
	var bandwidthInjector *injectors.ToxiProxyBandwidthInjector
	var timeoutInjector *injectors.ToxiProxyTimeoutInjector

	if useToxiProxy && toxiproxyClient != nil {
		// Latency for database calls
		latencyInjector = injectors.ToxiProxyLatency(
			toxiproxyClient,
			"postgres-proxy",
			100*time.Millisecond, // 100ms latency
			20*time.Millisecond,  // ±20ms jitter
		)

		// Bandwidth limiting (simulates slow network)
		bandwidthInjector = injectors.ToxiProxyBandwidth(
			toxiproxyClient,
			"postgres-proxy",
			500, // 500 KB/s limit
		)

		// Connection timeouts
		timeoutInjector = injectors.ToxiProxyTimeout(
			toxiproxyClient,
			"postgres-proxy",
			2*time.Second, // 2 second timeout
		)
	}

	// Create ChaosKit scenario
	log.Println("[Setup] Building chaos scenario...")
	scenarioBuilder := chaoskit.NewScenario("floxy-stress-test").
		WithTarget(floxyTarget).
		Step("execute-workflow", RunFloxyWorkflow).
		// Monkey patching injectors (always active)
		Inject("monkey-panic", monkeyPanicInjector).
		Inject("monkey-delay", monkeyDelayInjector)

	// Gofail injector (may fail if not built with -tags failpoint)
	if err := gofailInjector.Inject(ctx); err != nil {
		if err == injectors.ErrFailpointDisabled {
			log.Println("[Warning] Gofail injector disabled (build with -tags failpoint to enable)")
		} else {
			log.Printf("[Warning] Gofail injector error: %v", err)
		}
	} else {
		log.Println("[Setup] Gofail injector enabled")
		scenarioBuilder = scenarioBuilder.Inject("gofail-panic", gofailInjector)
	}

	// ToxiProxy injectors (conditional)
	if useToxiProxy && latencyInjector != nil {
		scenarioBuilder = scenarioBuilder.
			Inject("db-latency", latencyInjector).
			Inject("db-bandwidth", bandwidthInjector).
			Inject("db-timeout", timeoutInjector)
		log.Println("[Setup] ToxiProxy injectors enabled")
	}

	scenario := scenarioBuilder.
		// Validators
		Assert("recursion-depth", validators.RecursionDepthLimit(50)).
		Assert("goroutine-leak", validators.GoroutineLimit(500)).
		Assert("no-infinite-loop", validators.NoInfiniteLoop(15*time.Second)).
		RunFor(60 * time.Second). // Run for 1 minute
		Build()

	// Create executor with ContinueOnFailure policy
	executor := chaoskit.NewExecutor(
		chaoskit.WithFailurePolicy(chaoskit.ContinueOnFailure),
	)

	// Run in background
	log.Println("[Main] Starting chaos scenario...")
	go func() {
		if err := executor.Run(ctx, scenario); err != nil {
			log.Printf("Scenario error: %v", err)
		}
	}()

	// Wait for signal
	<-sigCh
	log.Println("\n[Main] Shutting down...")
	cancel()

	// Stop injectors
	log.Println("[Cleanup] Stopping injectors...")
	if useToxiProxy && toxiproxyManager != nil {
		if err := toxiproxyManager.DeleteProxy("postgres-proxy"); err != nil {
			log.Printf("[Cleanup] Error removing proxy: %v", err)
		}
	}

	// Wait a bit for cleanup
	time.Sleep(2 * time.Second)

	// Print final report
	log.Println("\n=== Final Report ===")
	log.Println(executor.Reporter().GenerateReport())

	// Print Floxy stats
	stats := floxyTarget.GetStats()
	log.Printf("\n=== Floxy Statistics ===")
	log.Printf("Total workflows: %d", stats["total_workflows"])
	log.Printf("Successful runs: %d", stats["successful_runs"])
	log.Printf("Failed runs: %d", stats["failed_runs"])
	log.Printf("Rollback count: %d", stats["rollback_count"])
	log.Printf("Max rollback depth: %d", stats["max_rollback_depth"])
	log.Printf("Metrics: %+v", stats["metrics"])
}
