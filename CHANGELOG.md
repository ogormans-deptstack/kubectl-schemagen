# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v0.4.0

Released: 2026-05-23

Major performance optimization and schema fidelity improvements. Addresses the
performance gap noted by @ahmetb during krew review.

PERFORMANCE:

- **~12x faster schema loading** for single-resource generation. New `FetchForResource` fetches only the group-version schema containing the requested type instead of downloading all 30-50+ schemas from the cluster. Well-known resource types (Deployment, Pod, Service, etc.) resolve with zero extra HTTP calls via a static path lookup. CRDs use a concurrent fallback scan.
- Concurrent `FetchAll` and `ListGVKs` fetch group-version schemas in parallel using goroutines, reducing wall-clock time for `--list` and multi-resource scaffold.
- ~12x reduction in memory allocation (38MB -> 3MB for single-resource generation).
- Scaffold uses targeted fetching for single resource type.

ENHANCEMENTS:

- **Format-aware string placeholders**: CRD string fields with `format` now generate appropriate placeholder values (email, hostname, ipv4, ipv6, uri, uuid, byte, password, duration, date) instead of the generic "example".
- **Schema example/default priority for CRDs**: CRD-authored `example` and `default` values in the OpenAPI schema now take priority over hardcoded field defaults. Built-in K8s types retain current priority order.
- **Constraint-aware numeric values**: Integer and number fields respect `minimum`, `maximum`, `exclusiveMinimum`, and `multipleOf` constraints from the schema.
- **JSON output** (`manifest --output json` / `-o json`): Generate manifests as indented JSON for programmatic consumption and piping to `jq`.
- **Migrate stdin support**: Read manifests from stdin using `-` as filename (`kubectl get deploy -o yaml | kubectl schemagen migrate -`).
- **Migrate exit codes for CI**: Exit 0 = all OK, exit 1 = removed APIs, exit 2 = deprecated APIs (none removed). Enables CI pipelines to fail on removed APIs while allowing deprecated ones.
- Add `GenerateJSON` to `ResourceGenerator` interface.

TESTING:

- Add benchmarks comparing merged vs targeted schema fetch performance.
- Add `TestSingleDocProducesSameOutput` verifying single-GV generation produces identical output to full merged document.
- Add tests for format-aware placeholders, CRD example/default priority, constraint-aware numerics, and JSON output.

## v0.3.0

Released: 2026-05-23

Restructure as kubectl-schemagen with three subcommands.

ENHANCEMENTS:

- Restructure CLI as `kubectl-schemagen` with `manifest`, `migrate`, and `scaffold` subcommands.
- Add `migrate` subcommand: detect deprecated/removed Kubernetes API versions in YAML manifests.
- Add `scaffold` subcommand: generate kustomize base directories from resource types.
- Fix `--replicas` on non-workload types, `--set` metadata path overrides.
- Add e2e tests for all subcommands including cross-subcommand round-trips.

## v0.2.1

BUG FIXES:

- Fix `--set` overrides silently overwritten by post-processors ([#3](https://github.com/ogormans-deptstack/kubectl-schemagen/issues/3)). `applyOverrides()` now runs after all post-processors, and supports dot-path keys (e.g. `--set spec.template.spec.restartPolicy=OnFailure`) and array indexing (e.g. `--set containers[0].image=nginx`).

ENHANCEMENTS:

- Add fuzzy matching for resource type suggestions ([#2](https://github.com/ogormans-deptstack/kubectl-schemagen/issues/2)). Typos now suggest the closest match using Levenshtein distance (e.g. `Deploymnet` suggests `Deployment`).

## v0.2.0

Released: 2026-04-15

This release renames the project from `kubectl-example` to `kubectl-generate`.

ENHANCEMENTS:

- Rename `kubectl-example` to `kubectl-generate` across all binaries, module path, and documentation
- Expand native resource coverage from 13 to 30 types (RBAC, storage, scheduling, admission, CRDs)
- Add CRD support for Gateway API (10 types), Argo Workflows (4 types), cert-manager (3 types), Crossplane (3 types)
- Add GoReleaser config and krew manifest for distribution
- Add GitHub infrastructure: issue templates, PR template, CODEOWNERS, branch protection via OpenTofu
- Strip mutually exclusive issuer types from cert-manager Issuer/ClusterIssuer
- Strip mutually exclusive Argo template types, fix CronWorkflow schedule field
- Fix krew template indentation for addURIAndSha output

BUG FIXES:

- Fix Argo CRD install in CI (use full CRDs from upstream manifests)
- Fix Gateway API CI (use experimental-install.yaml for full type coverage)
- Fix CronWorkflow validation (use `schedules` plural field, not `schedule`)

## v0.1.0

Released: 2026-04-14

Initial release.

ENHANCEMENTS:

- OpenAPI v3 schema-driven YAML generation from live cluster
- 13 core resource types with server-side dry-run validation
- CRD support (CronTab custom resource)
- Smart field selection: required fields always included, optional fields via important-fields registry
- Sensible defaults: nginx:latest images, RollingUpdate strategy, label/selector wiring
- Override flags: `--name`, `--image`, `--replicas`, `--set key=value`
- Dynamic flag generation from schema introspection
- `--list` to enumerate all available resource types
- CI pipeline with unit tests, lint, and e2e against kind cluster
