# ChaosKit Safety Guard

ChaosKit provides powerful mechanisms for fault injection and chaos testing.
These features **must not** be used in production environments without explicit, deliberate configuration.

## 1. Build Tag Isolation

ChaosKit’s chaos injectors are compiled only when the `chaos` build tag is enabled:

```sh
go build -tags=chaos ./...
```

Without this tag, all chaos-related code is excluded from the binary.

## 2. Responsibility

ChaosKit does **not** guarantee safe behavior, correctness, or isolation under all conditions.
You are responsible for ensuring proper environment separation between production and testing systems.

## 3. Recommended Practices

* Use chaos injection only in CI, staging, or sandbox clusters.
* Avoid embedding chaos configuration in long-lived binaries.
* Keep chaos scenarios versioned and reproducible.
* Review all `-tags=chaos` usage in build pipelines.

ChaosKit is powerful — use it intentionally.
