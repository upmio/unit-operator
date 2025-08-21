## Unit Operator Helm Chart

This chart deploys the Unit Operator to Kubernetes (v1.27+) and OpenShift (v4.6+) clusters.

### TL;DR

```sh
# Add the Helm repository (prefer released versions over main)
helm repo add unit-operator https://upmio.github.io/unit-operator
helm repo update

# Install the operator (install CRDs together as an example)
helm install unit-operator unit-operator/unit-operator \
  --namespace upm-system \
  --create-namespace \
  --set crds.enabled=true
```

### Prerequisites

- Kubernetes ≥ 1.27 or OpenShift ≥ 4.6
- `helm` v3 installed and available
- Permissions to create CRDs, RBAC, and ServiceAccounts in the target namespace

### Overview

By default, the Unit Operator is installed into the `upm-system` namespace.

For Custom Resource Definitions (CRDs), you can install them manually (recommended for production) or enable installation via this chart:

- Install CRDs manually: uninstalling the chart will not affect the CRDs and their CR instances
- Install CRDs via this chart (`crds.enabled=true`): uninstalling the chart will also remove the CRDs and thus delete all related CRs; use with care

CRDs and documentation:

- `Unit`: see repository documentation at `doc/unit-operator_api.md#unit`
- `UnitSet`: see repository documentation at `doc/unit-operator_api.md#unitset`

Once the Unit Operator is installed, you can apply the corresponding Custom Resources (CRs) to your cluster. The operator will then create and manage the required resources accordingly.

### Install

```sh
helm install unit-operator unit-operator/unit-operator \
  --namespace upm-system \
  --create-namespace
```

Common options:

- **Install CRDs together**: `--set crds.enabled=true`
- **Customize image**: `--set image.registry=...,image.repository=...,image.tag=...`

### Upgrade

```sh
helm repo update
helm upgrade unit-operator unit-operator/unit-operator \
  --namespace upm-system \
  --install \
  --reuse-values
```

To change configuration, pass values via `--set` or `-f values.yaml`.

### Uninstall

Before uninstalling, it is recommended to ensure there are no operator-managed resources left:

```sh
kubectl get units,unitsets --all-namespaces
```

Uninstall with Helm:

```sh
helm uninstall unit-operator --namespace upm-system --wait

# Optionally remove the repo
helm repo remove unit-operator
```

Important: If CRDs were installed via this chart (`crds.enabled=true`), uninstalling will remove the CRDs as well, which deletes all related CRs.

If you installed CRDs manually in production and need to remove them (dangerous; will delete all related CRs):

```sh
kubectl delete crd units.upm.syntropycloud.io,unitsets.upm.syntropycloud.io --ignore-not-found
```

### Configuration

There are two primary ways to customize the deployment:

- `--set key=value`: ad-hoc overrides
- `-f my-values.yaml`: provide a values file for versioned configuration

#### Parameters (Values)

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `global.imageRegistry` | string | `""` | Global image registry prefix override |
| `global.imagePullSecrets` | list | `[]` | Global image pull secrets |
| `nameOverride` | string | `""` | Name override (part of chart name) |
| `fullnameOverride` | string | `""` | Full name override |
| `image.registry` | string | `"quay.io"` | Image registry |
| `image.repository` | string | `"upmio/unit-operator"` | Image repository |
| `image.tag` | string | `"v1.0.0"` | Image tag |
| `image.digest` | string | `""` | Image digest (takes precedence over tag) |
| `image.pullPolicy` | string | `"IfNotPresent"` | Image pull policy |
| `image.pullSecrets` | list | `[]` | Image pull secrets |
| `replicaCount` | int | `1` | Number of replicas |
| `healthCheckPort` | int | `20152` | Health check port |
| `crds.enabled` | bool | `false` | Whether to install CRDs with the chart |
| `rbac.create` | bool | `true` | Whether to create RBAC resources |
| `serviceAccount.create` | bool | `true` | Whether to create a ServiceAccount |
| `serviceAccount.name` | string | `""` | Name of the ServiceAccount to create/use |
| `serviceAccount.automountServiceAccountToken` | bool | `true` | Auto-mount the SA token |
| `serviceAccount.annotations` | object | `{}` | Additional annotations for the ServiceAccount |
| `serviceAccount.labels` | object | `{}` | Additional labels for the ServiceAccount |
| `resources` | object | `{}` | Pod resource requests/limits |
| `service.type` | string | `"ClusterIP"` | Service type |
| `service.port` | int | `443` | Service port |
| `service.ipFamilyPolicy` | string | `""` | IP family policy |
| `service.ipFamilies` | list | `[]` | IP families (IPv4/IPv6) |
| `podAntiAffinityPreset` | string | `"soft"` | Pod anti-affinity preset (soft/hard) |
| `nodeAffinityPreset.type` | string | `""` | Node affinity type (soft/hard) |
| `nodeAffinityPreset.key` | string | `""` | Node label key |
| `nodeAffinityPreset.values` | list | `[]` | Node label values |
| `affinity` | object | `{}` | Custom affinity settings (overrides presets above) |
| `livenessProbe.enabled` | bool | `true` | Enable liveness probe |
| `livenessProbe.initialDelaySeconds` | int | `5` | Liveness probe initial delay |
| `livenessProbe.periodSeconds` | int | `10` | Liveness probe period |
| `livenessProbe.timeoutSeconds` | int | `1` | Liveness probe timeout |
| `livenessProbe.failureThreshold` | int | `3` | Liveness probe failure threshold |
| `livenessProbe.successThreshold` | int | `1` | Liveness probe success threshold |
| `readinessProbe.enabled` | bool | `true` | Enable readiness probe |
| `readinessProbe.initialDelaySeconds` | int | `5` | Readiness probe initial delay |
| `readinessProbe.periodSeconds` | int | `10` | Readiness probe period |
| `readinessProbe.timeoutSeconds` | int | `1` | Readiness probe timeout |
| `readinessProbe.failureThreshold` | int | `3` | Readiness probe failure threshold |
| `readinessProbe.successThreshold` | int | `1` | Readiness probe success threshold |
| `startupProbe.enabled` | bool | `true` | Enable startup probe |
| `startupProbe.initialDelaySeconds` | int | `15` | Startup probe initial delay |
| `startupProbe.periodSeconds` | int | `10` | Startup probe period |
| `startupProbe.timeoutSeconds` | int | `1` | Startup probe timeout |
| `startupProbe.failureThreshold` | int | `15` | Startup probe failure threshold |
| `startupProbe.successThreshold` | int | `1` | Startup probe success threshold |
| `tolerations` | list | `[]` | Taints tolerations |

### Documentation

- Operator API docs: `doc/unit-operator_api.md`
- Examples: see the `examples/` directory in the repository

### Compatibility

- Kubernetes: v1.27+
- OpenShift: v4.6+
