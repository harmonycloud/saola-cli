# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-10-24

### Added
- `saola images export --format docker-multi` creates a root OCI layout for multi-platform Docker loading, with `--platforms` defaulting to `linux/amd64,linux/arm64`.
- Multi-platform exports check that every image has all required architectures before downloading layers and try subsequent candidate repositories when a repository lacks a required platform.
- English and Chinese examples for offline package delivery, image export/import, local validation, and the public command surface.

### Fixed
- Middleware creation checks for a MiddlewareOperator whose `spec.baseline` matches the referenced OperatorBaseline, avoiding false missing-operator warnings when the instance and baseline names differ.
- Check archive close errors in image tests so release candidates pass the configured lint checks.

### Compatibility
- Multi-platform export requires `skopeo`; loading the resulting archive requires Docker's containerd image store. Use `--format docker --platform <platform>` for older single-platform Docker environments.
- The experimental SSA migration commands were removed before release. This release does not expose `migrate ssa`; OpenSaola establishes resource-intent records during normal updates without a bulk CLI migration.

## [1.0.0] - 2026-07-13

### Added
- First stable release of the CLI, including middleware/package lifecycle commands and image discovery, export, and import.
- Reproducible Linux AMD64 and ARM64 binaries, SHA-256 checksums, SBOMs, and signed release artifacts.

## [0.1.0] - 2025-01-01

### Added
- kubectl-style CLI with verb + resource commands (get/create/delete/describe/upgrade/run)
- Interactive TUI creation for Middleware and MiddlewareOperator with JSON Schema support
- Bilingual support (--lang zh|en) for all commands and help text
- Package management: build, install, upgrade, uninstall, inspect
- Multiple output formats: table, yaml, json, wide, name
- Middleware and Operator version upgrade with --wait support
- MiddlewareAction triggering via run command
- Baseline and package content inspection
- Apache License 2.0
- GitHub Actions CI (lint, test, build)
- Comprehensive usage documentation (Chinese + English)

### Fixed
- Module path migrated from internal gitea to gitee.com/opensaola
- All zeus-operator references renamed to opensaola
- Kubeconfig credentials removed from repository
- Binary removed from git tracking
- E2E test report sanitized (internal IPs removed)
