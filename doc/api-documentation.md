# Unit Operator API Documentation

## 📖 Overview

The Unit Operator provides a comprehensive API for managing database and middleware workloads in Kubernetes. This document details all available API resources, their specifications, and usage patterns.

## 🏗️ API Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Unit Operator API                        │
├─────────────────────────────────────────────────────────────┤
│  v1alpha1 (upm.syntropycloud.io)                           │
│  ├── GrpcCall - Execute operations on unit agents          │
│  └── MysqlReplication - MySQL replication management       │
│                                                             │
│  v1alpha2 (upm.syntropycloud.io)                           │
│  ├── UnitSet - Manage database clusters                    │
│  └── Unit - Individual database instances                  │
└─────────────────────────────────────────────────────────────┘
```

## 📚 API Versions

### v1alpha1 (upm.syntropycloud.io)
- **GrpcCall**: Execute gRPC operations on unit agents
- **MysqlReplication**: Manage MySQL replication (from compose-operator)
- **PostgresReplication**: Manage PostgreSQL replication (from compose-operator)

### v1alpha2 (upm.syntropycloud.io)
- **Unit**: Individual database instance
- **UnitSet**: Collection of database units with shared configuration

---

# 🎯 GrpcCall (v1alpha1)

## Overview

GrpcCall is a resource that executes specific operations on unit agents via gRPC calls. It provides a unified interface for performing database operations like backup, restore, and configuration changes.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: example-grpccall
  namespace: default
spec:
  targetUnit: mysql-unit-0
  type: mysql
  action: logical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "gzip"
    destination: "s3://backups/mysql/"
status:
  result: Success
  message: "Backup completed successfully"
  startTime: "2024-01-01T10:00:00Z"
  completionTime: "2024-01-01T10:30:00Z"
```

## Specification (Spec)

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `targetUnit` | string | Name of the target Unit resource |
| `type` | UnitType | Type of the target unit (mysql, postgresql, proxysql) |
| `action` | Action | Operation to perform on the unit |
| `ttlSecondsAfterFinished` | int32 | Time-to-live in seconds after completion |
| `parameters` | map[string]JSON | Action-specific parameters |

### UnitType Enum

| Value | Description |
|-------|-------------|
| `mysql` | MySQL database instance |
| `postgresql` | PostgreSQL database instance |
| `proxysql` | ProxySQL proxy instance |

### Action Enum

| Action | Description | Supported Unit Types |
|--------|-------------|---------------------|
| `logical-backup` | Perform logical database backup | mysql, postgresql |
| `physical-backup` | Perform physical database backup | mysql, postgresql |
| `restore` | Restore database from backup | mysql, postgresql |
| `gtid-purge` | Purge GTID information | mysql |
| `set-variable` | Set runtime configuration variables | mysql, postgresql |
| `clone` | Clone from another instance | mysql, postgresql |

### Action Parameters

#### logical-backup
```yaml
parameters:
  backupType: "full"          # full, incremental
  compression: "gzip"         # gzip, bzip2, none
  destination: "s3://backups/mysql/"
  retention: "7d"             # Retention period
  encryption: "AES256"        # Encryption algorithm
```

#### physical-backup
```yaml
parameters:
  backupType: "full"          # full, incremental
  compression: "true"         # Enable compression
  destination: "s3://backups/mysql/"
  parallel: "4"               # Number of parallel threads
```

#### restore
```yaml
parameters:
  source: "s3://backups/mysql/backup-20240101.tar.gz"
  targetDatabase: "restored_db"
  overwrite: "false"          # Overwrite existing data
```

#### set-variable
```yaml
parameters:
  variables:
    max_connections: "200"
    innodb_buffer_pool_size: "1G"
    log_bin: "ON"
```

#### clone
```yaml
parameters:
  sourceHost: "source-mysql.example.com"
  sourcePort: "3306"
  sourceUser: "replication"
  sourcePassword: "password"
  databases: ["db1", "db2"]  # Specific databases to clone
```

## Status

| Field | Type | Description |
|-------|------|-------------|
| `result` | Result | Operation result (Success, Failed) |
| `message` | string | Detailed result message |
| `startTime` | Time | Operation start time |
| `completionTime` | Time | Operation completion time |

### Result Enum

| Value | Description |
|-------|-------------|
| `Success` | Operation completed successfully |
| `Failed` | Operation failed |

## Usage Examples

### MySQL Logical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-logical-backup
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: logical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "gzip"
    destination: "s3://backups/mysql/"
    encryption: "AES256"
```

### PostgreSQL Physical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: postgresql-physical-backup
  namespace: default
spec:
  targetUnit: postgresql-cluster-0
  type: postgresql
  action: physical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "true"
    destination: "s3://backups/postgresql/"
    parallel: "4"
```

### MySQL Configuration Change
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-config-change
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: set-variable
  ttlSecondsAfterFinished: 600
  parameters:
    variables:
      max_connections: "500"
      innodb_buffer_pool_size: "2G"
      log_bin: "ON"
```

## Lifecycle

1. **Creation**: User creates GrpcCall resource
2. **Validation**: Operator validates target unit and action
3. **Execution**: Operator sends gRPC call to unit agent
4. **Monitoring**: Operator monitors operation progress
5. **Completion**: Operator updates status with result
6. **Cleanup**: Resource is automatically deleted after TTL

## Best Practices

- **TTL Management**: Set appropriate TTL values to prevent resource accumulation
- **Parameter Validation**: Validate all parameters before creating GrpcCall
- **Error Handling**: Monitor status field for operation results
- **Resource Cleanup**: Use finalizers for cleanup operations
- **Security**: Ensure sensitive parameters are stored in secrets

---

# 🏗️ UnitSet (v1alpha2)

## Overview

UnitSet is a Kubernetes resource that manages a collection of related database units with shared configuration, scaling capabilities, and lifecycle management. It serves as the primary resource for deploying database clusters.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: mysql-cluster
  namespace: default
  labels:
    upm.api/service-group.name: mysql-cluster
    upm.api/service-group.type: mysql-sg
spec:
  type: mysql
  version: "8.0.41"
  edition: community
  units: 3
  sharedConfigName: mysql-cluster-config
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 100Gi
      storageClassName: standard
  secret:
    mountPath: /SECRET_MOUNT
    name: mysql-cluster-secret
  env:
    - name: ARCH_MODE
      value: rpl_semi_sync
    - name: ADM_USER
      value: root
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      partition: 0
status:
  units: 3
  readyUnits: 3
  inUpdate: "false"
  unitPVCSynced: Synced
  unitImageSynced: Synced
  unitResourceSynced: Synced
```

## Specification (Spec)

### Core Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `type` | string | Yes | - | Database type (mysql, postgresql, proxysql) |
| `version` | string | Yes | - | Database version (e.g., "8.0.41") |
| `edition` | string | No | - | Database edition (community, enterprise) |
| `units` | int | Yes | - | Number of units to create |
| `sharedConfigName` | string | No | - | Name of shared configuration ConfigMap |

### Resource Management

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `resources` | ResourceRequirements | No | Resource limits and requests for containers |
| `env` | []EnvVar | No | Environment variables for units |
| Annotation `upm.io/node-name-map` | map[string]string (JSON) | No | Explicit node scheduling per unit; stored in metadata.annotations |

### Storage Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `storages` | []StorageSpec | No | Persistent storage configurations |
| `emptyDir` | []EmptyDirSpec | No | Ephemeral storage configurations |

#### StorageSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the storage volume |
| `mountPath` | string | Yes | Mount path in container |
| `size` | string | Yes | Size of the storage |
| `storageClassName` | string | No | Storage class name |

#### EmptyDirSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the emptyDir volume |
| `mountPath` | string | Yes | Mount path in container |
| `size` | string | No | Size limit for the volume |

### Security and Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secret` | SecretInfo | No | Secret mounting configuration |
| `certificateSecret` | CertificateSecretSpec | No | TLS certificate configuration |

#### SecretInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the secret |
| `mountPath` | string | Yes | Mount path in container |

#### CertificateSecretSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the certificate secret |
| `organization` | string | No | Organization name for certificate |

### Networking and Services

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `externalService` | ExternalServiceSpec | No | External service configuration |
| `unitService` | UnitServiceSpec | No | Internal service configuration |

#### ExternalServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (NodePort, LoadBalancer) |

#### UnitServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (ClusterIP) |

### Scheduling and Updates

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `updateStrategy` | UpdateStrategySpec | No | Update strategy configuration |
| `nodeAffinityPreset` | []NodeAffinityPresetSpec | No | Node affinity rules |
| `podAntiAffinityPreset` | string | No | Pod anti-affinity policy |

#### UpdateStrategySpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Update strategy type |
| `rollingUpdate` | RollingUpdateSpec | No | Rolling update configuration |

#### RollingUpdateSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `partition` | int | No | Number of partitions for update |
| `maxUnavailable` | int | No | Maximum unavailable pods during update |

#### NodeAffinityPresetSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | Node label key |
| `values` | []string | Yes | Node label values |

## Status

| Field | Type | Description |
|-------|------|-------------|
| `units` | int | Current number of units |
| `readyUnits` | int | Number of ready units |
| `inUpdate` | string | Update status |
| `unitPVCSynced` | PvcSyncStatus | PVC synchronization status |
| `unitImageSynced` | ImageSyncStatus | Image synchronization status |
| `unitResourceSynced` | ResourceSyncStatus | Resource synchronization status |

### Status Enums

#### PvcSyncStatus
- `Synced`
- `NotSynced`
- `Syncing`

#### ImageSyncStatus
- `Synced`
- `NotSynced`
- `Syncing`

#### ResourceSyncStatus
- `Synced`
- `NotSynced`
- `Syncing`

## Supported Database Types

### MySQL
- **Versions**: 5.7, 8.0+
- **Editions**: community, enterprise
- **Replication Modes**: async, semi-sync, group-replication

### PostgreSQL
- **Versions**: 12, 13, 14, 15+
- **Editions**: community, enterprise
- **Replication Modes**: streaming-replication

### ProxySQL
- **Versions**: 2.0+
- **Editions**: community, enterprise
- **Replication Modes**: N/A

## Usage Examples

### MySQL Cluster with Semi-Sync Replication
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: mysql-cluster
  namespace: default
  labels:
    upm.api/service-group.name: mysql-cluster
    upm.api/service-group.type: mysql-sg
spec:
  type: mysql
  version: "8.0.41"
  edition: community
  units: 3
  sharedConfigName: mysql-cluster-config
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 100Gi
      storageClassName: standard
  secret:
    mountPath: /SECRET_MOUNT
    name: mysql-cluster-secret
  env:
    - name: ARCH_MODE
      value: rpl_semi_sync
    - name: ADM_USER
      value: root
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      partition: 0
```

### PostgreSQL Cluster with Streaming Replication
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: postgresql-cluster
  namespace: default
  labels:
    upm.api/service-group.name: postgresql-cluster
    upm.api/service-group.type: postgresql-sg
spec:
  type: postgresql
  version: "15.12"
  units: 3
  sharedConfigName: postgresql-cluster-config
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi
  emptyDir:
    - name: data
      mountPath: /DATA_MOUNT
      size: 10Gi
    - name: log
      mountPath: /LOG_MOUNT
      size: 5Gi
  secret:
    mountPath: /SECRET_MOUNT
    name: postgresql-cluster-secret
  env:
    - name: ADM_USER
      value: postgres
    - name: MON_USER
      value: monitor
    - name: REPL_USER
      value: replication
  nodeAffinityPreset:
    - key: kubernetes.io/arch
      values:
        - amd64
    - key: upm.api/software.postgresql
      values:
        - "true"
```

### ProxySQL Configuration
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: proxysql-cluster
  namespace: default
spec:
  type: proxysql
  version: "2.4.6"
  units: 1
  sharedConfigName: proxysql-cluster-config
  resources:
    limits:
      cpu: "1"
      memory: 1Gi
    requests:
      cpu: "500m"
      memory: 512Mi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 10Gi
      storageClassName: standard
  secret:
    mountPath: /SECRET_MOUNT
    name: proxysql-cluster-secret
  externalService:
    type: LoadBalancer
```

## Lifecycle Management

### Unit Creation
1. **Validation**: Operator validates UnitSet specification
2. **Configuration**: Creates shared ConfigMaps and Secrets
3. **Service Creation**: Creates headless and external services
4. **Unit Creation**: Creates individual Unit resources
5. **Pod Scheduling**: Units create pods with persistent storage

### Scaling Operations
- **Scale Up**: Add new units to the cluster
- **Scale Down**: Remove units gracefully with data migration
- **Rolling Updates**: Update units one by one with zero downtime

### Update Strategies
- **RollingUpdate**: Update units sequentially
- **Partition Updates**: Update specific partitions
- **Canary Updates**: Test updates on subset of units

## Best Practices

### Resource Management
- **Resource Limits**: Set appropriate CPU and memory limits
- **Storage Planning**: Plan storage requirements based on data growth
- **Node Affinity**: Use node affinity for performance optimization

### High Availability
- **Multiple Units**: Deploy at least 3 units for production
- **Replication**: Configure appropriate replication mode
- **Monitoring**: Set up comprehensive monitoring

### Security
- **Secrets Management**: Use Kubernetes secrets for credentials
- **Network Policies**: Implement network policies for security
- **TLS Configuration**: Enable TLS for communication

---

# 📦 Unit (v1alpha2)

## Overview

Unit is a Kubernetes resource that represents an individual database instance. It's typically created and managed by a UnitSet but can also be created independently for specific use cases.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Unit
metadata:
  name: mysql-cluster-0
  namespace: default
spec:
  unbindNode: false
  startup: true
  sharedConfigName: mysql-cluster-config
  configTemplateName: mysql-config-template
  configValueName: mysql-config-value
  template:
    spec:
      containers:
        - name: mysql
          image: mysql:8.0.41
          ports:
            - containerPort: 3306
              name: mysql
          volumeMounts:
            - name: data
              mountPath: /DATA_MOUNT
            - name: secret
              mountPath: /SECRET_MOUNT
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: mysql-cluster-0-data
        - name: secret
          secret:
            secretName: mysql-cluster-secret
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 100Gi
        storageClassName: standard
status:
  phase: Running
  nodeName: node-1
  hostIP: 192.168.1.100
  podIPs:
    - ip: 10.244.0.5
  configSyncStatus:
    lastTransitionTime: "2024-01-01T10:00:00Z"
  persistentVolumeClaim:
    - name: data
      volumeName: pvc-12345
      accessModes: ["ReadWriteOnce"]
      capacity:
        storage: 100Gi
      phase: Bound
```

## Specification (Spec)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `unbindNode` | bool | No | false | Controls node binding behavior |
| `startup` | bool | No | true | Controls service startup |
| `sharedConfigName` | string | No | - | Shared configuration name |
| `configTemplateName` | string | No | - | Configuration template name |
| `configValueName` | string | No | - | Configuration value name |
| `template` | PodTemplateSpec | Yes | - | Pod specification template |
| `volumeClaimTemplates` | []PersistentVolumeClaim | No | - | PVC templates |

### Node Binding Behavior

| `unbindNode` | Behavior |
|-------------|----------|
| `false` | Pod.Spec.NodeName is written to annotations and Spec |
| `true` | No automatic node binding |

### Configuration Management

| Field | Purpose |
|-------|---------|
| `sharedConfigName` | References shared configuration from UnitSet |
| `configTemplateName` | Template for generating configuration |
| `configValueName` | Unit-specific configuration values |

## Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | UnitPhase | Lifecycle state of the unit |
| `nodeName` | string | Node where the unit is scheduled |
| `hostIP` | string | Host IP address |
| `podIPs` | []PodIP | Pod IP addresses |
| `configSyncStatus` | ConfigSyncStatus | Configuration synchronization status |
| `persistentVolumeClaim` | []PvcInfo | PVC status information |

### UnitPhase Enum

| Phase | Description |
|-------|-------------|
| `Pending` | Unit accepted but not yet started |
| `Running` | Unit is running and healthy |
| `Ready` | Unit is ready to serve traffic |
| `Succeeded` | Unit completed successfully |
| `Failed` | Unit failed to start or crashed |
| `Unknown` | Unit state cannot be determined |

### PvcInfo

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | PVC name |
| `volumeName` | string | Volume name |
| `accessModes` | []PersistentVolumeAccessMode | Access modes |
| `capacity` | PvcCapacity | Storage capacity |
| `phase` | PersistentVolumeClaimPhase | PVC phase |

### PvcCapacity

| Field | Type | Description |
|-------|------|-------------|
| `storage` | Quantity | Storage amount |

## Usage Examples

### MySQL Unit
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Unit
metadata:
  name: mysql-cluster-0
  namespace: default
spec:
  unbindNode: false
  startup: true
  sharedConfigName: mysql-cluster-config
  template:
    spec:
      containers:
        - name: mysql
          image: mysql:8.0.41
          ports:
            - containerPort: 3306
              name: mysql
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mysql-cluster-secret
                  key: root
          volumeMounts:
            - name: data
              mountPath: /DATA_MOUNT
            - name: secret
              mountPath: /SECRET_MOUNT
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: mysql-cluster-0-data
        - name: secret
          secret:
            secretName: mysql-cluster-secret
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 100Gi
        storageClassName: standard
```

### PostgreSQL Unit
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Unit
metadata:
  name: postgresql-cluster-0
  namespace: default
spec:
  unbindNode: false
  startup: true
  sharedConfigName: postgresql-cluster-config
  template:
    spec:
      containers:
        - name: postgresql
          image: postgres:15.12
          ports:
            - containerPort: 5432
              name: postgresql
          env:
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgresql-cluster-secret
                  key: postgres
          volumeMounts:
            - name: data
              mountPath: /DATA_MOUNT
            - name: log
              mountPath: /LOG_MOUNT
            - name: secret
              mountPath: /SECRET_MOUNT
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: postgresql-cluster-0-data
        - name: log
          emptyDir:
            sizeLimit: 5Gi
        - name: secret
          secret:
            secretName: postgresql-cluster-secret
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 100Gi
        storageClassName: standard
```

## Lifecycle Management

### Unit Creation
1. **Validation**: Operator validates Unit specification
2. **PVC Creation**: Creates persistent volume claims
3. **Pod Creation**: Creates pod with database container
4. **Configuration**: Applies configuration from templates
5. **Readiness**: Waits for database to become ready

### Unit Updates
- **Rolling Updates**: Update pods without downtime
- **Configuration Changes**: Apply new configuration changes
- **Storage Expansion**: Expand persistent storage

### Unit Deletion
- **Graceful Shutdown**: Properly shutdown database
- **Data Cleanup**: Clean up persistent data if configured
- **Resource Cleanup**: Remove associated resources

## Agent Integration

Each Unit includes a sidecar agent that provides:
- **Configuration Management**: Dynamic configuration updates
- **Backup Operations**: Backup and restore functionality
- **Health Monitoring**: Database health checks
- **Metrics Collection**: Performance metrics collection
- **gRPC Operations**: Execute operations via GrpcCall

## Best Practices

### Resource Management
- **Storage Planning**: Plan for data growth and retention
- **Memory Management**: Set appropriate memory limits
- **CPU Allocation**: Allocate sufficient CPU resources

### Configuration Management
- **Template Usage**: Use configuration templates for consistency
- **Secret Management**: Store sensitive data in secrets
- **Environment Variables**: Use environment variables for configuration

### Monitoring and Observability
- **Health Checks**: Configure proper health checks
- **Metrics Collection**: Enable metrics collection
- **Logging**: Configure appropriate logging levels

---

# 🔄 MysqlReplication (v1alpha1)

## Overview

MysqlReplication is a resource that manages MySQL replication relationships between database instances. It's typically used in conjunction with UnitSet to establish replication topologies.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
metadata:
  name: mysql-cluster-replication
  namespace: default
spec:
  mode: rpl_semi_sync
  secret:
    name: mysql-cluster-secret
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
    - name: mysql-cluster-2
      host: mysql-cluster-2.mysql-cluster-headless-svc.default
      port: 3306
  service:
    type: NodePort
```

## Specification (Spec)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | Yes | Replication mode |
| `secret` | SecretRef | Yes | Secret reference for credentials |
| `source` | SourceSpec | Yes | Primary database specification |
| `replica` | []ReplicaSpec | Yes | Replica database specifications |
| `service` | ServiceSpec | No | Service configuration |

### Replication Modes

| Mode | Description |
|------|-------------|
| `rpl_async` | Asynchronous replication |
| `rpl_semi_sync` | Semi-synchronous replication |
| `group_replication` | Group replication |

### SecretRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Secret name |
| `mysql` | string | Yes | Key for MySQL user in secret |
| `replication` | string | Yes | Key for replication user in secret |

### SourceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Source unit name |
| `host` | string | Yes | Source host address |
| `port` | int | Yes | Source port |

### ReplicaSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Replica unit name |
| `host` | string | Yes | Replica host address |
| `port` | int | Yes | Replica port |

### ServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (NodePort, LoadBalancer) |

## Usage Examples

### Semi-Synchronous Replication
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
metadata:
  name: mysql-cluster-replication
  namespace: default
spec:
  mode: rpl_semi_sync
  secret:
    name: mysql-cluster-secret
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
    - name: mysql-cluster-2
      host: mysql-cluster-2.mysql-cluster-headless-svc.default
      port: 3306
  service:
    type: NodePort
```

### Group Replication
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
metadata:
  name: mysql-group-replication
  namespace: default
spec:
  mode: group_replication
  secret:
    name: mysql-cluster-secret
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
    - name: mysql-cluster-2
      host: mysql-cluster-2.mysql-cluster-headless-svc.default
      port: 3306
  service:
    type: LoadBalancer
```

## Best Practices

### Replication Configuration
- **Network Connectivity**: Ensure proper network connectivity between instances
- **Credential Management**: Use appropriate replication users with required privileges
- **Monitoring**: Monitor replication lag and health

### High Availability
- **Multiple Replicas**: Use multiple replicas for high availability
- **Automatic Failover**: Configure automatic failover mechanisms
- **Backup Strategy**: Implement regular backup strategies

---

# 🔄 PostgresReplication (v1alpha1)

## Overview

PostgresReplication is a resource that manages PostgreSQL streaming replication between database instances. It's used to establish replication topologies for PostgreSQL clusters.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: PostgresReplication
metadata:
  name: postgresql-cluster-replication
  namespace: default
spec:
  mode: rpl_sync
  secret:
    name: postgresql-cluster-secret
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
    - name: postgresql-cluster-2
      host: postgresql-cluster-2.postgresql-cluster-headless-svc.default
      port: 5432
  service:
    type: NodePort
```

## Specification (Spec)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | Yes | Replication mode |
| `secret` | SecretRef | Yes | Secret reference for credentials |
| `primary` | PrimarySpec | Yes | Primary database specification |
| `standby` | []StandbySpec | Yes | Standby database specifications |
| `service` | ServiceSpec | No | Service configuration |

### Replication Modes

| Mode | Description |
|------|-------------|
| `rpl_async` | Asynchronous replication |
| `rpl_sync` | Synchronous replication |

### SecretRef

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Secret name |
| `postgres` | string | Yes | Key for postgres user in secret |
| `replication` | string | Yes | Key for replication user in secret |

### PrimarySpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Primary unit name |
| `host` | string | Yes | Primary host address |
| `port` | int | Yes | Primary port |

### StandbySpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Standby unit name |
| `host` | string | Yes | Standby host address |
| `port` | int | Yes | Standby port |

### ServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (NodePort, LoadBalancer) |

## Usage Examples

### Synchronous Replication
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: PostgresReplication
metadata:
  name: postgresql-cluster-replication
  namespace: default
spec:
  mode: rpl_sync
  secret:
    name: postgresql-cluster-secret
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
    - name: postgresql-cluster-2
      host: postgresql-cluster-2.postgresql-cluster-headless-svc.default
      port: 5432
  service:
    type: NodePort
```

## Best Practices

### Replication Configuration
- **Network Connectivity**: Ensure proper network connectivity between instances
- **WAL Configuration**: Configure appropriate WAL settings
- **Archive Mode**: Enable archive mode for point-in-time recovery

### High Availability
- **Multiple Standbys**: Use multiple standby instances
- **Automatic Failover**: Configure automatic failover with tools like Patroni
- **Monitoring**: Monitor replication lag and WAL archive status

---

# 🎯 Common Use Cases

## 🚀 Database Cluster Deployment

### MySQL Cluster with Semi-Sync Replication
```yaml
# Secret
apiVersion: v1
kind: Secret
metadata:
  name: mysql-cluster-secret
  namespace: default
data:
  root: VjVyOExqQmxyM3N2ODNsUHVrbmhDK29KZGRRQXl1dzlOWTJmNEJ6djdoRT0=
  replication: V1Y2MVZyZFRjbEVuT1lkbEx0R1c5cTljUmd0UjZubTlOM05aZkw2NkRxbz0=
type: Opaque

# ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql-cluster-config
  namespace: default
data:
  service_group_uid: "6fa3ca2a-0ffd-4ca7-8615-e2589f7dd413"
  mysql_ports: '[{"name": "mysql", "containerPort": "3306","protocol": "TCP"}]'

# UnitSet
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: mysql-cluster
  namespace: default
spec:
  type: mysql
  version: "8.0.41"
  edition: community
  units: 3
  sharedConfigName: mysql-cluster-config
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 100Gi
      storageClassName: standard
  secret:
    mountPath: /SECRET_MOUNT
    name: mysql-cluster-secret
  env:
    - name: ARCH_MODE
      value: rpl_semi_sync
    - name: ADM_USER
      value: root

# MysqlReplication
apiVersion: upm.syntropycloud.io/v1alpha1
kind: MysqlReplication
metadata:
  name: mysql-cluster-replication
  namespace: default
spec:
  mode: rpl_semi_sync
  secret:
    name: mysql-cluster-secret
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
    - name: mysql-cluster-2
      host: mysql-cluster-2.mysql-cluster-headless-svc.default
      port: 3306
  service:
    type: NodePort
```

## 💾 Database Backup Operations

### MySQL Logical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-logical-backup
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: logical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "gzip"
    destination: "s3://backups/mysql/"
    encryption: "AES256"
    retention: "7d"
```

### PostgreSQL Physical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: postgresql-physical-backup
  namespace: default
spec:
  targetUnit: postgresql-cluster-0
  type: postgresql
  action: physical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "true"
    destination: "s3://backups/postgresql/"
    parallel: "4"
```

## ⚙️ Configuration Management

### MySQL Configuration Update
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-config-update
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: set-variable
  ttlSecondsAfterFinished: 600
  parameters:
    variables:
      max_connections: "500"
      innodb_buffer_pool_size: "2G"
      log_bin: "ON"
      slow_query_log: "ON"
      long_query_time: "2"
```

### PostgreSQL Configuration Update
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: postgresql-config-update
  namespace: default
spec:
  targetUnit: postgresql-cluster-0
  type: postgresql
  action: set-variable
  ttlSecondsAfterFinished: 600
  parameters:
    variables:
      max_connections: "200"
      shared_buffers: "1GB"
      effective_cache_size: "3GB"
      maintenance_work_mem: "256MB"
      checkpoint_completion_target: "0.9"
```

## 🔄 Database Cloning

### MySQL Clone Operation
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-clone-operation
  namespace: default
spec:
  targetUnit: mysql-cluster-1
  type: mysql
  action: clone
  ttlSecondsAfterFinished: 7200
  parameters:
    sourceHost: "source-mysql.example.com"
    sourcePort: "3306"
    sourceUser: "replication"
    sourcePassword: "password"
    databases: ["app_db", "config_db"]
    skipTables: ["temp_data.*"]
```

---

# 🔧 Advanced Configuration

## 🏷️ Labels and Annotations

### Recommended Labels
```yaml
metadata:
  labels:
    upm.api/service-group.name: "mysql-cluster"
    upm.api/service-group.type: "mysql-sg"
    upm.api/service.name: "mysql-cluster"
    upm.io/owner: "database-team"
    app.kubernetes.io/name: "mysql"
    app.kubernetes.io/instance: "mysql-cluster"
    app.kubernetes.io/component: "database"
    app.kubernetes.io/managed-by: "unit-operator"
```

### Common Annotations
```yaml
metadata:
  annotations:
    unit-operator.io/backup-enabled: "true"
    unit-operator.io/monitoring-enabled: "true"
    unit-operator.io/maintenance-window: "sunday 2:00-4:00"
    unit-operator.io/environment: "production"
```

## 🌐 Networking Configuration

### Service Configuration
```yaml
spec:
  # External service for application access
  externalService:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
  
  # Internal service for inter-pod communication
  unitService:
    type: ClusterIP
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-internal: "true"
```

### Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mysql-cluster-network-policy
  namespace: default
spec:
  podSelector:
    matchLabels:
      upm.api/service-group.name: mysql-cluster
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: app-namespace
    ports:
    - protocol: TCP
      port: 3306
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: storage-namespace
    ports:
    - protocol: TCP
      port: 3306
```

## 📊 Monitoring and Observability

### Prometheus ServiceMonitor
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: mysql-cluster-monitoring
  namespace: monitoring
  labels:
    app: mysql-exporter
spec:
  selector:
    matchLabels:
      upm.api/service-group.name: mysql-cluster
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Custom Metrics
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql-metrics-config
  namespace: default
data:
  metrics.yaml: |
    metrics:
      - name: mysql_connections
        query: "SHOW STATUS LIKE 'Threads_connected'"
        type: gauge
      - name: mysql_queries
        query: "SHOW STATUS LIKE 'Queries'"
        type: counter
      - name: mysql_slow_queries
        query: "SHOW STATUS LIKE 'Slow_queries'"
        type: counter
```

## 🔐 Security Configuration

### TLS Configuration
```yaml
spec:
  certificateSecret:
    name: mysql-tls-secret
    organization: "MyCompany"
  env:
    - name: SSL_MODE
      value: "REQUIRED"
    - name: SSL_CA
      value: "/etc/ssl/certs/ca.crt"
    - name: SSL_CERT
      value: "/etc/ssl/certs/server.crt"
    - name: SSL_KEY
      value: "/etc/ssl/certs/server.key"
```

### RBAC Configuration
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: mysql-cluster-role
  namespace: default
rules:
- apiGroups: ["upm.syntropycloud.io"]
  resources: ["units", "unitsets", "grpccalls"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
```

---

# 🚨 Troubleshooting

## Common Issues and Solutions

### UnitSet Creation Fails
```bash
# Check UnitSet status
kubectl describe unitset mysql-cluster

# Check operator logs
kubectl logs -n upm-system deployment/unit-operator

# Check events
kubectl get events --field-selector involvedObject.name=mysql-cluster
```

### Unit Pods Stuck in Pending
```bash
# Check pod events
kubectl describe pod mysql-cluster-0

# Check resource availability
kubectl get nodes
kubectl top nodes

# Check storage class
kubectl get storageclass
```

### Replication Not Working
```bash
# Check replication status
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW SLAVE STATUS\\G"

# Check network connectivity
kubectl exec -it mysql-cluster-0 -- nslookup mysql-cluster-1

# Check secret configuration
kubectl get secret mysql-cluster-secret -o yaml
```

### GrpcCall Operations Failing
```bash
# Check GrpcCall status
kubectl describe grpccall mysql-backup

# Check unit agent logs
kubectl logs mysql-cluster-0 -c agent

# Check gRPC connectivity
kubectl exec -it mysql-cluster-0 -- curl -X POST http://localhost:8080/health
```

## Debug Commands

### General Health Check
```bash
# Check all resources
kubectl get unitsets,units,grpccalls -n default

# Check resource status
kubectl get unitset mysql-cluster -o yaml

# Check pod status
kubectl get pods -l upm.api/service-group.name=mysql-cluster
```

### Database Health Check
```bash
# MySQL health check
kubectl exec -it mysql-cluster-0 -- mysql -e "SELECT 1"

# PostgreSQL health check
kubectl exec -it postgresql-cluster-0 -- psql -c "SELECT 1"

# ProxySQL health check
kubectl exec -it proxysql-cluster-0 -- mysql -h 127.0.0.1 -P 6032 -e "SELECT 1"
```

### Configuration Validation
```bash
# Check configuration files
kubectl exec -it mysql-cluster-0 -- cat /etc/mysql/my.cnf

# Check environment variables
kubectl exec -it mysql-cluster-0 -- env | grep MYSQL_

# Check mounted secrets
kubectl exec -it mysql-cluster-0 -- ls -la /SECRET_MOUNT/
```

## Performance Monitoring

### Resource Usage
```bash
# Monitor CPU and memory usage
kubectl top pods -l upm.api/service-group.name=mysql-cluster

# Check storage usage
kubectl exec -it mysql-cluster-0 -- df -h

# Check network usage
kubectl exec -it mysql-cluster-0 -- netstat -s
```

### Database Performance
```bash
# MySQL performance metrics
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW GLOBAL STATUS"
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW PROCESSLIST"

# PostgreSQL performance metrics
kubectl exec -it postgresql-cluster-0 -- psql -c "SELECT * FROM pg_stat_activity"
kubectl exec -it postgresql-cluster-0 -- psql -c "SELECT * FROM pg_stat_database"
```

---

# 📚 Additional Resources

## Official Documentation
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [MySQL Documentation](https://dev.mysql.com/doc/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [ProxySQL Documentation](https://github.com/sysown/proxysql/wiki)

## Related Tools
- [Kubebuilder](https://book.kubebuilder.io/) - Kubernetes API development framework
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller framework
- [Prometheus](https://prometheus.io/) - Monitoring system
- [Grafana](https://grafana.com/) - Visualization dashboard

## Community Resources
- [GitHub Issues](https://github.com/upmio/unit-operator/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/upmio/unit-operator/discussions) - Community discussions
- [Stack Overflow](https://stackoverflow.com/) - Q&A with unit-operator tag

## Contributing
- [Contributing Guidelines](CONTRIBUTING.md) - How to contribute to the project
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community guidelines
- [Development Setup](README.md#development) - Setting up development environment

---

<div align="center">
  <p>
    <img src="https://img.icons8.com/color/96/000000/kubernetes.png" alt="Kubernetes" width="32" height="32">
    <img src="https://img.icons8.com/color/96/000000/database.png" alt="Database" width="32" height="32">
    <img src="https://img.icons8.com/color/96/000000/api.png" alt="API" width="32" height="32">
  </p>
  <p><strong>Unit Operator API Documentation</strong></p>
  <p>Complete reference for managing database workloads in Kubernetes</p>
</div>