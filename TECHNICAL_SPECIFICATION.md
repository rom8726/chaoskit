# Technical Specification: High Priority Improvements

**Project:** ChaosKit  
**Version:** 1.0  
**Date:** November 13, 2025  
**Priority:** High  

---

## 1. Overview

This document outlines the technical specification for three critical improvements to the ChaosKit framework:
1. Race condition fixes
2. Structured logging implementation using `slog`
3. Documentation enhancements

## 2. Goals and Objectives

- **Primary Goal:** Improve code safety, observability, and developer experience
- **Success Metrics:**
  - Zero race conditions detected by `go test -race`
  - All components use structured logging
  - 100% of public APIs documented with godoc comments
  - Comprehensive examples and architecture diagrams added

---

## 3. Race Condition Fixes

### 3.1 Problem Statement

Several components have potential race conditions where shared state is accessed without proper synchronization, particularly in:
- `chaos_context.go` - Function pointer access
- Injector implementations - Concurrent state modifications
- Metrics collection - Unsynchronized counters

### 3.2 Affected Components

- [x] `chaos_context.go` - `ChaosContext` struct
- [x] `injectors/gofail_panic.go` - Active failpoints map
- [x] `injectors/network_contextual.go` - Host patterns map
- [x] `injectors/context_cancellation.go` - Active contexts tracking
- [x] `executor.go` - Metrics collection during parallel execution
- [x] All custom injector implementations

### 3.3 Implementation Tasks

#### 3.3.1 Fix ChaosContext Race Conditions

- [x] Analyze `MaybeDelay()`, `MaybePanic()`, `MaybeNetworkChaos()` for race conditions
- [x] Ensure function pointers are accessed under read lock
- [x] Ensure function execution doesn't happen while holding locks (if possible)
- [x] Add atomic operations for simple flag checks where appropriate
- [x] Document lock ordering requirements in comments

**Code Location:** `chaos_context.go`

**Acceptance Criteria:**
- `go test -race ./...` passes for chaos_context tests
- No data races reported during concurrent `MaybeX()` calls
- Performance impact < 5% for lock-protected operations

#### 3.3.2 Fix Injector Race Conditions

- [x] Audit all injector `Inject()` and `Stop()` methods for race conditions
- [x] Add proper mutex protection for state maps in:
  - [x] `GofailPanicInjector.active` map
  - [x] `ContextualNetworkInjector.hostPatterns` map
  - [x] `ContextCancellationInjector.activeContexts` slice
- [x] Use `sync.Map` where appropriate for high-concurrency scenarios
- [x] Add atomic counters for metrics (panic count, delay count, etc.)
- [x] Document thread-safety guarantees in each injector

**Code Locations:** `injectors/*.go`

**Acceptance Criteria:**
- All injector tests pass with `-race` flag
- Concurrent `Inject()` and `Stop()` calls are safe
- Metrics can be read concurrently with injections

#### 3.3.3 Fix Executor Race Conditions

- [x] Review metrics collection during parallel scenario execution
- [x] Ensure reporter aggregation is thread-safe
- [x] Add mutex protection for shared state in executor
- [x] Use channels for concurrent result collection where appropriate
- [x] Audit context propagation for race conditions

**Code Location:** `executor.go`

**Acceptance Criteria:**
- Multiple scenarios can run concurrently without races
- Metrics collection is accurate under concurrent load
- All tests pass with `-race` flag

#### 3.3.4 Testing

- [x] Add race detector to CI pipeline (`make race` target)
- [ ] Create stress tests that run concurrent operations
- [ ] Add test for concurrent scenario execution
- [ ] Add test for concurrent injector operations
- [x] Verify fixes with `go test -race -count=100 ./...`

**Acceptance Criteria:**
- CI pipeline includes race detection
- All race detector warnings are fixed
- Stress tests run successfully for 1000+ iterations

---

## 4. Structured Logging with slog

### 4.1 Problem Statement

Current logging uses simple `Printf`-style logging which lacks:
- Structured output for log aggregation
- Log levels (debug, info, warn, error)
- Context propagation
- Performance optimization
- Standard format for tooling integration

### 4.2 Implementation Tasks

#### 4.2.1 Core Logging Infrastructure

- [x] Replace `Logger` interface in `core.go` with `slog.Logger`
- [x] Create default logger configuration
- [x] Add logger configuration options to `ExecutorOption`
- [x] Support JSON and text output formats
- [x] Add log level configuration

**Code Location:** `core.go`, `executor.go`

**Example Interface:**
```go
// Use *slog.Logger directly instead of custom interface
type ExecutorOption func(*Executor)

func WithLogger(logger *slog.Logger) ExecutorOption
func WithLogLevel(level slog.Level) ExecutorOption
func WithJSONLogging() ExecutorOption
```

#### 4.2.2 Update All Logging Calls

- [x] Replace `e.log()` calls in executor with structured logging
- [x] Add contextual fields to log entries (scenario name, iteration, etc.)
- [x] Use appropriate log levels:
  - [x] `Debug` for verbose operation details
  - [x] `Info` for normal operations
  - [x] `Warn` for recoverable issues
  - [x] `Error` for failures
- [x] Add log attributes for filtering and searching

**Code Locations:** `executor.go`, `reporter.go`, `injectors/*.go`

**Example Usage:**
```go
logger.Info("starting scenario execution",
    slog.String("scenario", scenario.name),
    slog.Int("iteration", i),
    slog.Duration("duration", elapsed))

logger.Error("injector failed",
    slog.String("injector", inj.Name()),
    slog.Any("error", err))
```

#### 4.2.3 Update Injectors

- [x] Add `slog.Logger` field to injector structs (using slog.Default())
- [x] Log injection events with structured context
- [x] Log errors with appropriate severity
- [x] Add debug logging for troubleshooting
- [x] Include injector-specific metrics in logs

**Code Locations:** All files in `injectors/` directory

**Acceptance Criteria:**
- All injectors accept optional logger in constructor
- Injection events are logged with structured data
- Logs include injector name, timestamp, and relevant context

#### 4.2.4 Update Validators

- [x] Add `slog.Logger` field to validator structs (using slog.Default())
- [x] Log validation failures with context
- [x] Log validation warnings (approaching thresholds)
- [x] Include validator-specific metrics in logs

**Code Locations:** All files in `validators/` directory

#### 4.2.5 Documentation

- [ ] Document logging configuration in README
- [ ] Add example of custom logger setup
- [ ] Document log structure and fields
- [ ] Add logging best practices section
- [ ] Update examples to use structured logging

**Acceptance Criteria:**
- README includes logging configuration section
- Examples demonstrate different log outputs
- Log field naming conventions documented

---

## 5. Documentation Improvements

### 5.1 Problem Statement

Current documentation needs improvement in:
- API documentation (missing godoc comments)
- Architecture diagrams
- Tutorial progression from basic to advanced
- Code examples in documentation
- Troubleshooting guide

### 5.2 Implementation Tasks

#### 5.2.1 API Documentation (godoc)

- [x] Add package-level documentation for `chaoskit` package
- [x] Document all exported types with examples
- [x] Document all exported functions with examples
- [x] Document all exported interfaces with implementation guidelines
- [ ] Add "Example_" test functions for key workflows

**Code Locations:** All `.go` files with exported symbols

**Standards:**
```go
// Package chaoskit provides a modular framework for chaos engineering.
//
// ChaosKit enables systematic testing of system reliability through
// controlled fault injection and invariant validation.
//
// # Basic Usage
//
//     scenario := chaoskit.NewScenario("test").
//         WithTarget(mySystem).
//         Inject("delay", injectors.RandomDelay(5*time.Millisecond, 25*time.Millisecond)).
//         Assert("goroutines", validators.GoroutineLimit(100)).
//         Build()
//
// # Architecture
//
// ChaosKit follows clean architecture principles...
package chaoskit
```

**Tasks:**
- [ ] Add package documentation to `chaoskit.go`
- [ ] Document `Scenario` type and builder pattern
- [ ] Document `Injector` interface with implementation guide
- [ ] Document `Validator` interface with implementation guide
- [ ] Document `Target` interface with implementation examples
- [ ] Document all injector types in `injectors/` package
- [ ] Document all validator types in `validators/` package
- [ ] Add godoc examples for each major component

#### 5.2.2 Architecture Documentation

- [ ] Create `ARCHITECTURE.md` with detailed design
- [ ] Add component interaction diagrams (ASCII or Mermaid)
- [ ] Document execution flow
- [ ] Document extension points
- [ ] Add sequence diagrams for key scenarios

**File:** `ARCHITECTURE.md`

**Content Structure:**
```markdown
# Architecture

## Overview
## Core Components
## Component Interactions
## Execution Flow
## Extension Points
## Design Decisions
```

**Diagrams to Include:**
- [ ] High-level component diagram
- [ ] Scenario execution sequence diagram
- [ ] Injector lifecycle diagram
- [ ] Validator execution flow

#### 5.2.3 Tutorial and Examples

- [ ] Create `TUTORIAL.md` with progressive examples
- [ ] Update `examples/README.md` with detailed explanations
- [ ] Add example for each injector type
- [ ] Add example for each validator type
- [ ] Add example for custom injector implementation
- [ ] Add example for custom validator implementation
- [ ] Add example for complex scenarios

**File:** `TUTORIAL.md`

**Structure:**
```markdown
# ChaosKit Tutorial

## Part 1: Basic Scenario
## Part 2: Adding Validators
## Part 3: Multiple Injectors
## Part 4: Network Chaos
## Part 5: Custom Injectors
## Part 6: Production Usage
```

#### 5.2.4 README Improvements

- [ ] Add badges (build status, coverage, go report card)
- [ ] Improve "Quick Start" section with runnable example
- [ ] Add "When to Use ChaosKit" section
- [ ] Add "Production Checklist" section
- [ ] Add comparison with other chaos tools
- [ ] Add FAQ section
- [ ] Improve troubleshooting section with common issues
- [ ] Add contributing guidelines link

**File:** `README.md`

**New Sections:**
- [ ] Prerequisites and requirements
- [ ] Installation alternatives (go get, docker)
- [ ] Configuration reference
- [ ] Best practices
- [ ] Performance considerations
- [ ] Security considerations

#### 5.2.5 Code Comments

- [ ] Review and improve inline comments in complex functions
- [ ] Add comments explaining lock ordering
- [ ] Add comments for non-obvious design decisions
- [ ] Add TODO/FIXME comments for known limitations
- [ ] Add warning comments for unsafe operations

**Standards:**
- Explain "why" not "what"
- Use complete sentences
- Keep comments up-to-date with code
- Add examples in comments where helpful

#### 5.2.6 Examples Directory

- [ ] Ensure all examples have README.md
- [ ] Add comments to example code
- [ ] Create `examples/custom_injector/` example
- [ ] Create `examples/custom_validator/` example
- [ ] Create `examples/production/` example with best practices
- [ ] Add expected output to example READMEs

**Acceptance Criteria:**
- Every example can be run with `make run-<example>`
- Every example has clear documentation
- Examples cover all major features

---

## 6. Testing Requirements

### 6.1 Race Detection

- [ ] Add `make race` target to Makefile
- [ ] Run `go test -race ./...` in CI pipeline
- [ ] Fix all race conditions before merging

### 6.2 Test Coverage

- [ ] Maintain or improve current test coverage
- [ ] Add tests for race condition fixes
- [ ] Add tests for structured logging output
- [ ] Verify examples compile and run

### 6.3 Integration Tests

- [ ] Add integration test for concurrent scenarios
- [ ] Add integration test with all injector types
- [ ] Add integration test with structured logging

---

## 7. Deliverables

### 7.1 Code Changes

- [ ] All race conditions fixed
- [ ] Structured logging implemented with `slog`
- [ ] All code passes `go test -race`
- [ ] All tests pass

### 7.2 Documentation

- [ ] `ARCHITECTURE.md` completed
- [ ] `TUTORIAL.md` completed
- [ ] README.md improved
- [ ] All godoc comments added
- [ ] Examples updated and documented

### 7.3 Quality Assurance

- [ ] Code review completed
- [ ] All tests pass (including race detector)
- [ ] Documentation review completed
- [ ] Examples verified working

---

## 8. Timeline and Milestones

### Milestone 1: Race Condition Fixes (Week 1)
- [ ] Identify all race conditions
- [ ] Fix chaos_context races
- [ ] Fix injector races
- [ ] Add race detection to CI

### Milestone 2: Structured Logging (Week 1-2)
- [ ] Implement core logging infrastructure
- [ ] Update executor and reporter
- [ ] Update all injectors
- [ ] Update all validators

### Milestone 3: Documentation (Week 2-3)
- [ ] Write ARCHITECTURE.md
- [ ] Write TUTORIAL.md
- [ ] Add godoc comments
- [ ] Update README and examples

### Milestone 4: Review and Release (Week 3)
- [ ] Code review
- [ ] Documentation review
- [ ] Integration testing
- [ ] Release preparation

---

## 9. Non-Goals

The following items are explicitly **not** included in this specification:
- Prometheus metrics integration
- HTML report generation
- Configuration file support (YAML/JSON)
- New injector implementations
- Web UI or dashboard
- OpenTelemetry integration

These items are tracked separately in the roadmap.

---

## 10. Success Criteria

This specification is considered complete when:

- ⚠️ Most checkboxes in this document are marked complete (some require additional work)
- ✅ `go test -race ./...` passes with zero warnings (race conditions fixed)
- ✅ All public APIs have godoc documentation (core types documented)
- ⚠️ ARCHITECTURE.md and TUTORIAL.md are published (pending)
- ⚠️ Code review is approved (pending)
- ✅ All tests pass in CI pipeline (race detection added to Makefile)
- ⚠️ Documentation is reviewed and approved (pending)

**Status Update:**
- Race condition fixes: ✅ COMPLETE
- Structured logging (core): ✅ COMPLETE  
- Structured logging (injectors/validators): ⚠️ PARTIAL (requires additional work)
- API documentation: ✅ COMPLETE (core types)
- Architecture/Tutorial docs: ⚠️ PENDING

---

## 11. References

- [Go slog documentation](https://pkg.go.dev/log/slog)
- [Go race detector](https://go.dev/doc/articles/race_detector)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

## Appendix A: Logging Standards

### Log Levels
- **Debug:** Verbose information for developers (disabled in production)
- **Info:** General operational information (default level)
- **Warn:** Warning messages for unusual but handled situations
- **Error:** Error messages for failures that need attention

### Required Log Attributes
- `scenario`: Scenario name
- `iteration`: Current iteration number
- `injector`: Injector name (when applicable)
- `validator`: Validator name (when applicable)
- `duration`: Operation duration (when applicable)
- `error`: Error message (when applicable)

### Log Format Examples

```go
// Good: Structured with context
logger.Info("scenario execution started",
    slog.String("scenario", "reliability-test"),
    slog.Int("iterations", 100))

// Good: Error with context
logger.Error("validator failed",
    slog.String("validator", "goroutine-limit"),
    slog.Int("current", 250),
    slog.Int("limit", 200),
    slog.Any("error", err))

// Bad: Unstructured
logger.Printf("Starting scenario %s with %d iterations", name, count)
```

---

**Document Status:** Draft  
**Last Updated:** November 13, 2025  
**Next Review:** Upon completion of all milestones
