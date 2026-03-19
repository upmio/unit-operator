# Unit Operator API Documentation

## Overview

The Unit Operator provides a comprehensive API for managing database and middleware workloads in Kubernetes. This document details all available API resources, their specifications, and usage patterns.

## API Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Unit Operator API                        │
├─────────────────────────────────────────────────────────────┤
│  v1alpha1 (upm.syntropycloud.io)                           │
│  └── GrpcCall - Execute operations on unit agents          │
│                                                             │
│  v1alpha2 (upm.syntropycloud.io)                           │
│  ├── UnitSet - Manage database clusters                    │
│  ├── Unit - Individual database instances                  │
│  └── Project - Project-level configuration and resources   │
└─────────────────────────────────────────────────────────────┘
```

> Webhooks: `UnitSet`/`Unit` admission webhooks are enabled by default (can be disabled via `ENABLE_WEBHOOKS=false`).
> - Defaulting: `UnitSet` creation/update automatically attaches finalizers (`upm.io/unit-delete`, `upm.io/configmap-delete`).
> - Validation: Validation hooks are placeholders and can be extended as needed.

## API Versions

### v1alpha1 (upm.syntropycloud.io)
- **GrpcCall**: Execute gRPC operations on unit agents

### v1alpha2 (upm.syntropycloud.io)
- **Unit**: Individual database instance
- **UnitSet**: Collection of database units with shared configuration
- **Project**: Project-level configuration and resources

---

# GrpcCall (v1alpha1)

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

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `ttlSecondsAfterFinished` | int32 | TTL after completion; if set, resource is eligible for auto-deletion after TTL |
| `parameters` | map[string]JSON | Action-specific parameters (defaults to empty if omitted) |

### UnitType Enum (matches code)

| Value | Description |
|-------|-------------|
| `mysql` | MySQL database instance |
| `postgresql` | PostgreSQL database instance |
| `proxysql` | ProxySQL proxy instance |
| `redis` | Redis database instance |
| `redis-sentinel` | Redis Sentinel instance |
| `mongodb` | MongoDB database instance |
| `milvus` | Milvus vector database instance |

### Action Enum (matches code)

| Action | Description | Supported Unit Types |
|--------|-------------|---------------------|
| `logical-backup` | Perform logical database backup | mysql, postgresql |
| `physical-backup` | Perform physical database backup | mysql, postgresql |
| `restore` | Restore database from backup | mysql, postgresql, redis, mongodb, milvus |
| `gtid-purge` | Purge GTID information | mysql |
| `set-variable` | Set runtime configuration variables | mysql, postgresql, redis, mongodb, milvus |
| `clone` | Clone from another instance | mysql |
| `backup` | Generic backup operation | redis, mongodb, milvus |

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

### MySQL Physical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-physical-backup
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: physical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "true"
    destination: "s3://backups/mysql/"
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
- **Security**: Ensure sensitive parameters are stored in Kubernetes Secrets

---

# UnitSet (v1alpha2)

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
| `resizePolicy` | []ContainerResizePolicy | No | Resize policy for container resources |
| `env` | []EnvVar | No | Environment variables for units |
| `extraVolume` | []ExtraVolumeInfo | No | Extra volume configurations |
| Annotation `upm.io/node-name-map` | map[string]string (JSON) | No | Explicit node scheduling per unit; stored in metadata.annotations |

#### ExtraVolumeInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `volume` | Volume | Yes | Volume definition (corev1.Volume) |
| `volumeMountPath` | string | Yes | Mount path for the extra volume |

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
| `certificateProfile` | CertificateProfile | No | Additional CA and org settings |

#### SecretInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Name of the secret |
| `mountPath` | string | No | Mount path in container |

#### CertificateSecretSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Name of the certificate secret |
| `organization` | string | No | Organization name for certificate |

#### CertificateProfile

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `organizations` | []string | No | List of organization names for CA/cert |
| `root_secret` | string | No | Root CA secret name |

### Networking and Services

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `externalService` | ExternalServiceSpec | No | External service configuration |
| `unitService` | UnitServiceSpec | No | Internal service configuration |

#### ExternalServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (ClusterIP, NodePort, LoadBalancer, ExternalName) |

#### UnitServiceSpec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | No | Service type (ClusterIP, NodePort, LoadBalancer, ExternalName) |

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

### Monitoring

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `podMonitor` | PodMonitorInfo | No | PodMonitor configuration for Prometheus scraping |

#### PodMonitorInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enable` | bool | No | Enable PodMonitor creation (default: false) |
| `endpoints` | []PodMonitorEndpoint | No | Scrape endpoint configurations |

#### PodMonitorEndpoint

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `port` | string | No | Pod port name exposed for scraping (default: "metrics") |
| `relabelConfigs` | []PodMonitorRelabelConfig | No | Relabeling rules for this endpoint |

#### PodMonitorRelabelConfig

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `targetLabel` | string | No | Label to which the resulting string is written |
| `replacement` | string | No | Replacement value against which regex match is performed |
| `action` | string | No | Action to perform based on regex matching |

## Status

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | []metav1.Condition | No | Array of conditions representing the observed state |
| `observedGeneration` | int64 | No | Most recent generation observed |
| `units` | int | Current number of units |
| `readyUnits` | int | Number of ready units |
| `inUpdate` | string | Update status |
| `unitPVCSynced` | PvcSyncStatus | PVC synchronization status |
| `unitImageSynced` | ImageSyncStatus | Image synchronization status |
| `unitResourceSynced` | ResourceSyncStatus | Resource synchronization status |
| `externalService` | ExternalServiceStatus | No | External service information |
| `unitService` | UnitServiceStatus | No | Unit service information |

#### ExternalServiceStatus

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name of the external service |

#### UnitServiceStatus

| Field | Type | Description |
|-------|------|-------------|
| `name` | map[string]string | Map of unit name to service name |

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
- **Editions**: community, enterprise
- **Replication Modes**: async, semi-sync, group-replication

### PostgreSQL
- **Editions**: community, enterprise
- **Replication Modes**: streaming-replication

### ProxySQL
- **Editions**: community, enterprise

### Redis
- **Editions**: community
- **Replication Modes**: async, semi-sync

### Redis Sentinel
- **Editions**: community
- **Replication Modes**: sentinel-managed failover

### MongoDB
- **Editions**: community, enterprise
- **Replication Modes**: replica set

### Milvus
- **Editions**: community, enterprise
- **Replication Modes**: distributed

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
  nodeAffinityPreset:
    - key: kubernetes.io/arch
      values:
        - amd64
    - key: upm.api/software.mysql
      values:
        - "true"
```

### MySQL Single Instance
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: mysql-single
  namespace: default
spec:
  type: mysql
  version: "8.0.41"
  edition: community
  units: 1
  sharedConfigName: mysql-config
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
      size: 50Gi
      storageClassName: standard
  secret:
    mountPath: /SECRET_MOUNT
    name: mysql-secret
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
- **Secrets Management**: Use Kubernetes Secrets for credentials
- **Network Policies**: Implement network policies for security
- **TLS Configuration**: Enable TLS for communication

---

# Unit (v1alpha2)

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
| `failedPodRecoveryPolicy` | FailedPodRecoveryPolicy | No | - | Failed pod recovery policy |

#### FailedPodRecoveryPolicy

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Enable/disable failed pod recovery |
| `reconcileThreshold` | int | No | Threshold for failed pod recovery |

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
| `conditions` | []metav1.Condition | Array of conditions representing the observed state |
| `observedGeneration` | int64 | Most recent generation observed |
| `phase` | UnitPhase | Lifecycle state of the unit |
| `nodeName` | string | Node where the unit is scheduled |
| `hostIP` | string | Host IP address |
| `podIPs` | []PodIP | Pod IP addresses |
| `nodeReady` | string | Node readiness state |
| `task` | string | Current task handled by operator |
| `processState` | string | Current process state inside operator |
| `configSynced` | ConfigSyncStatus | Configuration synchronization status |
| `persistentVolumeClaim` | []PvcInfo | PVC status information |

### UnitPhase Enum (matches code)

| Phase | Description |
|-------|-------------|
| `Pending` | Unit accepted but not yet started |
| `Running` | Unit is running and healthy |
| `Ready` | Unit is ready to serve traffic |
| `Succeeded` | Unit completed successfully |
| `Failed` | Unit failed to start or crashed |
| `Unknown` | Unit state cannot be determined |  
> Note: `Unknown` is marked as deprecated in comments (kept in code for compatibility).

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
                  key: mysql
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
            claimName: mysql-cluster-0-data
        - name: log
          emptyDir:
            sizeLimit: 5Gi
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

# Project (v1alpha2)

## Overview

Project is a cluster-scoped resource that defines project-level configuration and resources. It manages shared CA certificates and can be used to establish project-level security boundaries.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Project
metadata:
  name: my-project
spec:
  ca:
    enabled: true
    commonName: my-project-ca
    secretName: my-project-ca-secret
    duration: 87600h
    renewBefore: 720h
    privateKey:
      algorithm: ECDSA
      size: 256
```

## Specification (Spec)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ca` | CAInfo | No | Certificate Authority configuration |

### CAInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable CA configuration (default: false) |
| `commonName` | string | No | Common name for CA certificate |
| `secretName` | string | No | Kubernetes secret storing the CA |
| `duration` | string | No | Validity period (e.g., "87600h") |
| `renewBefore` | string | No | Renewal time before expiration |
| `privateKey` | PrivateKeyInfo | No | Private key configuration |

### PrivateKeyInfo

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `algorithm` | string | No | Cryptographic algorithm (RSA, ECDSA, Ed25519) |
| `size` | int | No | Private key size in bits |

## Status

Project currently has no status fields defined (empty status).

## Usage Examples

### Basic Project with CA
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Project
metadata:
  name: production-project
spec:
  ca:
    enabled: true
    commonName: production-ca
    secretName: production-ca-secret
    duration: 87600h
    renewBefore: 720h
    privateKey:
      algorithm: ECDSA
      size: 256
```

### Project without CA
```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: Project
metadata:
  name: development-project
spec: {}
```

## Best Practices

- **Cluster Scope**: Project is cluster-scoped, use meaningful names to identify environments
- **CA Management**: Store CA secrets securely and rotate periodically
- **Algorithm Selection**: ECDSA is recommended for new deployments

---

# MySQL Restore Example

## Overview

This example demonstrates how to restore a MySQL database from a backup using GrpcCall.

## Resource Definition

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-restore
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: restore
  ttlSecondsAfterFinished: 7200
  parameters:
    source: "s3://backups/mysql/backup-20240101.tar.gz"
    targetDatabase: "restored_db"
    overwrite: "false"
```

## Usage Examples

### Full Restore from S3
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-full-restore
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: restore
  ttlSecondsAfterFinished: 7200
  parameters:
    source: "s3://backups/mysql/backup-20240101.tar.gz"
    targetDatabase: "production_db"
    overwrite: "true"
    encryption: "AES256"
```

### Point-in-Time Recovery
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-pitr
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: restore
  ttlSecondsAfterFinished: 7200
  parameters:
    source: "s3://backups/mysql/"
    targetDatabase: "pitr_db"
    backupType: "incremental"
    pointInTime: "2024-01-01T12:00:00Z"
```

## Best Practices

### Restore Configuration
- **Verify Backup**: Ensure the backup source is accessible and valid
- **Target Database**: Create target database before restore if needed
- **Overwrite**: Set `overwrite: true` only when you want to replace existing data

### High Availability
- **Restore to Standby**: Consider restoring to a standby instance first
- **Verify Data**: After restore, verify data integrity before switching

---

# Common Use Cases

## Database Cluster Deployment

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
```

## Database Backup Operations

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

### MySQL Physical Backup
```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: mysql-physical-backup
  namespace: default
spec:
  targetUnit: mysql-cluster-0
  type: mysql
  action: physical-backup
  ttlSecondsAfterFinished: 3600
  parameters:
    backupType: "full"
    compression: "true"
    destination: "s3://backups/mysql/"
    parallel: "4"
```

## Configuration Management

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

## Database Cloning

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

# Advanced Configuration

## Labels and Annotations

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

## Networking Configuration

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

## Monitoring and Observability

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

## Security Configuration

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

# Troubleshooting

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

# Check MySQL replication status
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW REPLICA STATUS\\G"
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
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW ENGINE INNODB STATUS"
kubectl exec -it mysql-cluster-0 -- mysql -e "SHOW REPLICA STATUS\\G"
```

---

# Additional Resources

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