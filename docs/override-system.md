# Override System

This document describes how `--set` overrides interact with the post-processor pipeline in kubectl-schemagen.

## Override Priority

The generator applies changes in this order:

1. **Schema walk** -- generates the base manifest from OpenAPI v3
2. **Post-processors** -- fix up labels, selectors, strategies, and resource-specific defaults
3. **Overrides** -- `--name`, `--image`, `--replicas`, then `--set key=value`

Because overrides run last, `--set` always wins over any post-processor.

## Dot-Path Syntax

The `--set` flag supports dot-separated paths for nested fields:

```bash
# Set a deeply nested field
kubectl schemagen manifest Job --set spec.template.spec.restartPolicy=OnFailure

# Array indexing
kubectl schemagen manifest Deployment --set containers[0].image=myapp:v2

# The leading spec. prefix is optional (we're already inside spec)
kubectl schemagen manifest Deployment --set template.spec.containers[0].image=myapp:v2
```

## Post-Processor Reference

The following post-processors run on generated manifests. Each one targets specific resource types and fields.

| Post-Processor | Applies To | What It Does |
|----------------|-----------|--------------|
| `injectTemplateLabels` | Deployment, StatefulSet, DaemonSet, Job, CronJob | Sets `metadata.labels` and `selector.matchLabels` on pod templates |
| `injectTemplateRestartPolicy` | Job, CronJob | Sets `restartPolicy: Never` on pod specs |
| `fixStrategyDefaults` | Deployment, StatefulSet, DaemonSet | Sets `strategy.type` or `updateStrategy.type` to `RollingUpdate` |
| `injectServiceSelector` | Service | Adds `selector` with app label (only if selector is absent) |
| `fixCRDDefaults` | CustomResourceDefinition | Removes `conversion`, auto-generates valid CRD name from plural + group |
| `fixPDBDefaults` | PodDisruptionBudget | Removes `maxUnavailable` when `minAvailable` is present (mutually exclusive) |
| `fixPVDefaults` | PersistentVolume | Removes all 21 volume source types (mutually exclusive; hostPath is set separately) |
| `fixIngressClassDefaults` | IngressClass | Removes `parameters` (generates invalid values from schema) |
| `fixIssuerDefaults` | Issuer, ClusterIssuer | Keeps only `acme` issuer type, removes `ca`, `vault`, `venafi` (mutually exclusive) |
| `fixArgoDefaults` | Workflow, CronWorkflow, WorkflowTemplate, ClusterWorkflowTemplate | Keeps only `container` template type, fixes Prometheus metric names |
| `fixLimitRangeDefaults` | LimitRange | Sets sensible `default`, `defaultRequest`, `min`, `max` values |
| `stripNoisyFields` | Pod, Deployment, StatefulSet, DaemonSet, Job, ReplicaSet, CronJob | Removes `tolerations`, `topologySpreadConstraints`, `overhead`, `readinessGates` |

## Bug Fix: Override Ordering (v0.2.1)

Prior to v0.2.1, `applyOverrides()` ran before post-processors. This meant post-processors could silently overwrite user-provided `--set` values. For example:

```bash
# Before v0.2.1: restartPolicy was set to OnFailure, then overwritten back to Never
kubectl schemagen manifest Job --set spec.template.spec.restartPolicy=OnFailure
```

Fixed by moving `applyOverrides()` to run after all post-processors. See [#3](https://github.com/ogormans-deptstack/kubectl-schemagen/issues/3).

## Limitations

- `--set` cannot create intermediate maps that don't exist. If a post-processor deletes a field (e.g. `conversion` on CRDs), `--set spec.conversion.strategy=Webhook` won't recreate the nested structure.
- Complex values (arrays, objects) cannot be set via `--set`. Use it for scalar values only.
- The `name` key is reserved for `metadata.name` and cannot be used as a spec-level override via `--set name=X`. Use `--name` instead.
