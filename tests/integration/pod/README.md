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
```

Run a single case:

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestContainerDiagnosis/CrashLoop
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestInitContainerDiagnosis/InitCrashLoop
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestPendingDiagnosis/InsufficientCPU
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestProbeDiagnosis/LivenessProbe
go test -v -tags integration ./tests/integration/pod/... -timeout 5m -run TestProbeDiagnosis/ReadinessProbe
```
