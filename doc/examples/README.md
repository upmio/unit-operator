# Unit Operator Examples

This directory contains practical examples for using the Unit Operator to deploy and manage database workloads in Kubernetes.

## Example Structure

```text
examples/
├── unitsets/                    # UnitSet deployment examples
│   ├── mysql-community-rpl_semi_sync-unitset.yaml
│   ├── postgresql-replication-unitset.yaml
│   └── proxysql-clone-unitset.yaml
└── operations/                  # GrpcCall operation examples
    ├── mysql-backup.yaml
    └── postgresql-backup.yaml
```

## Quick Start Examples

### MySQL Cluster

Deploy a 3-node MySQL cluster:

```bash
# Deploy MySQL cluster
kubectl apply -f examples/unitsets/mysql-community-rpl_semi_sync-unitset.yaml

# Check deployment status
kubectl get unitset mysql-cluster
kubectl get units -l upm.api/service-group.name=mysql-cluster
kubectl get pods -l upm.api/service-group.name=mysql-cluster
```

### PostgreSQL Cluster

Deploy a 3-node PostgreSQL cluster:

```bash
# Deploy PostgreSQL cluster
kubectl apply -f examples/unitsets/postgresql-replication-unitset.yaml

# Check deployment status
kubectl get unitset postgresql-cluster
kubectl get units -l upm.api/service-group.name=postgresql-cluster
```

## Available Examples

### UnitSet Deployments

#### MySQL Community Edition

- **File**: `mysql-community-rpl_semi_sync-unitset.yaml`
- **Features**: 3-node cluster, persistent storage
- **Use Case**: Production MySQL deployment

#### PostgreSQL

- **File**: `postgresql-replication-unitset.yaml`
- **Features**: 3-node cluster, emptyDir storage
- **Use Case**: Production PostgreSQL deployment

#### ProxySQL Configuration

- **File**: `proxysql-clone-unitset.yaml`
- **Features**: Single ProxySQL instance
- **Use Case**: Database proxy and load balancing

### Database Operations

#### MySQL Backup Operations

- **File**: `operations/mysql-backup.yaml`
- **Features**: Logical and physical backup examples
- **Use Case**: Regular database backups

#### PostgreSQL Backup Operations

- **File**: `operations/postgresql-backup.yaml`
- **Features**: Physical backup with compression
- **Use Case**: PostgreSQL backup strategies

## Common Use Cases

### Production Database Deployment

```bash
# Deploy production MySQL cluster
kubectl apply -f examples/unitsets/mysql-community-rpl_semi_sync-unitset.yaml

# Configure backups
kubectl apply -f examples/operations/mysql-backup.yaml
```

### Development Environment Setup

```bash
# Deploy single-node PostgreSQL
kubectl apply -f examples/unitsets/postgresql-replication-unitset.yaml
```

## Customizing Examples

### Prerequisites

Before using the examples, ensure you have:

1. **Kubernetes cluster** (v1.29+)
2. **Unit Operator installed**
3. **Storage classes configured**
4. **Network policies** (optional)

### Customization Steps

1. **Update Secrets**: Replace placeholder secrets with actual credentials
2. **Adjust Resources**: Modify CPU, memory, and storage requirements
3. **Configure Networking**: Update service types and network policies
4. **Set Environment**: Adjust environment variables for your use case

### Example Customization

```yaml
# Original example
spec:
  resources:
    limits:
      cpu: "2"
      memory: 4Gi
    requests:
      cpu: "1"
      memory: 2Gi

# Customized for production
spec:
  resources:
    limits:
      cpu: "4"
      memory: 16Gi
    requests:
      cpu: "2"
      memory: 8Gi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 500Gi
      storageClassName: fast-ssd
```

## Example Templates

### Secret Template

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: database-secret
  namespace: default
data:
  # Base64 encoded passwords
  root: <base64-encoded-root-password>
  replication: <base64-encoded-replication-password>
  monitor: <base64-encoded-monitor-password>
type: Opaque
```

### ConfigMap Template

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: database-config
  namespace: default
data:
  service_group_uid: "<unique-identifier>"
  mysql_ports: '[{"name": "mysql", "containerPort": "3306","protocol": "TCP"}]'
  postgresql_ports: '[{"name": "postgresql", "containerPort": "5432","protocol": "TCP"}]'
```

### UnitSet Template

```yaml
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: <cluster-name>
  namespace: <namespace>
  labels:
    upm.api/service-group.name: <cluster-name>
    upm.api/service-group.type: <database-type>-sg
spec:
  type: <database-type>
  version: "<version>"
  units: <number-of-units>
  sharedConfigName: <config-name>
  # ... other configuration
```

## Important Notes

### Security Considerations

1. **Secret Management**: Always use Kubernetes secrets for credentials
2. **Network Security**: Implement network policies for database access
3. **TLS Encryption**: Enable TLS for production deployments
4. **Access Control**: Use RBAC for resource access control

### Best Practices

1. **Resource Planning**: Plan resources based on expected workload
2. **Backup Strategy**: Implement regular backup schedules
3. **Monitoring**: Set up comprehensive monitoring
4. **Testing**: Test all operations in non-production environments

### Production Readiness

1. **High Availability**: Use at least 3 nodes for production
2. **Disaster Recovery**: Have backup and restore procedures
3. **Performance Tuning**: Optimize configuration for workload
4. **Documentation**: Maintain deployment and operation documentation

## Updating Examples

The examples are regularly updated to:

1. **Support New Features**: Include new Unit Operator capabilities
2. **Security Updates**: Apply latest security best practices
3. **Performance Improvements**: Optimize configurations
4. **Bug Fixes**: Address reported issues

Always check for the latest examples when deploying new clusters.

## Additional Resources

- [API Documentation](../api-documentation.md) - Complete API reference
- [Quick Reference](../api-quick-reference.md) - Fast reference guide
- [Troubleshooting Guide](../README.md#troubleshooting) - Common issues and solutions
- [Best Practices](../README.md#best-practices) - Production deployment guidelines

---

![Kubernetes](https://img.icons8.com/color/48/000000/kubernetes.png) ![Database](https://img.icons8.com/color/48/000000/database.png) ![Code](https://img.icons8.com/color/48/000000/code.png)

Practical examples for database deployment and management
