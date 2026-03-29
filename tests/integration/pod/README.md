# Pod Analyzer Integration Tests

Integration tests that deploy deliberately broken pods to a live Kubernetes
cluster and verify the analyzer produces the correct diagnosis category.

## Test suites

### `TestContainerDiagnosis` — container-level failures

Defined in `container_test.go`. Each subtest waits for the pod to reach the
expected bad container state before invoking the analyzer.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `ImagePull` | Image `does-not-exist-registry.io/fake-image:missing` — nonexistent registry | `ImagePull` |
| `CrashLoop` | `busybox` with `command: ["sh", "-c", "exit 1"]` + `OnFailure` restart policy | `CrashLoop` |
| `OOMKilled` | `polinux/stress` allocating 32 MB against a 4 Mi memory limit | `OOMKilled` |
| `RuntimeError` | `busybox` with `command: ["/this-binary-does-not-exist"]` | `RuntimeError` |

### `TestInitContainerDiagnosis` — init container failures

Defined in `init_test.go`. Each subtest deploys a pod whose init container is
deliberately broken and waits for it to enter the expected failure state in
`InitContainerStatuses`. A failing init container blocks all regular containers
from starting, which the analyzer makes explicit in its output.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `InitCrashLoop` | Init container runs `exit 1` + `OnFailure` restart policy | `InitCrashLoop` |
| `InitImagePull` | Init container references `does-not-exist-registry.io/fake-image:missing` | `InitImagePull` |

### `TestProbeDiagnosis` — probe failures

Defined in `probes_test.go`. Each subtest deploys a `busybox sleep 3600` pod
with a deliberately broken exec probe (`command: ["false"]`) and waits for the
expected failure state before invoking the analyzer.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `LivenessProbe` | Liveness exec probe `["false"]`, `failureThreshold=10` — generates `Unhealthy` events; high threshold prevents the container being killed before analysis | `ProbeFailure` |
| `ReadinessProbe` | Readiness exec probe `["false"]`, `failureThreshold=1` — pod stays Running with `ContainersReady=False` indefinitely | `ProbeFailure` |

### `TestPendingDiagnosis` — scheduling failures

Defined in `pending_test.go`. Each subtest deploys a pod that the scheduler
cannot place on any node and waits for the `PodScheduled=False` condition.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `InsufficientCPU` | Requests `9999` CPU cores — exceeds any real node's allocatable capacity | `Scheduling` |
| `NodeAffinityMismatch` | `RequiredDuringScheduling` affinity on label `kubewhy.io/nonexistent-label=required` that no node carries | `Scheduling` |

### `TestConfigDiagnosis` — config errors

Defined in `config_test.go`. Each subtest deploys a pod referencing a
non-existent ConfigMap or Secret and waits for the container to enter
`CreateContainerConfigError`.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `MissingConfigMap` | `envFrom` referencing `does-not-exist-configmap` | `ConfigError` |
| `MissingSecret` | `envFrom` referencing `does-not-exist-secret` | `ConfigError` |

### `TestVolumeDiagnosis` — volume mount failures

Defined in `volume_test.go`. Each subtest deploys a pod that references a
non-existent PVC and waits for `FailedMount` events.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `MissingPVC` | Mounts `does-not-exist-pvc` as a volume | `VolumeMount` |

### `TestEvictionDiagnosis` — eviction

Defined in `eviction_test.go`. Simulates eviction by patching a running pod's
status subresource to `Phase=Failed, Reason=Evicted`.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `Evicted` | Status patched to simulate MemoryPressure eviction | `Eviction` |

### `TestTerminatingDiagnosis` — stuck Terminating

Defined in `terminating_test.go`. Each subtest deploys a pod with a custom
finalizer, deletes it, and verifies the analyzer reports it as stuck.

| Subtest | How it breaks | Expected category |
|---|---|---|
| `StuckFinalizer` | Pod has finalizer `kubewhy.io/test-blocker`; deleted with grace=0 but finalizer blocks removal | `StuckTerminating` |

## Running

Run all integration tests:

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 5m
```

Run a single suite:

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestContainerDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestInitContainerDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestPendingDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestProbeDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestConfigDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestVolumeDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestEvictionDiagnosis
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestTerminatingDiagnosis
```

Run a single case:

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestContainerDiagnosis/CrashLoop
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestInitContainerDiagnosis/InitCrashLoop
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestPendingDiagnosis/InsufficientCPU
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestProbeDiagnosis/LivenessProbe
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestProbeDiagnosis/ReadinessProbe
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestConfigDiagnosis/MissingConfigMap
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestVolumeDiagnosis/MissingPVC
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestEvictionDiagnosis/Evicted
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestTerminatingDiagnosis/StuckFinalizer
```
