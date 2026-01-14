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

## Project Structure

```
kubewhy/
├── cmd/kubewhy/main.go      # CLI entry point
├── internal/
│   ├── cli/
│   │   ├── root.go          # Root command and global flags
│   │   └── pod.go           # Pod diagnosis subcommand
│   └── k8s/
│       └── client.go        # Kubernetes client wrapper
├── go.mod
└── README.md
```

## Development Status

This project is under active development. Current status:

- [x] Project structure and CLI framework
- [x] Kubernetes client wrapper
- [ ] Pod analyzer (diagnoses pending, crash loops, image errors)
- [ ] Output formatter (colored terminal output)
- [ ] Service analyzer
- [ ] Ingress analyzer

## Contributing

Contributions are welcome! To add a new resource analyzer:

1. Create a new analyzer in `internal/analyzer/<resource>/`
2. Add a new CLI command in `internal/cli/<resource>.go`
3. Register the command in `internal/cli/root.go`

## License

MIT License - see [LICENSE](LICENSE) for details.
