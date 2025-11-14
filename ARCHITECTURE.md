# ChaosKit Architecture

## Overview

ChaosKit is a modular framework for chaos engineering that follows clean architecture principles. The framework enables systematic testing of system reliability through controlled fault injection and invariant validation.

## Core Principles

- **Separation of Concerns**: Clear boundaries between components
- **Interface-Based Design**: Extensibility through well-defined interfaces
- **Context Propagation**: Chaos capabilities flow through context
- **Thread Safety**: All components are designed for concurrent execution
- **Structured Logging**: Comprehensive observability with slog

## High-Level Architecture

```mermaid
graph TB
    User[User Code] --> Scenario[Scenario Builder]
    Scenario --> Executor[Executor]
    
    Executor --> Target[Target System]
    Executor --> Injectors[Injectors]
    Executor --> Validators[Validators]
    Executor --> Metrics[MetricsCollector]
    Executor --> Reporter[Reporter]
    
    Target --> Step1[Step 1]
    Target --> Step2[Step 2]
    Target --> StepN[Step N]
    
    Injectors --> Delay[DelayInjector]
    Injectors --> Panic[PanicInjector]
    Injectors --> Network[NetworkInjector]
    Injectors --> MonkeyPatch[MonkeyPatchInjector]
    
    Validators --> Goroutine[GoroutineLeakValidator]
    Validators --> Recursion[RecursionDepthValidator]
    Validators --> PanicRecovery[PanicRecoveryValidator]
    Validators --> Memory[MemoryLimitValidator]
    
    Metrics --> Stats[Execution Statistics]
    Reporter --> Report[Execution Report]
    
    style Executor fill:#e1f5ff
    style Injectors fill:#fff4e1
    style Validators fill:#e8f5e9
    style Metrics fill:#f3e5f5
    style Reporter fill:#fce4ec
```

## Component Interaction

```mermaid
graph LR
    subgraph "Scenario Definition"
        SB[ScenarioBuilder] --> S[Scenario]
        S --> T[Target]
        S --> ST[Steps]
        S --> I[Injectors]
        S --> V[Validators]
    end
    
    subgraph "Execution"
        E[Executor] --> TC[Target Setup]
        E --> IS[Injector Setup]
        E --> EX[Execute Steps]
        E --> VA[Validate]
        E --> CL[Cleanup]
    end
    
    subgraph "Context Flow"
        CTX[Context] --> CC[ChaosContext]
        CTX --> ER[EventRecorder]
        CTX --> RNG[Random Generator]
        CC --> UD[User Code]
    end
    
    S --> E
    EX --> CTX
    UD --> ER
    ER --> V
    
    style E fill:#e1f5ff
    style CTX fill:#fff9c4
    style UD fill:#c8e6c9
```

## Execution Flow

```mermaid
sequenceDiagram
    participant User
    participant Executor
    participant Target
    participant Injectors
    participant Steps
    participant Validators
    participant Metrics
    participant Reporter
    
    User->>Executor: Run(scenario)
    Executor->>Target: Setup()
    Target-->>Executor: OK
    
    Executor->>Injectors: Inject(ctx)
    Injectors-->>Executor: OK
    
    loop For each iteration
        Executor->>Validators: Reset()
        Validators-->>Executor: OK
        
        loop For each step
            Executor->>Injectors: BeforeStep(ctx)
            Injectors-->>Executor: OK
            
            Executor->>Steps: Execute(ctx, target)
            Steps->>Steps: MaybeDelay(ctx)
            Steps->>Steps: MaybePanic(ctx)
            Steps->>Steps: RecordRecursionDepth(ctx, depth)
            Steps-->>Executor: OK/Error
            
            Executor->>Injectors: AfterStep(ctx, err)
            Injectors-->>Executor: OK
        end
        
        Executor->>Validators: Validate(ctx, target)
        Validators-->>Executor: OK/Error
        
        Executor->>Metrics: RecordExecution(result)
        Executor->>Reporter: AddResult(result)
    end
    
    Executor->>Injectors: Stop(ctx)
    Injectors-->>Executor: OK
    
    Executor->>Target: Teardown()
    Target-->>Executor: OK
    
    Executor->>Reporter: GenerateReport()
    Reporter-->>User: Report
```

## Injector Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: NewInjector()
    Created --> Injecting: Inject(ctx)
    Injecting --> Active: Background goroutines started
    Active --> Injecting: Apply chaos
    Active --> Stopping: Stop(ctx)
    Stopping --> Stopped: Cleanup complete
    Stopped --> [*]
    
    note right of Active
        Injectors can be:
        - Global (CPU, Memory)
        - Context-based (Delay, Panic)
        - Step-based (BeforeStep/AfterStep, e.g. DelayInjector)
        - Hybrid (Network)
    end note
```

## Context-Based Chaos Flow

```mermaid
graph TD
    Start[User Code] --> Check{ChaosContext<br/>in Context?}
    Check -->|No| NoOp[No-op, continue]
    Check -->|Yes| GetCC[GetChaosContext]
    
    GetCC --> Delay{MaybeDelay?}
    Delay -->|Yes| DelayFunc[Call delayFunc]
    DelayFunc --> Sleep[time.Sleep]
    Sleep --> Continue[Continue]
    Delay -->|No| Continue
    
    GetCC --> Panic{MaybePanic?}
    Panic -->|Yes| PanicFunc[Call panicFunc]
    PanicFunc --> Trigger{Should Panic?}
    Trigger -->|Yes| Panic[panic]
    Trigger -->|No| Continue
    Panic -->|No| Continue
    
    GetCC --> Network{MaybeNetworkChaos?}
    Network -->|Yes| NetworkFunc[Call networkFunc]
    NetworkFunc --> Latency[Apply Latency]
    NetworkFunc --> Drop[Drop Connection]
    Latency --> Continue
    Drop --> Continue
    
    Continue --> End[End]
    NoOp --> End
```

## Validator Execution Flow

```mermaid
flowchart TD
    Start[Start Validation] --> Reset{Resettable?}
    Reset -->|Yes| ResetState[Reset State]
    Reset -->|No| Validate
    ResetState --> Validate[Validate]
    
    Validate --> Check{Check Invariants}
    Check -->|Pass| LogPass[Log Debug: Pass]
    Check -->|Warn| LogWarn[Log Warn: Approaching Limit]
    Check -->|Fail| LogError[Log Error: Failed]
    
    LogPass --> ReturnOK[Return nil]
    LogWarn --> Continue[Continue]
    LogError --> ReturnError[Return error]
    
    Continue --> Next{More Validators?}
    Next -->|Yes| Validate
    Next -->|No| ReturnOK
    
    ReturnOK --> End[End]
    ReturnError --> End
```

## Injector Types

```mermaid
graph TB
    Injector[Injector Interface] --> Global[GlobalInjector]
    Injector --> Context[ContextInjector]
    Injector --> Step[StepInjector]
    Injector --> Hybrid[HybridInjector]
    
    Global --> CPU[CPUInjector]
    Global --> Memory[MemoryInjector]
    Global --> NetworkProxy[Network Proxy Injectors]
    
    Context --> Delay[DelayInjector]
    Context --> Panic[PanicInjector]
    Context --> Cancellation[ContextCancellationInjector]
    
    Step --> StepDelay[DelayInjector<br/>BeforeStep/AfterStep]
    Step --> ContextPanic[PanicInjector<br/>via MaybePanic()]
    
    Hybrid --> NetworkContextual[ContextualNetworkInjector]
    
    Injector --> Provider[ChaosProvider]
    Provider --> DelayProvider[ChaosDelayProvider]
    Provider --> PanicProvider[ChaosPanicProvider]
    Provider --> NetworkProvider[ChaosNetworkProvider]
    
    style Injector fill:#e1f5ff
    style Global fill:#fff4e1
    style Context fill:#e8f5e9
    style Step fill:#f3e5f5
    style Hybrid fill:#fce4ec
```

## Validator Types

```mermaid
graph TB
    Validator[Validator Interface] --> PanicRecovery[PanicRecoveryValidator]
    Validator --> Recursion[RecursionDepthValidator]
    Validator --> Goroutine[GoroutineLeakValidator]
    Validator --> InfiniteLoop[InfiniteLoopValidator]
    Validator --> ExecutionTime[ExecutionTimeValidator]
    Validator --> Memory[MemoryLimitValidator]
    Validator --> StateConsistency[StateConsistencyValidator]
    Validator --> Composite[CompositeValidator]
    
    PanicRecovery --> PanicRecorder[PanicRecorder Interface]
    Recursion --> RecursionRecorder[RecursionRecorder Interface]
    
    Composite --> V1[Validator 1]
    Composite --> V2[Validator 2]
    Composite --> VN[Validator N]
    
    style Validator fill:#e8f5e9
    style PanicRecovery fill:#c8e6c9
    style Recursion fill:#c8e6c9
    style Composite fill:#a5d6a7
```

## Extension Points

### Creating Custom Injectors

```mermaid
graph LR
    Custom[Custom Injector] --> Interface[Implement Injector]
    Interface --> Inject[Inject method]
    Interface --> Stop[Stop method]
    Interface --> Name[Name method]
    
    Custom --> Optional[Optional Interfaces]
    Optional --> Provider[ChaosProvider]
    Optional --> StepInjector[StepInjector]
    Optional --> MetricsProvider[MetricsProvider]
    
    style Custom fill:#e1f5ff
    style Interface fill:#fff4e1
    style Optional fill:#e8f5e9
```

### Creating Custom Validators

```mermaid
graph LR
    Custom[Custom Validator] --> Interface[Implement Validator]
    Interface --> Validate[Validate method]
    Interface --> Name[Name method]
    
    Custom --> Optional[Optional Interfaces]
    Optional --> Resettable[Resettable]
    Optional --> PanicRecorder[PanicRecorder]
    Optional --> RecursionRecorder[RecursionRecorder]
    
    style Custom fill:#e8f5e9
    style Interface fill:#c8e6c9
    style Optional fill:#a5d6a7
```

## Data Flow

```mermaid
graph TB
    Scenario[Scenario] --> Executor[Executor]
    Executor --> BuildContext[Build ChaosContext]
    BuildContext --> AttachContext[Attach to Context]
    
    AttachContext --> UserCode[User Code Execution]
    UserCode --> MaybeDelay[MaybeDelay]
    UserCode --> MaybePanic[MaybePanic]
    UserCode --> MaybeNetworkChaos[MaybeNetworkChaos]
    UserCode --> RecordEvents[Record Events]
    
    MaybeDelay --> DelayProvider[DelayProvider]
    MaybePanic --> PanicProvider[PanicProvider]
    MaybeNetworkChaos --> NetworkProvider[NetworkProvider]
    RecordEvents --> EventRecorder[EventRecorder]
    
    EventRecorder --> Validators[Validators]
    DelayProvider --> Injector[Injector]
    PanicProvider --> Injector
    NetworkProvider --> Injector
    
    Validators --> Result[Validation Result]
    Injector --> Metrics[Metrics]
    Result --> Metrics
    Metrics --> Reporter[Reporter]
    
    style UserCode fill:#c8e6c9
    style EventRecorder fill:#fff9c4
    style Validators fill:#e8f5e9
    style Metrics fill:#f3e5f5
```

## Thread Safety Model

```mermaid
graph TB
    Concurrent[Concurrent Access] --> Mutex[Mutex Protection]
    Concurrent --> Atomic[Atomic Operations]
    Concurrent --> Immutable[Immutable Data]
    
    Mutex --> Injectors[Injector State]
    Mutex --> Validators[Validator State]
    Mutex --> Metrics[Metrics Collection]
    Mutex --> Reporter[Reporter Results]
    
    Atomic --> Counters[Counters<br/>cancelCount, delayCount]
    
    Immutable --> Context[Context Values]
    Immutable --> Config[Configuration]
    
    style Mutex fill:#ffcdd2
    style Atomic fill:#c8e6c9
    style Immutable fill:#e1f5ff
```

## Design Decisions

### 1. Context-Based Chaos Injection

**Decision**: Use context to propagate chaos capabilities to user code.

**Rationale**:
- Non-intrusive: No need to pass injectors explicitly
- Thread-safe: Context is immutable and goroutine-safe
- Flexible: Can be attached/detached dynamically
- Standard: Follows Go idioms

**Trade-offs**:
- Requires code instrumentation (calling MaybeDelay, etc.)
- Context overhead (minimal)

### 2. Structured Logging with slog

**Decision**: Use Go's standard `log/slog` package for all logging.

**Rationale**:
- Standard library: No external dependencies
- Structured: Easy to parse and filter
- Performance: Optimized for production use
- Levels: Built-in support for Debug/Info/Warn/Error

**Trade-offs**:
- Requires Go 1.25+ (acceptable for modern projects)

### 3. Interface-Based Design

**Decision**: Use interfaces extensively for extensibility.

**Rationale**:
- Testability: Easy to mock and test
- Flexibility: Multiple implementations
- Extensibility: Users can create custom components
- Clean: Clear contracts between components

**Trade-offs**:
- More interfaces to maintain
- Potential over-engineering (mitigated by keeping interfaces focused)

### 4. Builder Pattern for Scenarios

**Decision**: Use fluent builder pattern for scenario construction.

**Rationale**:
- Readability: Clear, chainable API
- Type safety: Compile-time validation
- Flexibility: Optional parameters without function overloading
- Discoverability: IDE autocomplete guides usage

**Trade-offs**:
- More code to maintain
- Slightly more complex than struct initialization

### 5. Separation of Injectors and Validators

**Decision**: Keep injectors and validators as separate components.

**Rationale**:
- Single Responsibility: Each component has one job
- Composability: Mix and match independently
- Testability: Test injection and validation separately
- Clarity: Clear separation of concerns

**Trade-offs**:
- More components to manage
- Potential duplication (mitigated by shared interfaces)

## Performance Considerations

1. **Lock Granularity**: Fine-grained locks to minimize contention
2. **Atomic Operations**: Used for counters to avoid mutex overhead
3. **Context Immutability**: Context values are immutable, safe for concurrent access
4. **Copy-on-Read**: Maps and slices are copied when needed for thread safety
5. **Lazy Initialization**: Validators initialize baselines on first use

## Security Considerations

1. **Monkey Patching**: Only for testing, requires explicit build flags
2. **Network Proxies**: Requires explicit setup, not enabled by default
3. **Context Isolation**: Each scenario has isolated context
4. **Resource Limits**: Validators enforce resource limits

---

**Last Updated**: November 2025  
**Version**: 1.0

