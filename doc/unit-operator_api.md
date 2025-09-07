# API Reference

## Packages

- [upm.syntropycloud.io/v1alpha1](#upmsyntropycloudiov1alpha1)
- [upm.syntropycloud.io/v1alpha2](#upmsyntropycloudiov1alpha2)

## upm.syntropycloud.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the v1alpha1 API group

### v1alpha1 Resource Types

- [GrpcCall](#grpccall)
- [GrpcCallList](#grpccalllist)

> Note: Replication/topology CRDs (e.g., `MysqlReplication`, `PostgresReplication`)
> are provided by the Compose Operator project, not this repository.
> See: `https://github.com/upmio/compose-operator`.

#### Action

Underlying type: _string_

Action defines the specific operation to be sent to the unit-agent.
Each action corresponds to a gRPC method exposed by the unit-agent.
Enum: `logical-backup`, `physical-backup`, `restore`, `gtid-purge`, `set-variable`, `clone`.

_Appears in:_

- [GrpcCallSpec](#grpccallspec)

#### GrpcCall

GrpcCall is the Schema for the grpccalls API

_Appears in:_

- [GrpcCallList](#grpccalllist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha1` | | |
| `kind` _string_ | `GrpcCall` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GrpcCallSpec](#grpccallspec)_ |  |  |  |
| `status` _[GrpcCallStatus](#grpccallstatus)_ |  |  |  |

#### GrpcCallList

GrpcCallList contains a list of GrpcCall

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha1` | | |
| `kind` _string_ | `GrpcCallList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[GrpcCall](#grpccall) array_ |  |  |  |

#### GrpcCallSpec

GrpcCallSpec defines the desired behavior of a GrpcCall custom resource.

_Appears in:_

- [GrpcCall](#grpccall)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `targetUnit` _string_ | Name of the target Unit custom resource |  | Required: ✓ |
| `type` _[UnitType](#unittype)_ | Type of target unit |  | Required: ✓, Enum: `mysql`, `postgresql`, `proxysql` |
| `action` _[Action](#action)_ | Operation to perform |  | Required: ✓, Enum: `logical-backup`, `physical-backup`, `restore`, `gtid-purge`, `set-variable`, `clone` |
| `ttlSecondsAfterFinished` _integer_ | TTL after completion (seconds). If set, the resource is eligible for auto-deletion after TTL. |  | Required: ✓ |
| `parameters` _object_ | Action-specific parameters (map[string]JSON) |  | Required: ✓, Schemaless: {} |

#### GrpcCallStatus

GrpcCallStatus defines the observed state of a GrpcCall.

_Appears in:_

- [GrpcCall](#grpccall)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `result` _[Result](#result)_ | Final outcome of the operation |  | Required: ✓, Enum: `Success`, `Failed` |
| `message` _string_ | Detailed message or error context |  | Required: ✓ |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#time-v1-meta)_ | Completion time |  |  |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#time-v1-meta)_ | Start time |  |  |

#### Result

Underlying type: _string_

Result defines the outcome status of a GrpcCall execution.
It represents the final state of the gRPC request sent to the unit-agent.
Enum: `Success`, `Failed`.

_Appears in:_

- [GrpcCallStatus](#grpccallstatus)

#### UnitType

Underlying type: _string_

UnitType defines the type of unit this GrpcCall will interact with.
Currently supported types are "mysql", "proxysql" and "postgresql".
Enum: `mysql`, `postgresql`, `proxysql`.

_Appears in:_

- [GrpcCallSpec](#grpccallspec)

## upm.syntropycloud.io/v1alpha2

Package v1alpha2 contains API Schema definitions for the  v1alpha2 API group

### v1alpha2 Resource Types

- [Project](#project)
- [ProjectList](#projectlist)
- [Unit](#unit)
- [UnitList](#unitlist)
- [UnitSet](#unitset)
- [UnitSetList](#unitsetlist)

> Webhooks: `UnitSet`/`Unit` admission webhooks are enabled by default
> (can be disabled via `ENABLE_WEBHOOKS=false`). `UnitSet` attaches finalizers
> (`upm.io/unit-delete`, `upm.io/configmap-delete`) during defaulting.

#### CertificateProfile

CertificateProfile contains certificate profile information.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organizations` _string array_ | List of organization names |  |  |
| `root_secret` _string_ | Root secret name (CA) |  |  |

#### CertificateSecretSpec

CertificateSecretSpec defines the configuration for certificate secrets.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organization` _string_ | Organization name for the certificate |  |  |
| `name` _string_ | Name of the certificate secret |  |  |

#### EmptyDirSpec

EmptyDirSpec defines the configuration for emptyDir volumes

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the emptyDir volume. |
| `sizeLimit` _[Quantity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#quantity-resource-core)_ | SizeLimit is the total amount of local storage required for this EmptyDir volume. |

#### ExternalServiceSpec

ExternalServiceSpec defines the configuration for external services.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of the external service (e.g., NodePort) |  |  |

#### ExternalServiceStatus

ExternalServiceStatus defines the observed state of external service

_Appears in:_

- [UnitSetStatus](#unitsetstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the external service. |
| `ready` _boolean_ | Ready indicates whether the external service is ready. |
| `message` _string_ | Message provides additional information about the external service status. |

#### NodeAffinityPresetSpec

NodeAffinityPresetSpec defines node affinity rules.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ | Key for the node affinity |  |  |
| `values` _string array_ | Values for the node affinity |  |  |

#### PodMonitorInfo

PodMonitorInfo defines pod monitor information for monitoring

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description |
| --- | --- |
| `enabled` _boolean_ | Enabled indicates whether pod monitoring is enabled. |

#### Project

Project is the Schema for the projects API. Project is a Cluster-scoped resource.

_Appears in:_

- [ProjectList](#projectlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `Project` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProjectSpec](#projectspec)_ |  |  |  |
| `status` _[ProjectStatus](#projectstatus)_ |  |  |  |

#### ProjectList

ProjectList contains a list of Project

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `ProjectList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Project](#project) array_ |  |  |  |

#### ProjectSpec

ProjectSpec defines the desired state of Project. ProjectSpec is an empty object.

_Appears in:_

- [Project](#project)

#### ProjectStatus

ProjectStatus defines the observed state of Project. ProjectStatus is an empty object.

_Appears in:_

- [Project](#project)

#### RollingUpdateSpec

RollingUpdateSpec defines the rolling update configuration.

_Appears in:_

- [UpdateStrategySpec](#updatestrategyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `partition` _integer_ | Partition Number of partitions for the update |  |  |
| `maxUnavailable` _integer_ | MaxUnavailable Maximum number of unavailable pods during update |  |  |

#### SecretInfo

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the secret |  |  |
| `mountPath` _string_ | MountPath Mount path of the secret |  |  |

#### StorageSpec

StorageSpec defines the configuration for storage.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the storage |  |  |
| `size` _string_ | Size of the storage |  |  |
| `storageClassName` _string_ | StorageClassName storage class name |  |  |
| `mountPath` _string_ | MountPath Mount path |  |  |

#### Unit

Unit is the Schema for the units API

_Appears in:_

- [UnitList](#unitlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `Unit` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSpec](#unitspec)_ |  |  |  |
| `status` _[UnitStatus](#unitstatus)_ |  |  |  |

#### UnitList

UnitList contains a list of Unit

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `UnitList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Unit](#unit) array_ |  |  |  |

#### UnitPhase

Underlying type: _string_

UnitPhase is a label for the condition of a pod at the current time.

_Appears in:_

- [UnitStatus](#unitstatus)

| Value | Description |
| --- | --- |
| `Pending` | UnitPending means the pod has been accepted by the system, but one or more of the containers has not been started. This includes time before being bound to a node, as well as time spent pulling images onto the host. |
| `Running` | UnitRunning means the pod has been bound to a node and all of the containers have been started. At least one container is still running or is in the process of being restarted. |
| `Ready` | UnitReady means the pod Running and ready condition = true |
| `Succeeded` | UnitSucceeded means that all containers in the pod have voluntarily terminated with a container exit code of 0, and the system is not going to restart any of these containers. |
| `Failed` | UnitFailed means that all containers in the pod have terminated, and at least one container has terminated in a failure (exited with a non-zero exit code or was stopped by the system). |
| `Unknown` | UnitUnknown means that for some reason the state of the pod could not be obtained, typically due to an error in communicating with the host of the pod. Deprecated: It isn't being set since 2015 (74da3b14b0c0f658b3bb8d2def5094686d0e9095) |

#### UnitStatus

UnitStatus defines the observed state of Unit

_Appears in:_

- [Unit](#unit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#condition-v1-meta) array_ | Conditions represent the latest available observations of a unit's current state |  |  |
| `configSynced` _boolean_ | ConfigSynced indicates whether the configuration has been synchronized |  |  |
| `hostIP` _string_ | HostIP is the IP address of the host where the Unit's Pod is running |  |  |
| `nodeName` _string_ | NodeName is the name of the node where the Unit's Pod is running |  |  |
| `nodeReady` _boolean_ | NodeReady indicates whether the node is ready |  |  |
| `persistentVolumeClaim` _[PersistentVolumeClaimStatus](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#persistentvolumeclaimstatus-v1-core) array_ | PersistentVolumeClaim represents the status of persistent volume claims |  |  |
| `phase` _[UnitPhase](#unitphase)_ | Phase is the current phase of the Unit |  |  |
| `podIPs` _[PodIP](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podip-v1-core) array_ | PodIPs holds the IP addresses allocated to the pod |  |  |
| `processState` _string_ | ProcessState represents the current state of the process |  |  |
| `task` _string_ | Task represents the current task being executed |  |  |

#### UnitServiceSpec

UnitServiceSpec defines the specification for unit-specific services

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the unit service. |
| `type` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#servicetype-v1-core)_ | Type determines how the Service is exposed. |
| `ports` _[ServicePort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.31/#serviceport-v1-core) array_ | Ports is the list of ports that are exposed by this service. |
| `selector` _object (keys:string, values:string)_ | Selector is a map of {key,value} pairs. |

#### UnitServiceStatus

UnitServiceStatus defines the observed state of unit service

_Appears in:_

- [UnitSetStatus](#unitsetstatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the unit service. |
| `ready` _boolean_ | Ready indicates whether the unit service is ready. |
| `message` _string_ | Message provides additional information about the unit service status. |

#### UnitSet

UnitSet is the Schema for the unitsets API

_Appears in:_

- [UnitSetList](#unitsetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `UnitSet` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSetSpec](#unitsetspec)_ |  |  |  |
| `status` _[UnitSetStatus](#unitsetstatus)_ |  |  |  |

##### Scheduling via Annotation

Use `metadata.annotations.upm.io/node-name-map` on `UnitSet` to define per-Unit node bindings.
Example:

```yaml
metadata:
  annotations:
    upm.io/node-name-map: '{"mysql-cluster-0":"node-a","mysql-cluster-1":"noneSet"}'
```

Controller behavior:

- Adds nodeAffinity targeting the specified node for that Unit
- Writes `last.unit.belong.node` to the corresponding `Unit` annotation
- `noneSet` disables binding for that Unit

#### UnitSetList

UnitSetList contains a list of UnitSet

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `UnitSetList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[UnitSet](#unitset) array_ |  |  |  |

#### UnitSetSpec

UnitSetSpec defines the desired state of UnitSet

_Appears in:_

- [UnitSet](#unitset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `certificateProfile` _string_ | CertificateProfile defines certificate profile for this UnitSet |  |  |
| `edition` _string_ | Edition specifies the edition of the UnitSet |  |  |
| `emptyDir` _[EmptyDirSpec](#emptydirspec)_ | EmptyDir defines empty directory configuration |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envvar-v1-core) array_ | Env defines environment variables for the UnitSet |  |  |
| `externalService` _[ExternalServiceSpec](#externalservicespec)_ | ExternalService defines external service configuration |  |  |
| `nodeAffinityPreset` _[NodeAffinityPresetSpec](#nodeaffinitypresetspec)_ | NodeAffinityPreset defines node affinity preset for this UnitSet |  |  |
| `podAntiAffinityPreset` _string_ | PodAntiAffinityPreset defines pod anti-affinity preset |  |  |
| `podMonitor` _[PodMonitorInfo](#podmonitorinfo)_ | PodMonitor defines pod monitor information for this UnitSet |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#resourcerequirements-v1-core)_ | Resources defines resource requirements |  |  |
| `secret` _[SecretInfo](#secretinfo)_ | Secret defines secret information for this UnitSet |  |  |
| `storages` _[StorageSpec](#storagespec) array_ | Storages defines storage configuration for this UnitSet |  |  |
| `type` _string_ | Type specifies the type of the UnitSet |  |  |
| `unitService` _[UnitServiceSpec](#unitservicespec)_ | UnitService defines unit service configuration |  |  |
| `units` _integer_ | Units specifies the number of units in the UnitSet |  |  |
| `updateStrategy` _[UpdateStrategySpec](#updatestrategyspec)_ | UpdateStrategy defines the update strategy for the UnitSet |  |  |
| `version` _string_ | Version specifies the version of the UnitSet |  |  |

#### UnitSetStatus

UnitSetStatus defines the observed state of UnitSet

_Appears in:_

- [UnitSet](#unitset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#condition-v1-meta) array_ | Conditions represent the latest available observations of a UnitSet's current state |  |  |
| `externalService` _[ExternalServiceStatus](#externalservicestatus)_ | ExternalService represents the status of external service for this UnitSet |  |  |
| `inUpdate` _boolean_ | InUpdate indicates whether the UnitSet is currently being updated |  |  |
| `readyUnits` _integer_ | ReadyUnits is the number of ready units in the UnitSet |  |  |
| `unitImageSynced` _boolean_ | UnitImageSynced indicates whether unit images are synchronized |  |  |
| `unitPVCSynced` _boolean_ | UnitPVCSynced indicates whether unit PVCs are synchronized |  |  |
| `unitResourceSynced` _boolean_ | UnitResourceSynced indicates whether unit resources are synchronized |  |  |
| `unitService` _[UnitServiceStatus](#unitservicestatus)_ | UnitService represents the status of unit service for this UnitSet |  |  |
| `units` _integer_ | Units is the current number of units in the UnitSet |  |  |

#### UnitSpec

UnitSpec defines the desired state of Unit

_Appears in:_

- [Unit](#unit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `startup` _boolean_ | Startup indicates whether the unit should be started automatically |  |  |
| `configTemplateName` _string_ | ConfigTemplateName is the name of the config template |  |  |
| `configValueName` _string_ | ConfigValueName is the name of the config value |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#persistentvolumeclaim-v1-core) array_ | VolumeClaimTemplates is a list of claims that pods are allowed to reference |  |  |
| `template` _[PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podtemplatespec-v1-core)_ | Template is the object that describes the pod that will be created |  |  |

#### UpdateStrategySpec

UpdateStrategySpec defines the update strategy for the unit set.

_Appears in:_

- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of update strategy (e.g., RollingUpdate) |  |  |
| `rollingUpdate` _[RollingUpdateSpec](#rollingupdatespec)_ | RollingUpdate Rolling update configuration |  |  |
