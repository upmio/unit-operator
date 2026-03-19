# Unit Operator API Quick Reference

> Webhooks: `UnitSet`/`Unit` admission webhooks are enabled by default (disable via `ENABLE_WEBHOOKS=false`). `UnitSet` attaches finalizers during defaulting.

## API Resources Overview

| Resource | Version | Purpose | Key Features |
|----------|---------|---------|--------------|
| **UnitSet** | v1alpha2 | Database cluster management | Scaling, updates, shared config |
| **Unit** | v1alpha2 | Individual database instance | Pod management, storage, configuration |
| **Project** | v1alpha2 | Project-level configuration | CA management |
| **GrpcCall** | v1alpha1 | Database operations | Backup, restore, configuration changes |

---

# GrpcCall (v1alpha1)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: operation-name
spec:
  targetUnit: "unit-name"
  type: "mysql|postgresql|proxysql|redis|redis-sentinel|mongodb|milvus"
  action: "logical-backup|physical-backup|restore|set-variable|clone|gtid-purge|backup"
  ttlSecondsAfterFinished: 3600
  parameters:
    # Action-specific parameters
```

## Supported Actions

| Action | Description | Supported Types |
|--------|-------------|-----------------|
| `logical-backup` | Logical database backup | mysql, postgresql |
| `physical-backup` | Physical database backup | mysql, postgresql |
| `restore` | Restore from backup | mysql, postgresql, redis, mongodb, milvus |
| `set-variable` | Set configuration variables | mysql, postgresql, redis, mongodb, milvus |
| `clone` | Clone from source | mysql |
| `gtid-purge` | Purge GTID info (MySQL) | mysql |
| `backup` | Generic backup operation | redis, mongodb, milvus |

---

# UnitSet (v1alpha2)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: cluster-name
spec:
  type: "mysql|postgresql|proxysql|redis|redis-sentinel|mongodb|milvus"
  version: "x.x.x"
  edition: "community|enterprise"
  units: 3
  sharedConfigName: "config-name"
  resources: {}
  resizePolicy: []
  storages: []
  secret: {}
  env: []
  extraVolume: []
  updateStrategy: {}
  # optional fields
  externalService: {}
  unitService: {}
  nodeAffinityPreset: []
  podAntiAffinityPreset: ""
  emptyDir: []
  certificateSecret: {}
  certificateProfile: {}
  podMonitor: {}
```

## Key Configuration

| Field | Type | Required | Example |
|-------|------|----------|---------|
| `type` | string | Yes | `mysql` |
| `version` | string | Yes | `8.0.41` |
| `units` | int | Yes | `3` |
| `storages` | []StorageSpec | No | See below |
| `secret` | SecretInfo | No | See below |
| Annotation `upm.io/node-name-map` | map[string]string (JSON) | No | `{ "cluster-0": "node-a", "cluster-1": "noneSet" }` |

### Storage Configuration
```yaml
storages:
  - name: data
    mountPath: /DATA_MOUNT
    size: 100Gi
    storageClassName: standard
```

### Secret Configuration
```yaml
secret:
  name: secret-name
  mountPath: /SECRET_MOUNT
```

---

# Unit (v1alpha2)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Unit
metadata:
  name: unit-name
spec:
  startup: true
  sharedConfigName: "config-name"
  configTemplateName: "template-name"
  configValueName: "value-name"
  failedPodRecoveryPolicy: {}
  template:
    spec:
      containers: []
      volumes: []
  volumeClaimTemplates: []
```

## Key Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `startup` | bool | `true` | Start service automatically |
| `unbindNode` | bool | `false` | Node binding behavior |
| `configTemplateName` | string | - | Configuration template name |
| `configValueName` | string | - | Configuration value name |
| `failedPodRecoveryPolicy` | object | - | Failed pod recovery policy |
| `template` | PodTemplateSpec | Required | Pod specification |

---

# Project (v1alpha2)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Project
metadata:
  name: project-name
spec:
  ca:
    enabled: true
    commonName: "ca-name"
    secretName: "ca-secret"
    duration: "87600h"
    renewBefore: "720h"
    privateKey:
      algorithm: "ECDSA"
      size: 256
```

## Key Fields

| Field | Type | Description |
|-------|------|-------------|
| `ca.enabled` | bool | Enable CA configuration |
| `ca.commonName` | string | CA certificate common name |
| `ca.secretName` | string | Kubernetes secret storing the CA |
| `ca.duration` | string | Validity period (e.g., "87600h") |
| `ca.renewBefore` | string | Renewal time before expiration |
| `ca.privateKey.algorithm` | string | RSA, ECDSA, or Ed25519 |
| `ca.privateKey.size` | int | Private key size in bits |

---

# Common Labels

## Recommended Labels for UnitSet
```yaml
metadata:
  labels:
    upm.api/service-group.name: "cluster-name"
    upm.api/service-group.type: "database-type-sg"
    upm.api/service.name: "cluster-name"
    upm.io/owner: "team-name"
    app.kubernetes.io/name: "database-type"
    app.kubernetes.io/instance: "cluster-name"
    app.kubernetes.io/component: "database"
    app.kubernetes.io/managed-by: "unit-operator"
```

## Common Annotations
```yaml
metadata:
  annotations:
    unit-operator.io/backup-enabled: "true"
    unit-operator.io/monitoring-enabled: "true"
    unit-operator.io/maintenance-window: "sunday 2:00-4:00"
    unit-operator.io/environment: "production"
```

---

# 🚀 Common Commands

## Check Resource Status
```bash
# Check all Unit Operator resources
kubectl get unitsets,units,grpccalls

# Check specific UnitSet
kubectl describe unitset mysql-cluster

# Check UnitSet status
kubectl get unitset mysql-cluster -o yaml

# Check individual units
kubectl get units -l upm.api/service-group.name=mysql-cluster
```

## Check Database Health
```bash
# MySQL health check
kubectl exec -it mysql-cluster-0 -- mysql -e "SELECT 1"

# Check MySQL replication status
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW REPLICA STATUS\\G"
```

## Backup Operations
```bash
# Create logical backup
kubectl apply -f mysql-backup.yaml

# Check backup status
kubectl describe grpccall mysql-backup

# View backup logs
kubectl logs mysql-cluster-0 -c agent
```

---

# Common Configuration Patterns

## MySQL Cluster
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: mysql-cluster
spec:
  type: mysql
  version: "8.0.41"
  units: 3
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 100Gi
  secret:
    name: mysql-secret
    mountPath: /SECRET_MOUNT
  env:
    - name: ARCH_MODE
      value: rpl_semi_sync
```

## PostgreSQL Cluster
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: postgresql-cluster
spec:
  type: postgresql
  version: "15"
  units: 3
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 100Gi
  secret:
    name: postgresql-secret
    mountPath: /SECRET_MOUNT
  env:
    - name: ADM_USER
      value: postgres
```

## Redis Cluster
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: redis-cluster
spec:
  type: redis
  version: "7.0"
  units: 3
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 50Gi
  secret:
    name: redis-secret
    mountPath: /SECRET_MOUNT
```

---

# Status Fields

## UnitSet Status
```yaml
status:
  conditions: []        # Array of conditions
  observedGeneration: 1  # Most recent generation observed
  units: 3              # Current number of units
  readyUnits: 3         # Number of ready units
  inUpdate: "false"     # Update status
  unitPVCSynced: Synced      # PVC synchronization
  unitImageSynced: Synced    # Image synchronization
  unitResourceSynced: Synced # Resource synchronization
```

## Unit Status
```yaml
status:
  conditions: []        # Array of conditions
  observedGeneration: 1  # Most recent generation observed
  phase: Running        # Pod lifecycle phase
  nodeName: node-1      # Node where unit is running
  hostIP: 192.168.1.100 # Host IP
  podIPs:               # Pod IPs
    - ip: 10.244.0.5
  nodeReady: "True"
  task: ""
  processState: ""
  configSynced:
    status: "True"
    lastTransitionTime: "2024-01-01T10:00:00Z"
  persistentVolumeClaim:  # PVC information
    - name: data
      phase: Bound
      capacity:
        storage: 100Gi
```

## GrpcCall Status
```yaml
status:
  result: Success     # Operation result
  message: "Backup completed successfully"  # Status message
  startTime: "2024-01-01T10:00:00Z"  # Start time
  completionTime: "2024-01-01T10:30:00Z"  # Completion time
```

---

# Troubleshooting

## Common Issues

| Issue | Solution |
|-------|----------|
| UnitSet stuck in creation | Check operator logs, resource availability |
| Units not ready | Check pod logs, storage, resource limits |
| GrpcCall failing | Check agent logs, gRPC connectivity |

## Debug Commands
```bash
# Check operator logs
kubectl logs -n upm-system deployment/unit-operator

# Check pod events
kubectl describe pod mysql-cluster-0

# Check agent logs
kubectl logs mysql-cluster-0 -c agent

# Check resource status
kubectl get unitset mysql-cluster -o yaml
```

---

<div align="center">
  <p>
    <img src="https://img.icons8.com/color/48/000000/kubernetes.png" alt="Kubernetes" width="24" height="24">
    <img src="https://img.icons8.com/color/48/000000/database.png" alt="Database" width="24" height="24">
    <img src="https://img.icons8.com/color/48/000000/api.png" alt="API" width="24" height="24">
  </p>
  <p><strong>Unit Operator API Quick Reference</strong></p>
  <p>Fast reference for common API operations</p>
</div>