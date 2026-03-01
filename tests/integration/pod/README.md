# Pod Analyzer Integration Tests

Each test in `container_test.go` deploys a deliberately broken pod to the cluster
and asserts that the analyzer produces the correct diagnosis category.

## Test cases

| Subtest | How it breaks | Expected category |
|---|---|---|
| `ImagePull` | Image `does-not-exist-registry.io/fake-image:missing` — nonexistent registry | `ImagePull` |
| `CrashLoop` | `busybox` with `command: ["sh", "-c", "exit 1"]` + `OnFailure` restart policy | `CrashLoop` |
| `OOMKilled` | `polinux/stress` allocated 32 MB against a 4 Mi memory limit | `OOMKilled` |
| `RuntimeError` | `busybox` with `command: ["/this-binary-does-not-exist"]` | `RuntimeError` |

## Running

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 3m
```

To run a single case:

```bash
go test -v -tags integration ./tests/integration/pod/... -timeout 3m -run TestContainerDiagnosis/CrashLoop
```
