# kubewhy

Explains Kubernetes failures in plain English

## Overview

`kubewhy` is a diagnostic CLI tool that helps you understand why your Kubernetes resources are failing. Instead of parsing through `kubectl describe` output and cryptic error messages, kubewhy tells you exactly what's wrong and how to fix it.

## Installation

### Prerequisites

- Go 1.21 or later
- Access to a Kubernetes cluster (via kubeconfig)

### Build from Source

```bash
git clone https://github.com/michaelkoro/kubewhy.git
cd kubewhy
go mod tidy
go build -o kubewhy ./cmd/kubewhy
```

### Install Directly

```bash
go install github.com/michaelkoro/kubewhy/cmd/kubewhy@latest
```

## Usage

### Basic Commands

```bash
# Test connection to the Kubernetes cluster
kubewhy check

# Diagnose a specific pod
kubewhy pod my-failing-pod

# Diagnose all problematic pods in current namespace
kubewhy pod

# Diagnose pods in a specific namespace
kubewhy pod -n kube-system

# Diagnose pods across all namespaces
kubewhy pod -A
```

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--kubeconfig` | | Path to kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`) |
| `--namespace` | `-n` | Target Kubernetes namespace |
| `--all-namespaces` | `-A` | Search across all namespaces |
| `--help` | `-h` | Show help for any command |

## Testing

### Integration tests

Integration tests deploy deliberately broken pods to a live cluster and verify that the analyzer diagnoses each failure correctly. They require a running Kubernetes cluster — a local [k3d](https://k3d.io) cluster works well.

The tests are gated behind the `integration` build tag so they never run during a normal `go test ./...`.

```bash
# Run all integration tests (all resource types)
go test -v -tags integration ./tests/integration/... -timeout 5m

# Run only pod analyzer tests
go test -v -tags integration ./tests/integration/pod/... -timeout 3m

# Run a single subtest by name
go test -v -tags integration ./tests/integration/pod/... -timeout 3m -run TestContainerDiagnosis/CrashLoop
```

Make sure your active kubeconfig context points at the target cluster before running. Each test run creates an isolated `kubewhy-integration-<timestamp>` namespace and deletes it when finished.

## Project Structure

```
kubewhy/
├── cmd/kubewhy/main.go          # CLI entry point
├── internal/
│   ├── analyzer/
│   │   └── pod/                 # Pod failure analyzer
│   │       ├── analyzer.go      # Orchestrates all checks
│   │       ├── container.go     # Image pull, crash loop, OOMKilled, runtime errors
│   │       └── result.go        # DiagnosisResult and PodDiagnosis types
│   ├── cli/
│   │   ├── root.go              # Root command and global flags
│   │   ├── check.go             # Cluster connection check
│   │   └── pod.go               # Pod diagnosis subcommand
│   └── k8s/
│       └── client.go            # Kubernetes client wrapper
├── tests/
│   └── integration/
│       ├── framework/           # Shared test infrastructure (Env, WaitFor)
│       └── pod/                 # Pod analyzer integration tests
├── go.mod
└── README.md
```

## Contributing

Contributions are welcome! To add a new resource analyzer:

1. Create a new analyzer in `internal/analyzer/<resource>/`
2. Add a new CLI command in `internal/cli/<resource>.go`
3. Register the command in `internal/cli/root.go`

## License

MIT License - see [LICENSE](LICENSE) for details.
