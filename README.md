# Saola CLI

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)

**English** | [中文](README_zh.md)

The command-line tool for [OpenSaola](https://gitee.com/opensaola/opensaola) — manage middleware lifecycle on Kubernetes: package installation, instance creation, status monitoring, version upgrades, and resource cleanup.

## Features

- **kubectl-style** — verb + resource type (`get`/`create`/`delete`/`describe`), minimal learning curve
- **Interactive creation** — TUI form guides baseline selection and parameter input, no manual YAML needed
- **Bilingual** — `--lang zh|en` switches all help text and output messages
- **Package management** — build, install, upgrade, and uninstall middleware packages (zstd-compressed TAR)
- **Instance upgrades** — annotation-driven rolling upgrades with `--wait` for completion
- **Multiple output formats** — `table`/`yaml`/`json`/`wide`/`name` for easy scripting

## Installation

### Build from source

```bash
git clone https://gitee.com/opensaola/saola-cli.git
cd saola-cli
make build
```

The binary is at `bin/saola`. Copy it to your `$PATH`:

```bash
cp bin/saola /usr/local/bin/
```

### go install

```bash
go install gitee.com/opensaola/saola-cli/cmd/saola@latest
```

## Quick Start

```bash
# 1. Install a middleware package
saola install ./redis-pkg -n middleware-operator --wait 5m

# 2. Interactively create Middleware and MiddlewareOperator
saola create

# 3. Check instance status
saola get middleware -n my-ns
saola get operator -n my-ns

# 4. View instance details
saola describe middleware my-redis -n my-ns

# 5. Upgrade instance version
saola upgrade middleware my-redis --to-version 2.0.0 -n my-ns --wait 5m

# 6. Delete instance
saola delete middleware my-redis -n my-ns
```

## Command Reference

### Top-level Commands

| Command | Description |
|---------|-------------|
| `create` | Create Middleware / MiddlewareOperator from YAML or interactive TUI |
| `get` | List or view resources |
| `describe` | Show resource details (Spec, Status, Conditions) |
| `delete` | Delete resources |
| `run` | Trigger a MiddlewareAction (one-off operation) |
| `upgrade` | Upgrade package or Middleware / MiddlewareOperator instances |
| `install` | Install a middleware package to the cluster |
| `uninstall` | Uninstall a middleware package |
| `build` | Build a local directory into a zstd-compressed TAR (without installing) |
| `inspect` | View installed package contents and metadata |
| `version` | Show version information |

### Resource Subcommands

Using `get` as an example — other verbs (`describe`/`delete`/`upgrade`) follow the same pattern:

| Subcommand | Alias | Description |
|------------|-------|-------------|
| `get middleware [name]` | `mw` | List or view Middleware instances |
| `get operator [name]` | `op` | List or view MiddlewareOperator instances |
| `get action [name]` | `act` | List or view MiddlewareAction instances |
| `get baseline [name]` | `bl` | View baselines from installed packages |
| `get package [name]` | `pkg` | List or view installed packages |
| `get all` | - | Aggregate output of middleware + operator + action |

### upgrade Subcommands

```bash
# Package upgrade (replace installed Package Secret)
saola upgrade <pkg-dir>

# Instance upgrade (trigger controller via annotation)
saola upgrade middleware <name> --to-version <version> [--baseline <bl>] [--wait 5m]
saola upgrade operator <name>   --to-version <version> [--baseline <bl>] [--wait 5m]
```

### run Command

```bash
# Trigger an Action (e.g., backup, restore)
saola run <action-name> --middleware <mw-name> --params key1=val1,key2=val2 -n my-ns
```

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--kubeconfig` | - | `$KUBECONFIG` or `~/.kube/config` | kubeconfig file path |
| `--context` | - | - | kubeconfig context |
| `--namespace` | `-n` | - | Target namespace |
| `--pkg-namespace` | - | `middleware-operator` | Namespace where package Secrets reside |
| `--lang` | - | `zh` | Language: `zh` (Chinese) / `en` (English) |
| `--no-color` | - | `false` | Disable colored output |

Resource commands also support:

| Flag | Short | Description |
|------|-------|-------------|
| `--all-namespaces` | `-A` | List resources across all namespaces |
| `--output` | `-o` | Output format: `table`/`yaml`/`json`/`wide`/`name` |
| `--wait` | - | Timeout for waiting on operations (e.g., `5m`) |
| `--dry-run` | - | Preview changes without executing |

## Configuration

### Environment Variables

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | kubeconfig file path |
| `SAOLA_NAMESPACE` | Default namespace |
| `SAOLA_PKG_NAMESPACE` | Package Secret namespace (default: `middleware-operator`) |

### Priority

CLI Flag > Environment Variable > Default Value

## Project Structure

```
saola-cli/
├── cmd/saola/          # Entry point
├── internal/
│   ├── app/            # Root command registration
│   ├── cmd/            # All subcommand implementations
│   │   ├── action/     # MiddlewareAction
│   │   ├── baseline/   # Baseline queries
│   │   ├── build/      # Package building
│   │   ├── create/     # Resource creation (with interactive TUI)
│   │   ├── delete/     # Resource deletion
│   │   ├── describe/   # Resource details
│   │   ├── get/        # Resource listing
│   │   ├── inspect/    # Package inspection
│   │   ├── install/    # Package installation
│   │   ├── middleware/  # Middleware resource group
│   │   ├── operator/   # MiddlewareOperator resource group
│   │   ├── pkgcmd/     # Package management commands
│   │   ├── upgrade/    # Upgrades
│   │   ├── uninstall/  # Package uninstallation
│   │   ├── run/        # Action execution
│   │   └── version/    # Version info
│   ├── client/         # Kubernetes client wrapper
│   ├── config/         # Configuration management
│   ├── consts/         # Project constants
│   ├── lang/           # Bilingual support (zh/en)
│   ├── packager/       # TAR/zstd packaging
│   ├── printer/        # Output formatting (table/yaml/json)
│   ├── version/        # Version info injection
│   └── waiter/         # Async wait logic
├── Makefile
└── go.mod
```

## Build & Test

```bash
# Build
make build          # compile to bin/saola

# Test
make test           # run unit tests
make lint           # go vet static analysis

# Other
make tidy           # tidy go modules
make clean          # clean build artifacts
```

Version information is injected via ldflags:

```bash
saola version
# Version: v0.1.0
# Git Commit: 6bd4f94
# Build Date: 2026-03-31T07:46:11Z
```

## Managed CRD Types

Saola manages the following Kubernetes custom resources via the [OpenSaola](https://gitee.com/opensaola/opensaola) operator:

| CRD | Short | Description |
|-----|-------|-------------|
| Middleware | `mid` | Middleware instance |
| MiddlewareOperator | `mo` | Operator instance managing a middleware type |
| MiddlewareAction | `ma` | One-off operations (backup, restore, etc.) |
| MiddlewareBaseline | `mb` | Default spec template for middleware |
| MiddlewareOperatorBaseline | `mob` | Default spec template for operators |
| MiddlewarePackage | `mp` | Packaged middleware distribution unit |

## Documentation

- [OpenSaola Technical Documentation](https://gitee.com/opensaola/opensaola/blob/master/docs/opensaola-technical.md)
- [Package Authoring Guide](https://gitee.com/opensaola/opensaola/blob/master/docs/opensaola-packaging.md)
- [Troubleshooting Guide](https://gitee.com/opensaola/opensaola/blob/master/docs/troubleshooting.md)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) (coming soon) for guidelines.

## License

Copyright 2025 The OpenSaola Authors.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.
