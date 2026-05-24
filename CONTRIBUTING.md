# Contributing to kubectl-schemagen

Thank you for your interest in contributing to kubectl-schemagen. This document covers the process for contributing to this project.

## Getting Started

### Prerequisites

- Go 1.25 or later
- A Kubernetes cluster (for e2e tests, [kind](https://kind.sigs.k8s.io/) works well)
- [golangci-lint](https://golangci-lint.run/) v2.11+

### Building

```bash
git clone https://github.com/ogormans-deptstack/kubectl-schemagen.git
cd kubectl-schemagen
make build
```

### Running Tests

```bash
# Unit tests
make test-unit

# Lint
make lint

# e2e tests (requires a kind cluster with CRDs installed)
kind create cluster --name demo
kubectl apply -f test/fixtures/crontab-crd.yaml
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/experimental-install.yaml
make test-e2e
```

## How to Contribute

### Reporting Issues

- Check [existing issues](https://github.com/ogormans-deptstack/kubectl-schemagen/issues) before opening a new one
- Use the issue templates (bug report, feature request, CRD request)
- Include the output of `kubectl schemagen --version` and your Kubernetes version

### Submitting Changes

1. Open an issue first to discuss the approach, especially for larger changes
2. Fork the repository and create a branch from `main`
3. Write tests for new functionality -- the project uses table-driven Go tests
4. Run `make lint` and `make test-unit` before submitting
5. Submit a pull request referencing the related issue

### Pull Request Guidelines

- Keep changes focused -- one concern per PR
- Include tests that cover the new behavior
- Update documentation if the change affects user-facing behavior
- PRs require passing CI (lint, unit tests, e2e) before merge

### Commit Messages

Write clear commit messages that explain the why, not just the what. Use the imperative mood:

```
Fix CronWorkflow schedule field to use plural form

The Argo CRD schema uses `schedules` (plural) not `schedule`.
The singular form passes client-side but fails server-side validation.
```

### Testing Conventions

- Table-driven tests with descriptive test case names
- e2e tests validate generated YAML via `kubectl apply --dry-run=server`
- New resource types should include both unit and e2e test coverage

## Code of Conduct

This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
