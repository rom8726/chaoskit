# ChaosKit

![Go Version](https://img.shields.io/badge/Go-%3E=1.25-blue)
[![Go Reference](https://pkg.go.dev/badge/github.com/rom8726/chaoskit.svg)](https://pkg.go.dev/github.com/rom8726/chaoskit)
[![Go Report Card](https://goreportcard.com/badge/github.com/rom8726/chaoskit)](https://goreportcard.com/report/github.com/rom8726/chaoskit)
[![Coverage Status](https://coveralls.io/repos/github/rom8726/chaoskit/badge.svg?branch=main)](https://coveralls.io/github/rom8726/chaoskit?branch=main)
![Build Tag: chaos](https://img.shields.io/badge/build--tag-chaos-important)
[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)


**ChaosKit** is a Go framework for **code-level chaos engineering** and **fault injection**.
It enables controlled failures inside your functions — delays, panics, errors, resource pressure — and provides validators to ensure your application behaves correctly under adverse conditions.

Most chaos tools operate at the infrastructure level (containers, nodes, networks). ChaosKit focuses on what they cannot test: **business logic, compensations, concurrency behavior, state invariants, and internal error-handling paths**.

<img src="docs/chaoskit_logo.png" alt="ChaosKit Logo" width="500" />

---

## Features

* **Code-level fault injection** (panic, delay, error, resource faults)
* **Context-based activation** (no-op unless chaos context is attached)
* **Scenario DSL** (define steps, injectors, validators)
* **Built-in validators** (goroutine limits, recursion depth, infinite loop detection)
* **Long-running chaos tests** (minutes to days)
* **Integration with ToxiProxy** for network chaos
* **Optional monkey-patching** for advanced cases
* **Fully opt-in via build tags** (`-tags=chaos`)

ChaosKit is safe to include in your codebase:
fault injection **never activates accidentally**.

---

## Installation

```sh
go get github.com/rom8726/chaoskit
```

---

## Quick Example

```go
scenario := chaoskit.NewScenario("workflow-stability").
    WithTarget(engine).
    Step("run workflow", ExecuteWorkflow).
    Inject("delays", injectors.RandomDelay(10*time.Millisecond, 100*time.Millisecond)).
    Inject("panic", injectors.PanicProbability(0.01)).
    Assert("goroutines", validators.GoroutineLimit(200)).
    Assert("no-deadlock", validators.NoInfiniteLoop(5*time.Second)).
    Repeat(100).
    Build()

if err := chaoskit.Run(context.Background(), scenario); err != nil {
    log.Fatalf("Chaos scenario failed: %v", err)
}
```

---

## Build Tag Isolation

ChaosKit’s fault injectors are compiled **only** when explicitly enabled:

```sh
go build -tags=chaos .
```

Without the `chaos` tag:

* all injectors become **no-ops**,
* chaos code is **excluded** from the binary,
* scenarios run without injecting any faults.

This ensures ChaosKit never affects production binaries unless intentionally enabled.

---

## Context-Based Activation

Even with the build tag, chaos runs only when a special context is attached:

```go
ctx := chaoskit.AttachChaos(context.Background())
```

Without this context, calls like:

```go
chaoskit.MaybePanic(ctx)
chaoskit.MaybeDelay(ctx)
```

are strictly **no-op**.

---

## Scenario DSL

Define multi-step, multi-injector stress or chaos tests:

```go
chaoskit.NewScenario("example").
    WithTarget(client).
    Step("call API", callAPI).
    Inject("latency", injectors.RandomDelay(5*time.Millisecond, 50*time.Millisecond)).
    Inject("errors", injectors.ErrorProbability(io.ErrUnexpectedEOF, 0.02)).
    Assert("goroutines", validators.GoroutineLimit(100)).
    Repeat(50).
    Build()
```

ChaosKit supports:

* fixed-number runs (`Repeat(n)`)
* duration-based runs (`RunFor(time.Hour)`)

---

## Injectors

* `PanicProbability(p)`
* `RandomDelay(min, max)`
* `ErrorProbability(err, p)`
* `CompositeInjector(...)`
* network chaos via **ToxiProxy**
* optional monkey-patching for advanced scenarios

---

## Validators

* `GoroutineLimit(n)`
* `RecursionDepthLimit(n)`
* `NoInfiniteLoop(timeout)`
* `MemoryLimit(bytes)`
* custom validators via:

```go
type Validator interface {
    Validate(target Target) error
}
```

---

## When to Use ChaosKit

### Best use cases

* workflow engines (Saga, orchestration)
* systems with compensations or rollback logic
* retry-heavy algorithms
* concurrency-sensitive components
* correctness-critical state machines
* CI stress testing

### Less suitable for

* black-box testing without code access
* infrastructure-level chaos (use Chaos Mesh / Litmus / Gremlin instead)

---

## Network Chaos (ToxiProxy)

ChaosKit integrates with ToxiProxy without modifying your code:

* latency
* bandwidth limits
* timeouts
* connection cuts
* packet shaping

Useful for testing clients, message brokers, databases, etc.

---

## Why Code-Level Chaos Matters

Infrastructure chaos exposes resilience of clusters.
**ChaosKit exposes resilience of your logic.**

Examples of failures ChaosKit can detect:

* rollback recursion loops
* leaked goroutines
* unbounded retries
* inconsistent state after error paths
* panics during compensations
* subtle timing bugs

This is the category of failures that infrastructure-level tools cannot simulate.

---

## License (MIT)

ChaosKit is released under the **MIT License**.

**This software is provided “as is”, without warranty of any kind**,
express or implied, including but not limited to the warranties of
merchantability, fitness for a particular purpose, and noninfringement.
In no event shall the authors or copyright holders be liable for any
claim, damages, or other liability arising from the use of this software.

See [LICENSE](./LICENSE) for details.

---

## Status

ChaosKit is early-stage but stable enough for research, prototyping, and internal testing workflows.
Contributions, issue reports, and design discussions are welcome.
