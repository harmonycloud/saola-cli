# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
