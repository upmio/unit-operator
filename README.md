# Unit Operator

[![Go Report Card](https://goreportcard.com/badge/github.com/upmio/unit-operator)](https://goreportcard.com/report/github.com/upmio/unit-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/upmio/unit-operator/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/upmio/unit-operator)](https://github.com/upmio/unit-operator/releases)
[![Stars](https://img.shields.io/github/stars/upmio/unit-operator)](https://github.com/upmio/unit-operator)

<div align="center">
  <img src="https://raw.githubusercontent.com/kubernetes/kubernetes/master/logo/logo.png" alt="Kubernetes Operator" width="120" height="120">
  <h3>Unit Operator - Distributed Workload Operator for Kubernetes</h3>
  <p>Manage UnitSet, Unit, and GrpcCall resources with built-in high availability, scaling, and lifecycle management capabilities</p>
</div>

## ✨ Features

- 🎯 **UnitSet Management**: Manages sets of distributed units with shared configuration
- 📦 **Unit Lifecycle**: Individual workload instance management with sidecar agents
- 📞 **gRPC Operations**: Manages gRPC-based operations between units
- 🛡️ **High Availability**: Built-in replication and failover mechanisms
- 📈 **Scaling**: Horizontal and vertical scaling capabilities
- 🔄 **Lifecycle Management**: Automated backup, recovery, and upgrades
- ⚙️ **Configuration Management**: Template-based configuration with shared configs
- 📊 **Monitoring**: Integrated metrics and health checks
- 🔐 **Security**: Certificate management and secure credential handling

## 🏗️ Architecture

<div align="center">
  <img src="https://img.icons8.com/color/96/kubernetes.png" alt="Kubernetes" width="48" height="48">
  <img src="https://img.icons8.com/color/96/database.png" alt="Database" width="48" height="48">
  <img src="https://img.icons8.com/color/96/infinity.png" alt="Loop" width="48" height="48">
</div>

The Unit Operator follows a two-layer architecture:

```
┌─────────────────────────────────────┐
│           UnitSet                  │
│      (Cluster Manager)             │
├─────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  │
│  │   Unit-0    │  │   Unit-1    │  │
│  │ (Pod+Agent) │  │ (Pod+Agent) │  │
│  └─────────────┘  └─────────────┘  │
│                                     │
│  ┌─────────────┐  ┌─────────────┐  │
│  │   Unit-2    │  │    ...      │  │
│  │ (Pod+Agent) │  │             │  │
│  └─────────────┘  └─────────────┘  │
└─────────────────────────────────────┘
```

- 🎯 **UnitSet**: Manages a set of distributed units with shared configuration
- 📦 **Unit**: Individual workload instances with sidecar agents for advanced operations
- 📞 **GrpcCall**: Manages gRPC-based operations between units
- 🤖 **Agent**: Sidecar container providing unit-specific operations and configuration management

## 📋 Prerequisites

- ☸️ **Kubernetes**: v1.27+ or OpenShift v4.6+
- 🐹 **Go**: 1.23+ (for development)
- ⚓ **Helm**: 3.0+ (for deployment)

## 🚀 Quick Start

### 📦 Installation

1. **Install using Helm**:

```bash
# Add the Helm repository
helm repo add upm-charts https://upmio.github.io/helm-charts

# Install the operator
helm install unit-operator --namespace upm-system --create-namespace \
  --set crd.enabled=true \
  upm-charts/unit-operator
```

2. **Install CRDs manually** (recommended for production):

```bash
kubectl apply -f config/crd/bases/
```

### 🐳 Example: Deploy UnitSet with gRPC Communication

```yaml
# Create shared configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: unitset-config
  namespace: default
data:
  service_group_uid: "6fa3ca2a-0ffd-4ca7-8615-e2589f7dd413"
  mysql_ports: '[{"name": "grpc", "containerPort": "50051","protocol": "TCP"}]'

# Deploy UnitSet
apiVersion: upm.syntropycloud.io/v1alpha2
kind: UnitSet
metadata:
  name: example-unitset
  namespace: default
spec:
  type: mysql
  version: "1.0.0"
  units: 3
  sharedConfigName: unitset-config
  resources:
    limits:
      cpu: "1"
      memory: 2Gi
    requests:
      cpu: "500m"
      memory: 1Gi
  storages:
    - name: data
      mountPath: /DATA_MOUNT
      size: 10Gi
      storageClassName: standard
  env:
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
```

### 📞 Example: Create gRPC Call

```yaml
apiVersion: upm.syntropycloud.io/v1alpha1
kind: GrpcCall
metadata:
  name: example-grpc-call
  namespace: default
spec:
  targetUnit: "example-unitset-0"
  type: mysql
  action: set-variable
  ttlSecondsAfterFinished: 600
  parameters:
    variables:
      ping: "true"
```

### ✅ Verify Deployment

```bash
# Check UnitSet status
kubectl get unitset example-unitset

# Check individual units
kubectl get units

# Check pod status
kubectl get pods -l app=example-unitset
```

## ⚙️ Configuration

### 🎯 Supported Resource Types

| Resource | 📊 API Version | 📝 Description |
|-----------|-------------|-------------------|
| UnitSet | v1alpha2 | Manages a set of distributed units |
| Unit | v1alpha2 | Individual workload instances |
| GrpcCall | v1alpha1 | Manages gRPC-based operations |

### 📞 gRPC Communication

The operator supports gRPC communication between units:
- Service discovery and registration
- Health checks and monitoring
- Configuration synchronization
- Cross-unit communication

### 💾 Storage Configuration

```yaml
# Persistent storage
storages:
  - name: data
    mountPath: /DATA_MOUNT
    size: 10Gi
    storageClassName: standard

# Temporary storage
emptyDir:
  - name: temp
    mountPath: /TEMP_MOUNT
    size: 1Gi
```

### 🔄 Update Strategy

```yaml
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 1
    partition: 0
```

## 💻 Development

### 🛠️ Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/upmio/unit-operator.git
cd unit-operator

# Install dependencies
go mod download

# Install required tools
make install-tools
```

### 🏗️ Build and Run

```bash
# Build the operator
make build

# Run locally
make run

# Run tests
make test

# Run with coverage
make check-coverage
```

### 🔧 Code Generation

```bash
# Generate CRDs and manifests
make manifests

# Generate deepcopy methods
make generate

# Generate protobuf code
make pb-gen
```

## 📚 API Reference

The Unit Operator provides the following custom resources:

- 🎯 [UnitSet](doc/unit-operator_api.md#unitset): Manages a cluster of database instances
- 📦 [Unit](doc/unit-operator_api.md#unit): Individual database instance
- 📞 [GrpcCall](doc/unit-operator_api.md#grpccall): gRPC-based operations

## 📊 Monitoring

The operator exposes metrics on port `20154` by default:

```bash
# Access metrics
kubectl port-forward -n upm-system svc/unit-operator-metrics 20154:20154
curl http://localhost:20154/metrics
```

<div align="center">
  <img src="https://img.icons8.com/color/96/000000/analytics.png" alt="Monitoring" width="48" height="48">
  <p>Monitor your database clusters with built-in metrics</p>
</div>

## 🚨 Troubleshooting

### ⚠️ Common Issues

1. **Pods stuck in Pending state**
   - Check resource requests/limits
   - Verify storage class availability
   - Ensure sufficient cluster resources

2. **Replication not working**
   - Verify network connectivity between pods
   - Check credential secrets
   - Review replication configuration

3. **Upgrade failures**
   - Check operator logs for errors
   - Verify upgrade strategy configuration
   - Ensure sufficient disk space

### 🔍 Debug Commands

```bash
# Check operator logs
kubectl logs -n upm-system deployment/unit-operator

# Check UnitSet events
kubectl describe unitset <name>

# Check individual Unit status
kubectl describe unit <name>

# Check agent logs
kubectl logs <pod-name> -c agent
```

## 🤝 Contributing

We welcome contributions! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### 🔄 Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run linting and tests
6. Submit a pull request

### 🎨 Code Style

- Follow Go standard formatting
- Use `make fmt` and `make vet` before committing
- Ensure tests pass with `make test`
- Maintain test coverage above threshold

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📚 [Documentation](doc/unit-operator_api.md)
- 🐛 [Issues](https://github.com/upmio/unit-operator/issues)
- 💬 [Discussions](https://github.com/upmio/unit-operator/discussions)

## 🙏 Acknowledgments

<div align="center">
  <p>
    <img src="https://img.icons8.com/color/96/000000/github.png" alt="GitHub" width="32" height="32">
    Built with ❤️ using these amazing tools and frameworks
  </p>
</div>

- 🏗️ [Kubebuilder](https://book.kubebuilder.io/) - Kubernetes API development framework
- 🎛️ [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller framework
- 🐘 [Zalando Postgres Operator](https://github.com/zalando/postgres-operator) - Inspiration for PostgreSQL management
- 🐬 [Presslabs MySQL Operator](https://github.com/presslabs/mysql-operator) - Inspiration for MySQL management

---

<div align="center">
  <p>
    <img src="https://img.icons8.com/color/96/000000/kubernetes.png" alt="Kubernetes" width="32" height="32">
    Made with ❤️ by the Unit Operator community
  </p>
</div>