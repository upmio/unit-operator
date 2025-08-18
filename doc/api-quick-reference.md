# Unit Operator API Quick Reference

## 📋 API Resources Overview

| Resource | Version | Purpose | Key Features |
|----------|---------|---------|--------------|
| **UnitSet** | v1alpha2 | Database cluster management | Scaling, updates, shared config |
| **Unit** | v1alpha2 | Individual database instance | Pod management, storage, configuration |
| **GrpcCall** | v1alpha1 | Database operations | Backup, restore, configuration changes |
| **MysqlReplication** | v1alpha1 | MySQL replication | Async, semi-sync, group replication |
| **PostgresReplication** | v1alpha1 | PostgreSQL replication | Streaming replication |

---

# 🎯 GrpcCall (v1alpha1)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: operation-name
spec:
  targetUnit: "unit-name"
  type: "mysql|postgresql|proxysql"
  action: "logical-backup|physical-backup|restore|set-variable|clone|gtid-purge"
  ttlSecondsAfterFinished: 3600
  parameters:
    # Action-specific parameters
```

## Supported Actions

| Action | Description | Parameters |
|--------|-------------|------------|
| `logical-backup` | Logical database backup | backupType, compression, destination |
| `physical-backup` | Physical database backup | backupType, compression, destination |
| `restore` | Restore from backup | source, targetDatabase, overwrite |
| `set-variable` | Set configuration variables | variables (map) |
| `clone` | Clone from source | sourceHost, sourcePort, databases |
| `gtid-purge` | Purge GTID info (MySQL) | None |

---

# 🏗️ UnitSet (v1alpha2)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: cluster-name
spec:
  type: "mysql|postgresql|proxysql"
  version: "x.x.x"
  edition: "community|enterprise"
  units: 3
  sharedConfigName: "config-name"
  resources: {}
  storages: []
  secret: {}
  env: []
  updateStrategy: {}
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

# 📦 Unit (v1alpha2)

## Quick Reference

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Unit
metadata:
  name: unit-name
spec:
  startup: true
  sharedConfigName: "config-name"
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
| `template` | PodTemplateSpec | Required | Pod specification |

---

# 🔄 Replication Resources

## MysqlReplication (v1alpha1)

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
spec:
  mode: "rpl_async|rpl_semi_sync|group_replication"
  secret:
    name: secret-name
    mysql: replication-user-key
    replication: replication-user-key
  source:
    name: primary-unit
    host: primary-host
    port: 3306
  replica:
    - name: replica-unit
      host: replica-host
      port: 3306
```

## PostgresReplication (v1alpha1)

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: PostgresReplication
spec:
  mode: "rpl_async|rpl_sync"
  secret:
    name: secret-name
    postgres: postgres-user-key
    replication: replication-user-key
  primary:
    name: primary-unit
    host: primary-host
    port: 5432
  standby:
    - name: standby-unit
      host: standby-host
      port: 5432
```

---

# 🏷️ Common Labels

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

# PostgreSQL health check
kubectl exec -it postgresql-cluster-0 -- psql -c "SELECT 1"

# Check replication status
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW SLAVE STATUS\\G"
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

# 🔧 Common Configuration Patterns

## MySQL Cluster with Semi-Sync Replication
```yaml
# UnitSet
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

# MysqlReplication
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
metadata:
  name: mysql-replication
spec:
  mode: rpl_semi_sync
  secret:
    name: mysql-secret
    mysql: replication
    replication: replication
  source:
    name: mysql-cluster-0
    host: mysql-cluster-0.mysql-cluster-headless-svc.default
    port: 3306
  replica:
    - name: mysql-cluster-1
      host: mysql-cluster-1.mysql-cluster-headless-svc.default
      port: 3306
```

## PostgreSQL Cluster with Streaming Replication
```yaml
# UnitSet
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: postgresql-cluster
spec:
  type: postgresql
  version: "15.12"
  units: 3
  emptyDir:
    - name: data
      mountPath: /DATA_MOUNT
      size: 10Gi
  secret:
    name: postgresql-secret
    mountPath: /SECRET_MOUNT
  env:
    - name: ADM_USER
      value: postgres

# PostgresReplication
apiVersion: upm.syntropycloud.io/v1alpha1
kind: PostgresReplication
metadata:
  name: postgresql-replication
spec:
  mode: rpl_sync
  secret:
    name: postgresql-secret
    postgres: postgres
    replication: replication
  primary:
    name: postgresql-cluster-0
    host: postgresql-cluster-0.postgresql-cluster-headless-svc.default
    port: 5432
  standby:
    - name: postgresql-cluster-1
      host: postgresql-cluster-1.postgresql-cluster-headless-svc.default
      port: 5432
```

---

# 📊 Status Fields

## UnitSet Status
```yaml
status:
  units: 3           # Current number of units
  readyUnits: 3       # Number of ready units
  inUpdate: "false"   # Update status
  unitPVCSynced: Synced    # PVC synchronization
  unitImageSynced: Synced  # Image synchronization
  unitResourceSynced: Synced  # Resource synchronization
```

## Unit Status
```yaml
status:
  phase: Running      # Pod lifecycle phase
  nodeName: node-1    # Node where unit is running
  hostIP: 192.168.1.100  # Host IP
  podIPs:             # Pod IPs
    - ip: 10.244.0.5
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

# 🚨 Troubleshooting

## Common Issues

| Issue | Solution |
|-------|----------|
| UnitSet stuck in creation | Check operator logs, resource availability |
| Units not ready | Check pod logs, storage, resource limits |
| Replication not working | Check network connectivity, credentials |
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