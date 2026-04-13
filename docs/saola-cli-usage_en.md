# Saola CLI Usage Guide

[中文](saola-cli-usage.md) | **English**

## Table of Contents

- [1. Overview](#1-overview)
  - [1.1 What is saola-cli](#11-what-is-saola-cli)
  - [1.2 Core Features](#12-core-features)
  - [1.3 Relationship with OpenSaola / dataservice-baseline](#13-relationship-with-opensaola--dataservice-baseline)
  - [1.4 Installation and Build](#14-installation-and-build)
- [2. Global Configuration](#2-global-configuration)
  - [2.1 Kubeconfig Configuration](#21-kubeconfig-configuration)
  - [2.2 Namespace Options](#22-namespace-options)
  - [2.3 Output Format (-o flag)](#23-output-format--o-flag)
  - [2.4 Other Global Flags](#24-other-global-flags)
  - [2.5 Environment Variables](#25-environment-variables)
- [3. Command Reference](#3-command-reference)
  - [3.1 Package Management (saola pkg)](#31-package-management-saola-pkg)
  - [3.2 Middleware Instance Management (saola middleware)](#32-middleware-instance-management-saola-middleware)
  - [3.3 Operator Management (saola operator)](#33-operator-management-saola-operator)
  - [3.4 Baseline Management (saola baseline)](#34-baseline-management-saola-baseline)
  - [3.5 Action Management (saola action)](#35-action-management-saola-action)
  - [3.6 Shortcut Commands](#36-shortcut-commands)
- [4. Package Specification](#4-package-specification)
  - [4.1 TAR Package Format and Directory Structure](#41-tar-package-format-and-directory-structure)
  - [4.2 metadata.yaml Requirements](#42-metadatayaml-requirements)
  - [4.3 Package State Machine](#43-package-state-machine)
- [5. Secret Structure](#5-secret-structure)
  - [5.1 Secret Generated After Package Installation](#51-secret-generated-after-package-installation)
  - [5.2 Secret Labels and Annotations](#52-secret-labels-and-annotations)
  - [5.3 Secret data Fields](#53-secret-data-fields)
- [6. Complete Workflow Examples](#6-complete-workflow-examples)
  - [6.1 Complete Flow from Scratch](#61-complete-flow-from-scratch)
  - [6.2 Package Lifecycle](#62-package-lifecycle)
  - [6.3 Middleware Instance Lifecycle](#63-middleware-instance-lifecycle)
  - [6.4 Operator Lifecycle](#64-operator-lifecycle)
- [7. Interactive Creation Details](#7-interactive-creation-details)
  - [7.1 Interactive Flow](#71-interactive-flow)
  - [7.2 JSON Schema Support](#72-json-schema-support)
  - [7.3 Parameter Validation](#73-parameter-validation)
- [8. Output Formats](#8-output-formats)
  - [8.1 Table Format](#81-table-format)
  - [8.2 JSON Format](#82-json-format)
  - [8.3 YAML Format](#83-yaml-format)
  - [8.4 Name Format](#84-name-format)
- [9. Configuration and Environment](#9-configuration-and-environment)
  - [9.1 Configuration File](#91-configuration-file)
  - [9.2 Environment Variables](#92-environment-variables)
- [10. Data Flow with Other Projects](#10-data-flow-with-other-projects)
  - [10.1 saola-cli --> dataservice-baseline (Consuming Packages)](#101-saola-cli----dataservice-baseline-consuming-packages)
  - [10.2 saola-cli --> OpenSaola (Creating K8s Resources)](#102-saola-cli----opensaola-creating-k8s-resources)
  - [10.3 Complete Data Flow Diagram](#103-complete-data-flow-diagram)

---

## 1. Overview

### 1.1 What is saola-cli

saola-cli is the CLI companion tool for OpenSaola, used to manage middleware packages (Package), Middleware and MiddlewareOperator custom resources, trigger Action operations, and query Baseline templates in Kubernetes clusters.

Project name: `saola-cli`, compiled binary name: `saola`.

### 1.2 Core Features

- **Package Management**: Build local directories into zstd-compressed TAR archives, install/uninstall/upgrade/list/inspect installed middleware packages.
- **Middleware Instance Management**: Create, list, describe, upgrade, and delete Middleware custom resources.
- **MiddlewareOperator Management**: Create, list, describe, upgrade, and delete MiddlewareOperator custom resources.
- **Baseline Queries**: List and view MiddlewareBaseline / MiddlewareOperatorBaseline / MiddlewareActionBaseline embedded in installed packages.
- **Action Management**: Trigger one-off operations (MiddlewareAction), query and describe their execution status.
- **Interactive Creation**: Terminal-guided forms (based on charmbracelet/huh) for selecting baselines, filling parameters, previewing YAML, and confirming creation.
- **Bilingual Support**: Switch between Chinese (default) and English via `--lang`.
- **kubectl-style Shortcut Commands**: `saola get`, `saola create`, `saola delete`, `saola describe`, `saola run`, etc.

### 1.3 Relationship with OpenSaola / dataservice-baseline

| Project | Responsibility | Relationship with saola-cli |
|---------|---------------|-----------------------------|
| **OpenSaola** | Kubernetes Operator that watches CRD changes and reconciles middleware lifecycle | Secrets / CRs created by saola-cli are consumed and processed by OpenSaola |
| **dataservice-baseline** | Provides 40+ middleware CUE/Go templates, Baseline definitions, and Action templates | Package directory contents packaged by saola-cli originate from dataservice-baseline build artifacts |

Data flow:
```
dataservice-baseline (templates/Baselines) --> local package dir --> saola-cli pack --> K8s Secret --> OpenSaola reconcile
                                                                    saola-cli create --> K8s Middleware/Operator CR --> OpenSaola reconcile
```

### 1.4 Installation and Build

```bash
# Build from source
cd saola-cli
make build

# Binary is at bin/saola
./bin/saola version
```

Version information is injected via `-ldflags` during build:

| Variable | Description |
|----------|-------------|
| `Version` | git describe tag version, defaults to `dev` |
| `GitCommit` | git rev-parse short commit hash, defaults to `unknown` |
| `BuildDate` | UTC format build time, defaults to `unknown` |

---

## 2. Global Configuration

### 2.1 Kubeconfig Configuration

saola uses standard kubeconfig loading rules:

1. `--kubeconfig` flag to explicitly specify the path
2. `$KUBECONFIG` environment variable
3. `~/.kube/config` default path

Use `--context` to specify the kubeconfig context to use.

### 2.2 Namespace Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--namespace` | `-n` | empty (falls back to `$SAOLA_NAMESPACE`, then to `default`) | Kubernetes namespace for middleware resources |
| `--pkg-namespace` | - | `middleware-operator` | Namespace where package Secrets are stored |

> Note: The DataNamespace default in OpenSaola source code is `default`, but production deployments typically override it to `middleware-operator` via configuration. Ensure saola-cli's --pkg-namespace matches the OpenSaola configuration.

### 2.3 Output Format (-o flag)

Most `get` / `list` / `inspect` commands support the `-o` / `--output` option:

| Format | Description |
|--------|-------------|
| `table` | Default, aligned table output |
| `wide` | Extended table with additional LABELS column |
| `yaml` | Full YAML format output |
| `json` | Indented JSON format output |
| `name` | Only outputs `type/name`, similar to `kubectl get -o name` |

### 2.4 Other Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--lang` | string | `zh` | Display language: `zh` (Chinese) or `en` (English) |
| `--log-level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--no-color` | bool | `false` | Disable colored output |
| `-h`, `--help` | bool | `false` | Show help information |

### 2.5 Environment Variables

| Environment Variable | Corresponding Flag | Description |
|---------------------|-------------------|-------------|
| `KUBECONFIG` | `--kubeconfig` | kubeconfig file path |
| `SAOLA_NAMESPACE` | `--namespace` | Default resource namespace |
| `SAOLA_PKG_NAMESPACE` | `--pkg-namespace` | Default package Secret namespace |

> Note: CLI flags take precedence over environment variables. Environment variables only take effect when the corresponding flag is not explicitly set.

---

## 3. Command Reference

### 3.1 Package Management (saola pkg)

Package management commands are accessed via `saola package` (alias `saola pkg`), and can also be used through top-level shortcut commands.

---

#### saola pkg build

**Syntax**: `saola package build <pkg-dir> [flags]`

**Description**: Build a local directory into a zstd-compressed TAR file without installing. Suitable for CI pipelines or offline distribution scenarios.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<pkg-dir>` | - | positional | required | Local package directory path |
| `--output` | `-o` | string | `<name>-<version>.pkg` | Output file path |

**Examples**:
```bash
# Build current directory
saola package build .

# Specify output path
saola package build ./my-redis --output ./dist/redis-v1.pkg
```

**Notes**:
- The package directory must contain a `metadata.yaml` file with `name` and `version` as required fields.
- The output directory is created automatically if it does not exist.
- Hidden files and directories (starting with `.`) are automatically skipped.

---

#### saola pkg install

**Syntax**: `saola package install <pkg-dir> [flags]`

**Description**: Build a local package directory and create an Immutable Secret in the `pkg-namespace`. OpenSaola will automatically install the package upon detecting the Secret.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<pkg-dir>` | - | positional | required | Local package directory path |
| `--name` | - | string | `<name>-<version>` | Override Secret name |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for installation to complete (e.g., `5m`) |
| `--dry-run` | - | bool | `false` | Print Secret manifest without actually creating it |

**Examples**:
```bash
# Install from current directory
saola package install .

# Specify name and wait
saola package install ./my-redis --name redis-v1 --wait 5m

# Preview only
saola package install . --dry-run
```

**Notes**:
- If a Secret with the same name already exists, the command will error and suggest using `package upgrade`.
- `--wait` polls the Secret's `enabled` label to determine if installation is complete.
- On installation failure, the `installError` annotation on the Secret will contain the error message.

---

#### saola pkg uninstall

**Syntax**: `saola package uninstall <name> [flags]`

**Description**: Add an uninstall annotation (`middleware.cn/uninstall=true`) to the package's corresponding Secret. OpenSaola will automatically uninstall the package upon detecting the annotation.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Package Secret name |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for uninstallation to complete |

**Examples**:
```bash
saola package uninstall redis-v1
saola package uninstall redis-v1 --wait 5m
```

**Notes**:
- Uninstallation completion is determined by: uninstall annotation cleared and `enabled=false`, or Secret deleted (NotFound).

---

#### saola pkg upgrade

**Syntax**: `saola package upgrade <pkg-dir> [flags]`

**Description**: Replace the package Secret with new content from the local directory. Since Immutable Secrets cannot be updated in place, the command deletes the old Secret first and recreates it with the new data.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<pkg-dir>` | - | positional | required | Local package directory path |
| `--name` | - | string | `<name>-<version>` | Override Secret name |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for installation to complete after upgrade |

**Examples**:
```bash
saola package upgrade ./my-redis
saola package upgrade ./my-redis --name redis-custom
saola package upgrade ./my-redis --wait 5m
```

**Notes**:
- If the old Secret does not exist, the effect is equivalent to a fresh install.
- There is a brief window during upgrade when no Secret exists (between deletion and recreation).

---

#### saola pkg inspect

**Syntax**: `saola package inspect <name> [flags]`

**Description**: Read the specified package's Secret from `pkg-namespace`, decompress the TAR archive, and display the file list and metadata.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Package Secret name |
| `--output` | `-o` | string | `table` | Output format: `table`, `yaml`, `json` |

**Examples**:
```bash
saola package inspect redis-v1
saola package inspect redis-v1 -o yaml
```

**Notes**:
- `table` format outputs metadata fields and file list (with file sizes).
- `yaml` / `json` format outputs the complete package data structure.

---

#### saola pkg list

**Syntax**: `saola package list [flags]` (alias: `saola package ls`)

**Description**: List all installed middleware package Secrets in `pkg-namespace`.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--component` | - | string | empty | Filter by component name |
| `--version` | - | string | empty | Filter by package version |
| `--output` | `-o` | string | `table` | Output format: `table`, `yaml`, `json` |

**Examples**:
```bash
saola package list
saola package list --component redis
saola package list -o yaml
```

**Notes**:
- Table output columns: `NAME`, `COMPONENT`, `VERSION`, `ENABLED`, `CREATED`.
- When no packages are found, outputs "No packages found."

---

### 3.2 Middleware Instance Management (saola middleware)

Middleware management commands are accessed via `saola middleware` (alias `saola mw`).

---

#### saola middleware create

**Syntax**: `saola middleware create [flags]`

**Description**: Read a Middleware manifest from the specified YAML file and create it in the cluster. The command automatically finds the matching MiddlewareBaseline from installed packages based on `spec.baseline`, and populates the following labels and `spec.operatorBaseline`:

- `middleware.cn/packagename`
- `middleware.cn/packageversion`
- `middleware.cn/component`
- `middleware.cn/definition`

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--file` | `-f` | string | required | YAML manifest file path |
| `--namespace` | `-n` | string | empty (uses value from manifest) | Override namespace in manifest |

**Examples**:
```bash
saola middleware create -f middleware.yaml
saola middleware create -f middleware.yaml -n production
```

**Notes**:
- `spec.baseline` is a required field, used to locate the matching package and baseline.
- If no matching MiddlewareBaseline is found in installed packages, the command will error.
- After creation, if `spec.operatorBaseline.name` is non-empty but the corresponding MiddlewareOperator does not exist, a warning will be printed.
- Middleware does not automatically create a MiddlewareOperator; it must be created independently.

---

#### saola middleware get

**Syntax**: `saola middleware get [name] [flags]`

**Description**: List all Middleware resources in a namespace, or get a single resource by name.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `[name]` | - | positional | optional | If specified, get a single resource |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--all-namespaces` | `-A` | bool | `false` | List across all namespaces |
| `--output` | `-o` | string | `table` | Output format: `table`, `wide`, `yaml`, `json`, `name` |

**Examples**:
```bash
saola middleware get
saola middleware get my-redis
saola middleware get my-redis -o yaml
saola middleware get -A -o json
```

**Notes**:
- Table output columns: `NAME`, `NAMESPACE`, `BASELINE`, `STATE`, `AGE`.
- `wide` format additionally shows the `LABELS` column.
- `name` format outputs `middleware/<name>`.

---

#### saola middleware describe

**Syntax**: `saola middleware describe <name> [flags]`

**Description**: Get a single Middleware resource and output its spec, status, and conditions in a human-readable format.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Middleware resource name |
| `--namespace` | `-n` | string | empty | Target namespace |

**Examples**:
```bash
saola middleware describe my-redis
saola middleware describe my-redis -n production
```

**Output content**:
- Metadata: Name, Namespace, Age, Labels, Annotations
- Spec: Baseline, OperatorBaseline (Name/GvkName), Configurations, PreActions
- Status: State, Reason, ObservedGeneration, CustomResources (Type/Phase/Replicas/Reason)
- Conditions: TYPE, STATUS, REASON, MESSAGE

---

#### saola middleware upgrade

**Syntax**: `saola middleware upgrade <name> [flags]` (accessed via `saola upgrade middleware`)

**Description**: Trigger OpenSaola to perform a version upgrade by setting `middleware.cn/update` and `middleware.cn/baseline` annotations on the Middleware CR.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Middleware instance name |
| `--to-version` | - | string | required | Target version number |
| `--baseline` | - | string | empty (keep current value) | Target baseline name |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for upgrade to complete |

**Examples**:
```bash
saola upgrade middleware my-redis --to-version 7.2.1
saola upgrade middleware my-redis --to-version 7.2.1 --baseline redis-standalone-v2
saola upgrade mw my-redis --to-version 7.2.1 --wait 5m -n production
```

**Notes**:
- If the previous upgrade has not yet completed (annotation still exists), the command will refuse to execute and report an error.
- `--wait` polls every 3 seconds to check: `middleware.cn/update` annotation has disappeared and `status.state == Available`.
- Upgrade flow executed by controller: find target version package -> switch Spec.Baseline -> update Labels -> remove annotation -> State: Updating -> Available.

---

#### saola middleware delete

**Syntax**: `saola middleware delete <name> [flags]`

**Description**: Delete a Middleware resource by name. Silently skips if the resource does not exist.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Middleware resource name |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for deletion to complete (polls every 2 seconds) |

**Examples**:
```bash
saola middleware delete my-redis
saola middleware delete my-redis -n production
saola middleware delete my-redis --wait 2m
```

---

### 3.3 Operator Management (saola operator)

Operator management commands are accessed via `saola operator`.

---

#### saola operator create

**Syntax**: `saola operator create [flags]`

**Description**: Read a MiddlewareOperator manifest from a YAML file and create it in the cluster. The command automatically finds the matching MiddlewareOperatorBaseline from installed packages based on `spec.baseline`, and populates the four required labels.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--file` | `-f` | string | required | YAML manifest file path |
| `--namespace` | `-n` | string | empty | Override namespace in manifest |

**Examples**:
```bash
saola operator create -f operator.yaml
saola operator create -f operator.yaml --namespace my-ns
```

**Notes**:
- Checks for existing resource with the same name before creation; errors if it already exists.
- If no matching MiddlewareOperatorBaseline is found in installed packages, the command will error.
- `namespace` is required: must be provided via `--namespace`, the manifest's namespace, or global configuration.

---

#### saola operator get

**Syntax**: `saola operator get [name] [flags]`

**Description**: List or query MiddlewareOperator resources.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `[name]` | - | positional | optional | If specified, get a single resource |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--all-namespaces` | `-A` | bool | `false` | List across all namespaces |
| `--output` | `-o` | string | `table` | Output format: `table`, `wide`, `yaml`, `json`, `name` |

**Examples**:
```bash
saola operator get
saola operator get redis-operator -n my-ns
saola operator get -A -o yaml
```

**Notes**:
- Table output columns: `NAME`, `NAMESPACE`, `BASELINE`, `STATE`, `READY`, `RUNTIME`, `AGE`.
- `wide` format additionally shows the `LABELS` column.
- Namespace is required when querying a single resource.

---

#### saola operator describe

**Syntax**: `saola operator describe <name> [flags]`

**Description**: Print the complete spec, status, conditions, and Deployment status for each operator of a MiddlewareOperator.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | MiddlewareOperator resource name |
| `--namespace` | `-n` | string | empty (required) | Target namespace |

**Examples**:
```bash
saola operator describe redis-operator -n my-ns
```

**Output content**:
- Metadata: Name, Namespace, Labels, Annotations, Created
- Spec: Baseline, PermissionScope, Configurations, PreActions (including fixed/exposed properties)
- Status: State, Ready, Runtime, OperatorAvailable, Reason, ObservedGeneration
- Conditions: TYPE, STATUS, REASON, MESSAGE
- OperatorStatus: Replicas, Ready, Available, Updated, Conditions for each Deployment
- Finalizers

---

#### saola operator upgrade

**Syntax**: `saola operator upgrade <name> [flags]` (accessed via `saola upgrade operator`)

**Description**: Trigger controller upgrade by setting `middleware.cn/update` and `middleware.cn/baseline` annotations.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | MiddlewareOperator instance name |
| `--to-version` | - | string | required | Target upgrade version |
| `--baseline` | - | string | empty (keep current Baseline) | Target Baseline name |
| `--namespace` | `-n` | string | empty (required) | Target namespace |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for upgrade to complete |

Alias: `op`

**Examples**:
```bash
saola upgrade operator redis-op --to-version 1.2.0 -n my-ns
saola upgrade operator redis-op --to-version 1.2.0 --baseline redis-ha -n my-ns
saola upgrade op redis-op --to-version 1.2.0 -n my-ns --wait 5m
```

**Notes**:
- If the `middleware.cn/update` annotation already exists, the previous upgrade has not yet completed and the command will refuse to execute.
- The controller preserves Globe and PreActions during processing, clears remaining Spec, and switches to the new Baseline.
- `--wait` polls every 2 seconds to check: annotation has disappeared and `status.state == Available`; `StateUnavailable` is treated as upgrade failure.

---

#### saola operator delete

**Syntax**: `saola operator delete <name> [flags]`

**Description**: Delete a MiddlewareOperator by name.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | MiddlewareOperator resource name |
| `--namespace` | `-n` | string | empty (required) | Target namespace |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for deletion to complete (Finalizer cleanup may take time) |

**Examples**:
```bash
saola operator delete redis-operator --namespace my-ns
saola operator delete redis-operator -n my-ns --wait 2m
```

**Notes**:
- Silently skips if the resource does not exist.
- `--wait` polls every 2 seconds to check whether the object has been completely removed from the cluster.

---

### 3.4 Baseline Management (saola baseline)

Baseline commands are accessed via `saola baseline` (alias `bl`).

---

#### saola baseline get

**Syntax**: `saola baseline get <name> [flags]`

**Description**: Get a single MiddlewareBaseline or MiddlewareOperatorBaseline by name from installed middleware packages.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | Baseline name |
| `--package` | - | string | required | Package name to query |
| `--kind` | - | string | `middleware` | Baseline type: `middleware`, `operator` |
| `--output` | `-o` | string | `yaml` | Output format: `table`, `yaml`, `json` |

**Examples**:
```bash
saola baseline get default --package redis-v1
saola baseline get default --package redis-v1 --kind operator -o yaml
```

---

#### saola baseline list

**Syntax**: `saola baseline list [flags]` (alias: `saola baseline ls`)

**Description**: List all MiddlewareBaseline, MiddlewareOperatorBaseline, or MiddlewareActionBaseline in a specified installed package.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--package` | - | string | required | Package name to query |
| `--kind` | - | string | `middleware` | Baseline type: `middleware`, `operator`, `action` |
| `--output` | `-o` | string | `table` | Output format: `table`, `yaml`, `json` |

**Examples**:
```bash
saola baseline list --package redis-v1
saola baseline list --package redis-v1 --kind middleware
saola baseline list --package redis-v1 --kind operator
saola baseline list --package redis-v1 --kind action
```

**Notes**:
- Table output columns (middleware/operator): `NAME`, `OPERATOR`, `CONFIGURATIONS`, `PREACTIONS`.
- Table output columns (action): `NAME`.
- When no results are found, outputs "No xxx baselines found."

---

### 3.5 Action Management (saola action)

Action commands are accessed via `saola action` (alias `act`).

---

#### saola action get

**Syntax**: `saola action get [name] [flags]`

**Description**: List all MiddlewareAction resources in a namespace, or get a single resource by name.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `[name]` | - | positional | optional | If specified, get a single resource |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--all-namespaces` | `-A` | bool | `false` | List across all namespaces |
| `--output` | `-o` | string | `table` | Output format: `table`, `yaml`, `json` |

**Examples**:
```bash
saola action get
saola action get my-action-1234567890
saola action get -A
saola action get my-action-1234567890 -o yaml
```

**Notes**:
- Table output columns: `NAME`, `MIDDLEWARE`, `BASELINE`, `STATE`, `AGE`.
- When using `-A`, an additional `NAMESPACE` column is shown.

---

#### saola action describe

**Syntax**: `saola action describe <name> [flags]`

**Description**: Get a MiddlewareAction by name and display its full status and event information in a human-readable format.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<name>` | - | positional | required | MiddlewareAction name |
| `--namespace` | `-n` | string | empty | Target namespace |

**Examples**:
```bash
saola action describe my-action-1234567890
saola action describe my-action-1234567890 -n my-namespace
```

**Output content**:
- Metadata: Name, Namespace, Created
- Spec: Middleware, Baseline, Necessary (formatted JSON)
- Status: State, Reason, ObservedGeneration
- Conditions: TYPE, STATUS, REASON, MESSAGE

---

#### saola action run

**Syntax**: `saola action run [flags]`

**Description**: Create a MiddlewareAction CR to trigger a one-off operation on a specified Middleware instance. The Action name is automatically generated as `<baseline>-<unix-timestamp>` to avoid conflicts.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--middleware` | - | string | required | Associated Middleware instance name |
| `--baseline` | - | string | required | MiddlewareActionBaseline name |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--params` | - | string array | empty | Operation parameters in `key=value` format, comma-separated or repeated |
| `--wait` | - | duration | `0` (no wait) | Timeout for waiting for the operation to complete |

**Examples**:
```bash
saola action run --middleware my-redis --baseline redis-backup
saola action run --middleware my-redis --baseline redis-restore --params src=backup-001 --wait 5m
```

**Notes**:
- `--params` supports two styles:
  - Multiple specifications: `--params key1=val1 --params key2=val2`
  - Comma-separated: `--params k1=v1,k2=v2`
- Parameters are serialized as JSON into the `spec.necessary` field.
- `--wait` polls every 2 seconds to check `status.state`: `Available` means success, `Unavailable` means failure.

---

### 3.6 Shortcut Commands

Shortcut commands are top-level verb-style kubectl-like commands that provide a unified entry point across resource types.

---

#### saola build

`saola build` is equivalent to `saola package build`.

**Syntax**: `saola build <pkg-dir> [--output/-o <path>]`

---

#### saola install

`saola install` is equivalent to `saola package install`.

**Syntax**: `saola install <pkg-dir> [--name <name>] [--wait <duration>] [--dry-run]`

---

#### saola uninstall

`saola uninstall` is equivalent to `saola package uninstall`.

**Syntax**: `saola uninstall <name> [--wait <duration>]`

---

#### saola upgrade

`saola upgrade` is a multi-purpose upgrade command supporting both package upgrades and instance upgrades:

- **Package upgrade**: `saola upgrade <pkg-dir> [--name <name>] [--wait <duration>]`
  - Equivalent to `saola package upgrade`
- **Instance upgrade**: Routed via subcommands
  - `saola upgrade middleware <name> --to-version <v> [--baseline <b>] [-n <ns>] [--wait <d>]`
  - `saola upgrade operator <name> --to-version <v> [--baseline <b>] [-n <ns>] [--wait <d>]`
  - Aliases: `mw`, `op`

When the first argument is not a known subcommand (`middleware`/`mw`/`operator`/`op`), it automatically takes the package upgrade path.

---

#### saola delete

`saola delete` routes to the deletion logic of each resource type via resource type subcommands.

**Syntax**: `saola delete <resource-type> <name> [flags]`

**Supported resource types**:

| Type | Alias | Description |
|------|-------|-------------|
| `middleware` | `mw` | Delete Middleware resource |
| `operator` | `op` | Delete MiddlewareOperator resource |

**Additional Flags**:
- `--dry-run`: Only print resources to be deleted without actually executing
- `--namespace` / `-n`: Target namespace
- `--wait`: Timeout for waiting for deletion to complete

**Examples**:
```bash
saola delete middleware my-redis
saola delete mw my-redis -n production
saola delete operator redis-operator -n my-ns --wait 2m
saola delete middleware my-redis --dry-run
```

---

#### saola get

`saola get` is the unified resource query entry point, routing via resource type subcommands.

**Syntax**: `saola get <resource-type> [name] [flags]`

**Supported resource types**:

| Type | Alias | Description |
|------|-------|-------------|
| `middleware` | `mw` | Middleware resources |
| `operator` | `op` | MiddlewareOperator resources |
| `action` | `act` | MiddlewareAction resources |
| `baseline` | `bl` | Baseline resources (requires `--package`) |
| `package` | `pkg` | Installed middleware packages |
| `all` | - | Aggregate output of middleware, operator, and action |

**Examples**:
```bash
saola get middleware
saola get mw my-redis -o yaml
saola get operator -A
saola get action -n my-ns
saola get baseline --package redis-v1 --kind operator
saola get package --component redis
saola get package redis-v1
saola get all -n my-ns
```

**Notes**:
- `saola get package [name]`: With a name argument, equivalent to `inspect`; without arguments, equivalent to `list`.
- `saola get baseline [name]`: With a name argument, equivalent to `baseline get`; without arguments, equivalent to `baseline list`.
- `saola get all` outputs Middlewares, Operators, and Actions in order, with separator headers between each type.

---

#### saola describe

`saola describe` displays resource details via resource type subcommands.

**Syntax**: `saola describe <resource-type> <name> [flags]`

**Supported resource types**:

| Type | Alias |
|------|-------|
| `middleware` | `mw` |
| `operator` | `op` |
| `action` | `act` |

**Examples**:
```bash
saola describe middleware my-redis
saola describe mw my-redis -n production
saola describe operator redis-operator -n my-ns
saola describe action my-action-1234567890
```

---

#### saola inspect

`saola inspect` is equivalent to `saola package inspect`.

**Syntax**: `saola inspect <name> [-o table|yaml|json]`

---

#### saola create (with Interactive Mode)

`saola create` supports two modes:

- **File mode**: `saola create -f <file> [-n <namespace>] [--dry-run]`
  - Automatically detects the `kind` field in YAML and routes to Middleware or MiddlewareOperator creation logic
  - Supported kinds: `Middleware`, `MiddlewareOperator`
- **Interactive mode**: `saola create` (without `-f`)
  - Enters the terminal-guided creation flow (see Chapter 7 for details)

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--file` | `-f` | string | empty | YAML file path; enters interactive mode when empty |
| `--namespace` | `-n` | string | empty | Override namespace in manifest |
| `--dry-run` | - | bool | `false` | Only print the resource to be created (file mode only) |

**Examples**:
```bash
# Interactive creation
saola create

# Create from file
saola create -f middleware.yaml
saola create -f operator.yaml -n production
saola create -f middleware.yaml --dry-run
```

---

#### saola run

`saola run` is the top-level shortcut for `saola action run`, with the baseline name passed as a positional argument.

**Syntax**: `saola run <baseline> --middleware <name> [flags]`

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `<baseline>` | - | positional | required | MiddlewareActionBaseline name |
| `--middleware` | - | string | required | Associated Middleware instance name |
| `--namespace` | `-n` | string | empty | Target namespace |
| `--params` | - | string array | empty | Operation parameters in `key=value` format |
| `--wait` | - | duration | `0` | Wait for operation to complete |

**Examples**:
```bash
saola run redis-backup --middleware my-redis
saola run redis-restore --middleware my-redis --params src=backup-001 --wait 5m
```

---

#### saola resource

`saola resource` is not a user command but an internal resource type registry module that provides alias resolution.

Supported alias mappings:

| Alias | Canonical Name |
|-------|---------------|
| `mw` | `middleware` |
| `op` | `operator` |
| `act` | `action` |
| `bl` | `baseline` |
| `pkg` | `package` |

---

#### saola version

**Syntax**: `saola version [-o json|yaml]`

**Description**: Print saola-cli's version number, build time, and Git commit information.

**Arguments/Flags**:

| Name | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | empty (human-readable format) | Output format: `json`, `yaml` |

**Examples**:
```bash
saola version
saola version -o json
```

**Output example** (default format):
```
Version: v1.0.0
Git Commit: abc1234
Build Date: 2026-01-01T00:00:00Z
```

---

## 4. Package Specification

### 4.1 TAR Package Format and Directory Structure

saola-cli builds local directories into zstd-compressed TAR archives. The root directory prefix inside the TAR is `<name>-<version>/`, allowing OpenSaola's `ReadTarInfo` to correctly strip the first-level path.

**Expected directory structure**:

```
<pkg-dir>/
  metadata.yaml          # Package metadata (required)
  baselines/             # Middleware/Operator/Action Baseline definitions
  configurations/        # Configuration templates
  actions/               # Action templates
  crds/                  # CRD definitions
  manifests/             # Platform frontend configuration files
```

The `manifests/` directory contains configuration files used by the platform frontend (e.g., parameter form definitions `parameters.yaml`, internationalization `i18n.yaml`, version rules `middlewareversionrule.yaml`, hidden menus `hiddenmenus.yaml`). This directory's contents are not directly processed by OpenSaola but consumed by platform frontend components.

**Packaging rules**:
- Hidden files and directories (starting with `.`, e.g., `.git`, `.DS_Store`) are automatically skipped.
- Only regular files are packaged; directories themselves are not stored in the TAR.
- Path separators are normalized to `/` (cross-platform compatibility).
- Uses `packages.Compress()` for zstd compression.

### 4.2 metadata.yaml Requirements

```yaml
name: redis                      # Required: package name
version: "1.0.0"                 # Required: package version
app:
  version: ["7.2.0", "7.2.1"]   # Supported application version list
  deprecatedVersion: []          # Deprecated version list
owner: "team-a"                  # Package owner
type: "cache"                    # Package type (valid values: db / mq / cache / search / storage)
description: "Redis package"     # Package description
```

**Required fields**: `name`, `version`. `PackDir()` returns an error if they are missing.

### 4.3 Package State Machine

Packages manage state through Labels and Annotations on the Secret:

```
[Install Request]
    |
    v
Secret created (enabled=false, install=true)
    |
    v (OpenSaola processes)
Install success -> enabled=true, install annotation removed
Install failure -> enabled=false, installError annotation set
    |
[Uninstall Request]
    |
    v
Add uninstall annotation
    |
    v (OpenSaola processes)
Uninstall complete -> enabled=false, uninstall annotation removed
      or -> Secret directly deleted
```

---

## 5. Secret Structure

### 5.1 Secret Generated After Package Installation

The Secret created by `saola install` / `saola package install` has the following structure:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <name>-<version>         # Default, can be overridden with --name
  namespace: middleware-operator  # pkg-namespace
  labels:
    middleware.cn/project: OpenSaola
    middleware.cn/component: <metadata.name>
    middleware.cn/packageversion: <metadata.version>
    middleware.cn/packagename: <secret-name>
    middleware.cn/enabled: "false"
  annotations:
    middleware.cn/install: "true"
immutable: true
data:
  package: <zstd-compressed-tar-bytes>
```

### 5.2 Secret Labels and Annotations

**Labels**:

| Label Key | Value | Description |
|-----------|-------|-------------|
| `middleware.cn/project` | `OpenSaola` | Identifies the Secret as belonging to the OpenSaola system |
| `middleware.cn/component` | `<metadata.name>` | Middleware component name (e.g., `redis`) |
| `middleware.cn/packageversion` | `<metadata.version>` | Package version number |
| `middleware.cn/packagename` | `<secret-name>` | Package Secret name |
| `middleware.cn/enabled` | `"true"` / `"false"` | Whether the package has been successfully installed and enabled |

**Annotations**:

| Annotation Key | Value | Description |
|----------------|-------|-------------|
| `middleware.cn/install` | `"true"` | Triggers installation |
| `middleware.cn/uninstall` | `"true"` | Triggers uninstallation |
| `middleware.cn/installError` | error message | Set by operator on installation failure |

### 5.3 Secret data Fields

| Key | Description |
|-----|-------------|
| `package` (`packages.Release` constant, value is `"package"`) | zstd-compressed TAR archive byte content |

The Secret is marked as `immutable: true`, so upgrades require deletion and recreation.

---

## 6. Complete Workflow Examples

### 6.1 Complete Flow from Scratch

```bash
# 1. Build package (optional for offline scenarios)
saola build ./my-redis-pkg --output ./dist/redis-1.0.0.pkg

# 2. Install package to cluster
saola install ./my-redis-pkg --wait 5m

# 3. View installed packages
saola get package

# 4. View baselines in the package
saola get baseline --package redis-1.0.0

# 5. Create MiddlewareOperator (if needed)
saola create -f operator.yaml -n my-ns

# 6. Create Middleware instance
saola create -f middleware.yaml -n my-ns
# Or use interactive mode
saola create

# 7. View resource status
saola get all -n my-ns

# 8. View details
saola describe middleware my-redis -n my-ns
```

### 6.2 Package Lifecycle

```bash
# Install
saola install ./redis-pkg --wait 5m

# View
saola get package
saola inspect redis-1.0.0

# Upgrade
saola upgrade ./redis-pkg-v2 --wait 5m

# Uninstall
saola uninstall redis-1.0.0 --wait 5m
```

### 6.3 Middleware Instance Lifecycle

```bash
# Create
saola create -f middleware.yaml -n production

# View status
saola get middleware -n production
saola describe middleware my-redis -n production

# Version upgrade
saola upgrade middleware my-redis --to-version 2.0.0 -n production --wait 5m

# Execute operations
saola run redis-backup --middleware my-redis -n production --wait 5m

# Delete
saola delete middleware my-redis -n production --wait 2m
```

### 6.4 Operator Lifecycle

```bash
# Create
saola create -f operator.yaml -n production

# View status
saola get operator -n production
saola describe operator redis-operator -n production

# Upgrade
saola upgrade operator redis-operator --to-version 2.0.0 -n production --wait 5m

# Delete
saola delete operator redis-operator -n production --wait 2m
```

---

## 7. Interactive Creation Details

### 7.1 Interactive Flow

Run `saola create` (without the `-f` argument) to enter interactive creation mode, which uses the charmbracelet/huh library for terminal forms.

**Middleware creation flow**:

1. **Select resource type**: Middleware or MiddlewareOperator
2. **Select middleware component**: If installed packages contain multiple components (e.g., Redis, PostgreSQL), select the component type first
3. **Select Baseline**: Choose a specific MiddlewareBaseline under the selected component
4. **Fill in metadata**: Enter instance name and namespace
5. **Fill in necessary parameters**: Dynamically generated form based on the baseline's `spec.necessary` JSON Schema
6. **YAML preview**: Display the complete Middleware YAML to be created
7. **Confirm creation**: Create the resource after confirmation

**MiddlewareOperator creation flow**:

1. **Select resource type**: Choose MiddlewareOperator
2. **Select middleware component**: Same as above
3. **Select OperatorBaseline**: Choose a MiddlewareOperatorBaseline under the selected component
4. **Fill in metadata**: Enter instance name and namespace
5. **Configure Globe**: If the baseline contains `spec.globe` (image registry configuration), display editable input fields for each property, pre-filled with baseline defaults; only writes to `mo.spec.globe` when the user modifies at least one value
6. **YAML preview**: Display the complete MiddlewareOperator YAML
7. **Confirm creation**

### 7.2 JSON Schema Support

Interactive mode parses JSON Schema from the baseline's `spec.necessary` field and supports the following field types:

| Type | Form Component | Validation Rules |
|------|---------------|-----------------|
| `string` | Text input | `required` validation, `pattern` regex validation |
| `int` | Text input | Integer validation, `min`/`max` range validation |
| `password` | Password input (hidden) | `required` validation, multiple `patterns` regex validation |
| `enum` | Dropdown select | Options from `options` field (comma-separated) |
| `version` | Dropdown select / Text input | Options from package metadata's `app.version` list; falls back to text input when no list is available |
| `storageClass` | Dropdown select / Text input | Options from cluster's StorageClass list; falls back to text input when unavailable |

**FieldSchema structure**:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Field type (required, skipped if empty) |
| `label` | string | Human-readable display name |
| `required` | bool | Whether the field is required |
| `default` | interface{} | Default value |
| `description` | string | Supplementary description |
| `placeholder` | string | Hint text for empty input |
| `pattern` | string | Regex validation (used by string type) |
| `min` | float64 | Numeric lower bound (used by int type) |
| `max` | float64 | Numeric upper bound (used by int type) |
| `options` | string | Comma-separated list of allowed values (used by enum type) |
| `patterns` | []PatternRule | Multiple validation rules (used by password type) |

### 7.3 Parameter Validation

- **Required validation**: Fields with `required=true` do not accept blank input.
- **Regex validation**: Regular expressions defined in `pattern` or `patterns` are used for input validation; description is displayed when a match fails.
- **Range validation**: `int` type supports `min` and `max`; prompts with minimum/maximum values when out of range.
- **Enum validation**: `enum` type only allows selection from `options`.

**Value collection and nesting**:

User input values are collected as a flat map keyed by dot-separated paths (e.g., `resource.postgresql.limits.cpu`), then converted to a nested structure via `BuildNecessaryValues()`:

```
{"resource.postgresql.limits.cpu": "1"}
  -->
{"resource": {"postgresql": {"limits": {"cpu": "1"}}}}
```

---

## 8. Output Formats

### 8.1 Table Format

Default output format. Uses `text/tabwriter` to implement aligned tables. Supports three input data types:

- `[][]string`: First row is headers, subsequent rows are data
- `[]map[string]string`: Keys serve as column names
- Any struct slice: Exported field names serve as column names (uppercase), values formatted via `fmt.Sprint`

`wide` format is a variant of `table` where the caller passes row structures containing additional fields (such as LABELS).

### 8.2 JSON Format

Uses `encoding/json`'s `Encoder` with 2-space indentation.

### 8.3 YAML Format

Uses `gopkg.in/yaml.v3`'s `Encoder` with 2-space indentation.

### 8.4 Name Format

Outputs `type/name` format, one per line. Similar to `kubectl get -o name`.

- Implemented via NamePrinter, which requires the caller to set the `ResourceType` field.
- Supports extracting `Name` or `NAME` fields from structs via reflection, and also supports embedded `ObjectMeta.Name`.

---

## 9. Configuration and Environment

### 9.1 Configuration File

saola-cli currently does not use a standalone configuration file. All configuration is passed via CLI flags and environment variables.

The `Config` struct contains the following fields:

| Field | Source (highest to lowest priority) | Default |
|-------|-------------------------------------|---------|
| `Kubeconfig` | `--kubeconfig` -> `$KUBECONFIG` | empty (uses `~/.kube/config`) |
| `Context` | `--context` | empty (uses kubeconfig default context) |
| `Namespace` | `--namespace` -> `$SAOLA_NAMESPACE` | empty (command layer falls back to `default`) |
| `PkgNamespace` | `--pkg-namespace` -> `$SAOLA_PKG_NAMESPACE` | `middleware-operator` |
| `LogLevel` | `--log-level` | `info` |
| `NoColor` | `--no-color` | `false` |

### 9.2 Environment Variables

See [2.5 Environment Variables](#25-environment-variables).

---

## 10. Data Flow with Other Projects

### 10.1 saola-cli --> dataservice-baseline (Consuming Packages)

saola-cli's `build` / `install` commands package the local directories produced by the dataservice-baseline project build into TAR archives. The package directory contains:

- `metadata.yaml`: Package metadata
- `baselines/`: YAML/JSON definitions for MiddlewareBaseline, MiddlewareOperatorBaseline, MiddlewareActionBaseline
- `configurations/`: CUE/Go configuration templates
- `actions/`: Action execution templates
- `crds/`: CRD definition files

### 10.2 saola-cli --> OpenSaola (Creating K8s Resources)

saola-cli creates the following resources in the Kubernetes cluster, consumed by OpenSaola:

| Resource Type | Creation Method | OpenSaola Behavior |
|--------------|-----------------|-------------------|
| Secret (package) | `saola install` / `saola upgrade` | Watches `install` annotation, decompresses TAR, loads baseline/config/action templates |
| Middleware CR | `saola create` / `saola middleware create` | Reconcile loop: find package -> render templates -> deploy underlying resources -> report status |
| MiddlewareOperator CR | `saola create` / `saola operator create` | Reconcile loop: deploy operator deployment -> readiness check -> report status |
| MiddlewareAction CR | `saola run` / `saola action run` | Reconcile loop: execute one-off operation -> report status |

### 10.3 Complete Data Flow Diagram

```
+------------------------+
| dataservice-baseline   |
| (CUE/Go templates,    |
|  Baseline definitions) |
+----------+-------------+
           |
           | Build artifacts
           v
+----------+-------------+
| Local package dir      |
| metadata.yaml          |
| baselines/             |
| configurations/        |
| actions/               |
| crds/                  |
+----------+-------------+
           |
           | saola build / install
           v
+----------+-------------+
| K8s Secret             |
| (zstd TAR in data)     |  <--- pkg-namespace (middleware-operator)
| Labels + Annotations   |
+----------+-------------+
           |
           | OpenSaola Watch
           v
+----------+-------------+
| OpenSaola              |
| Package Service        |
| (decompress, parse,    |
|  cache)                |
+----------+-------------+
           |
           | saola create / middleware create / operator create
           v
+----------+---+---+-----+
| Middleware   | MiddlewareOperator  | MiddlewareAction  |
| CR           | CR                  | CR                |
+--------------+-----+---------+----+-------------------+
                     |         |        |
                     v         v        v
              OpenSaola Reconcile Controllers
              (deploy underlying resources, state management, upgrades, deletion)
```
