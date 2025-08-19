# API Reference

## Packages
- [upm.syntropycloud.io/v1alpha2](#upmsyntropycloudiov1alpha2)
- [upm.syntropycloud.io/v1alpha1](#upmsyntropycloudiov1alpha1)


## upm.syntropycloud.io/v1alpha2

Package v1alpha2 contains API Schema definitions for the  v1alpha2 API group

### Resource Types
- [Unit](#unit)
- [UnitList](#unitlist)
- [UnitSet](#unitset)
- [UnitSetList](#unitsetlist)

> Webhooks: `UnitSet`/`Unit` admission webhooks are enabled by default (can be disabled via `ENABLE_WEBHOOKS=false`). `UnitSet` attaches finalizers (`upm.io/unit-delete`, `upm.io/configmap-delete`) during defaulting.

## upm.syntropycloud.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the v1alpha1 API group

### Resource Types
- [GrpcCall](#grpccall)
- [GrpcCallList](#grpccallist)

> Note: Replication/topology CRDs (e.g., `MysqlReplication`, `PostgresReplication`) are provided by the Compose Operator project, not this repository. See: `https://github.com/upmio/compose-operator`.


#### Action

Underlying type: _string_

Action defines the specific operation to be sent to the unit-agent. Enum: `logical-backup`, `physical-backup`, `restore`, `gtid-purge`, `set-variable`, `clone`.

_Appears in:_
- [GrpcCallSpec](#grpccallspec)


#### GrpcCall

GrpcCall is the Schema for the grpccalls API

_Appears in:_
- [GrpcCallList](#grpccallist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha1` | | |
| `kind` _string_ | `GrpcCall` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GrpcCallSpec](#grpccallspec)_ |  |  |  |
| `status` _[GrpcCallStatus](#grpccallstatus)_ |  |  |  |


#### GrpcCallList

GrpcCallList contains a list of GrpcCall

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha1` | | |
| `kind` _string_ | `GrpcCallList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[GrpcCall](#grpccall) array_ |  |  |  |


#### GrpcCallSpec

GrpcCallSpec defines the desired behavior of a GrpcCall custom resource.

_Appears in:_
- [GrpcCall](#grpccall)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `targetUnit` _string_ | Name of the target Unit custom resource |  |  |
| `type` _[UnitType](#unittype)_ | Type of target unit |  | Enum: `mysql`, `postgresql`, `proxysql` |
| `action` _[Action](#action)_ | Operation to perform |  | Enum: `logical-backup`, `physical-backup`, `restore`, `gtid-purge`, `set-variable`, `clone` |
| `ttlSecondsAfterFinished` _integer_ | TTL after completion (seconds). If set, the resource is eligible for auto-deletion after TTL. |  |  |
| `parameters` _object_ | Action-specific parameters (map[string]JSON) |  | Schemaless: {} <br /> |


#### GrpcCallStatus

GrpcCallStatus defines the observed state of a GrpcCall.

_Appears in:_
- [GrpcCall](#grpccall)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `result` _[Result](#result)_ | Final outcome of the operation |  | Enum: `Success`, `Failed` |
| `message` _string_ | Detailed message or error context |  |  |
| `completionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Completion time |  |  |
| `startTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Start time |  |  |


#### Result

Underlying type: _string_

Result defines the outcome status of a GrpcCall execution. Enum: `Success`, `Failed`.

_Appears in:_
- [GrpcCallStatus](#grpccallstatus)


#### UnitType

Underlying type: _string_

UnitType defines the type of unit the GrpcCall will interact with. Enum: `mysql`, `postgresql`, `proxysql`.

_Appears in:_
- [GrpcCallSpec](#grpccallspec)



#### CertificateSecretSpec



CertificateSecretSpec defines the configuration for certificate secrets.



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organization` _string_ | Organization name for the certificate |  |  |
| `name` _string_ | Name of the certificate secret |  |  |

#### CertificateProfile

CertificateProfile contains certificate profile information.

_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organizations` _string array_ | List of organization names |  |  |
| `root_secret` _string_ | Root secret name (CA) |  |  |


#### ConfigSyncStatus







_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | LastTransitionTime the last transition time |  |  |


#### EmptyDirSpec







_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the storage |  |  |
| `size` _string_ | Size of the storage |  |  |
| `mountPath` _string_ | MountPath Mount path |  |  |


#### ExternalServiceSpec



ExternalServiceSpec defines the configuration for external services.



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of the external service (e.g., NodePort) |  |  |


#### NodeAffinityPresetSpec



NodeAffinityPresetSpec defines node affinity rules.



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ | Key for the node affinity |  |  |
| `values` _string array_ | Values for the node affinity |  |  |


#### PortInfo







_Appears in:_
- [Ports](#ports)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `containerPort` _string_ |  |  |  |
| `protocol` _string_ |  |  |  |




#### PvcCapacity



PvcCapacity represents the actual resources of the PVC.



_Appears in:_
- [PvcInfo](#pvcinfo)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storage` _[Quantity](#quantity)_ | Storage represents the actual resources of the PVC. |  |  |


#### PvcInfo



PvcInfo represents the current information/status of a persistent volume claim.



_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name name of a persistent volume claim. |  |  |
| `volumeName` _string_ | VolumeName name of volume |  |  |
| `accessModes` _[PersistentVolumeAccessMode](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeaccessmode-v1-core) array_ | AccessModes contains the actual access modes the volume backing the PVC has.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1 |  |  |
| `capacity` _[PvcCapacity](#pvccapacity)_ | Capacity represents the actual resources of the PVC. |  |  |
| `phase` _[PersistentVolumeClaimPhase](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimphase-v1-core)_ | Phase represents the current phase of PersistentVolumeClaim. |  |  |


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
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSpec](#unitspec)_ |  |  |  |
| `status` _[UnitStatus](#unitstatus)_ |  |  |  |


#### UnitList



UnitList contains a list of Unit





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `UnitList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Unit](#unit) array_ |  |  |  |


#### UnitPhase

_Underlying type:_ _string_

UnitPhase is a label for the condition of a pod at the current time.



_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description |
| `Pending` | UnitPending means the pod has been accepted by the system, but one or more of the containers<br />has not been started. This includes time before being bound to a node, as well as time spent<br />pulling images onto the host.<br /> |
| `Running` | UnitRunning means the pod has been bound to a node and all of the containers have been started.<br />At least one container is still running or is in the process of being restarted.<br /> |
| `Ready` | UnitReady means the pod Running and ready condition = true<br /> |
| `Succeeded` | UnitSucceeded means that all containers in the pod have voluntarily terminated<br />with a container exit code of 0, and the system is not going to restart any of these containers.<br /> |
| `Failed` | UnitFailed means that all containers in the pod have terminated, and at least one container has<br />terminated in a failure (exited with a non-zero exit code or was stopped by the system).<br /> |
| `Unknown` | UnitUnknown means that for some reason the state of the pod could not be obtained, typically due<br />to an error in communicating with the host of the pod.<br />Deprecated: It isn't being set since 2015 (74da3b14b0c0f658b3bb8d2def5094686d0e9095)<br /> |

#### UnitStatus

_Appears in:_
- [Unit](#unit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | Conditions is an array of conditions. |  |  |
| `phase` _[UnitPhase](#unitphase)_ | Current lifecycle phase |  |  |
| `nodeReady` _string_ | Node readiness state |  |  |
| `nodeName` _string_ | Node name of the unit |  |  |
| `task` _string_ | Current task |  |  |
| `processState` _string_ | Current process state |  |  |
| `hostIP` _string_ | Host IP |  |  |
| `podIPs` _[PodIP](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podip-v1-core) array_ | Pod IPs |  |  |
| `configSynced` _[ConfigSyncStatus](#configsyncstatus)_ | Config sync status |  |  |
| `persistentVolumeClaim` _[PvcInfo](#pvcinfo) array_ | PVC status info |  |  |


#### UnitServiceSpec



UnitServiceSpec defines the configuration for unit services.



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of the unit service (e.g., ClusterIP) |  |  |


#### UnitSet



UnitSet is the Schema for the unitsets API



_Appears in:_
- [UnitSetList](#unitsetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `upm.syntropycloud.io/v1alpha2` | | |
| `kind` _string_ | `UnitSet` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSetSpec](#unitsetspec)_ |  |  |  |
| `status` _[UnitSetStatus](#unitsetstatus)_ |  |  |  |


##### Scheduling via Annotation
Use `metadata.annotations.upm.io/node-name-map` on `UnitSet` to define per-Unit node bindings.
Example:
```
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
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[UnitSet](#unitset) array_ |  |  |  |


#### UnitSetSpec



UnitSetSpec defines the desired state of UnitSet



_Appears in:_
- [UnitSet](#unitset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is the type of the unitset |  |  |
| `edition` _string_ | Edition of the unit set |  |  |
| `version` _string_ | Version of the unit set |  |  |
| `units` _integer_ | Units Number of units in the unitset |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | Resources Resource requirements for the units |  | Schemaless: {} <br /> |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ | Env Environment variables for the units |  | Schemaless: {} <br /> |
| `externalService` _[ExternalServiceSpec](#externalservicespec)_ | ExternalService Configuration for external services |  |  |
| `unitService` _[UnitServiceSpec](#unitservicespec)_ | UnitService Configuration for unit services |  |  |
| `certificateSecret` _[CertificateSecretSpec](#certificatesecretspec)_ | CertificateSecret Secret configuration for certificates |  |  |
| `certificateProfile` _[CertificateProfile](#certificateprofile)_ | Additional CA and org settings |  |  |
| `sharedConfigName` _string_ | SharedConfigName Name of the shared configuration |  |  |
| `updateStrategy` _[UpdateStrategySpec](#updatestrategyspec)_ | UpdateStrategy Strategy for updating the unit set |  |  |
| `nodeAffinityPreset` _[NodeAffinityPresetSpec](#nodeaffinitypresetspec) array_ | NodeAffinityPreset  Node affinity rules |  |  |
| `podAntiAffinityPreset` _string_ | PodAntiAffinityPreset Pod anti-affinity policy |  |  |
| `storages` _[StorageSpec](#storagespec) array_ | Storages defines the configuration for storage |  |  |
| `emptyDir` _[EmptyDirSpec](#emptydirspec) array_ | EmptyDir defines the configuration for emptyDir |  |  |
| `secret` _[SecretInfo](#secretinfo)_ | Secret defines the configuration for secret |  |  |




#### UnitSetStatus

UnitSetStatus defines the observed state of UnitSet

_Appears in:_
- [UnitSet](#unitset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#condition-v1-meta) array_ | Conditions |  |  |
| `units` _integer_ | Current number of units |  |  |
| `readyUnits` _integer_ | Number of ready units |  |  |
| `unitPVCSynced` _[PvcSyncStatus](#pvcsyncstatus)_ | PVC synchronization status |  |  |
| `unitImageSynced` _[ImageSyncStatus](#imagesyncstatus)_ | Image synchronization status |  |  |
| `unitResourceSynced` _[ResourceSyncStatus](#resourcesyncstatus)_ | Resource synchronization status |  |  |
| `inUpdate` _string_ | Whether an update is in progress |  |  |

#### PvcSyncStatus

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Last transition time |  |  |
| `status` _string_ | True/False |  |  |

#### ImageSyncStatus

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Last transition time |  |  |
| `status` _string_ | True/False |  |  |

#### ResourceSyncStatus

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | Last transition time |  |  |
| `status` _string_ | True/False |  |  |
#### UnitSpec



UnitSpec defines the desired state of Unit



_Appears in:_
- [Unit](#unit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `unbindNode` _boolean_ | UnBindNode defines whether the unit is bound to a node or not.<br />if false: when pod scheduled ok, write pod.Spec.NodeName<br />to 'unit.annotations[last.unit.belong.node]' and 'unit.Spec.Template.Spec.NodeName' |  |  |
| `startup` _boolean_ | Startup defines whether the service is started or not |  |  |
| `sharedConfigName` _string_ | SharedConfigName defines the shared config name<br />derived from the same name field in unitset.<br />unit has no logic and is only used as a parameter when calling unit agent |  |  |
| `configTemplateName` _string_ | ConfigTemplateName defines the config template name.<br />A unitset is instantiated as a config template for the unitset<br />by copying the corresponding version template.<br />one for a set of unitsets.<br />The unitset is then assigned a value to the field.<br />unitset is not processed logically<br />and is passed as a parameter when the unit agent is called. |  |  |
| `configValueName` _string_ | ConfigValueName defines the config value name.<br />unitset instantiates one for each unit by copying the corresponding version template.<br />The value is then assigned to the field.<br />unitset does no logical processing<br />and is passed as a parameter in the call to the unit agent |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaim-v1-core) array_ | VolumeClaimTemplates is a user's request for and claim to a persistent volume |  | Schemaless: {} <br /> |
| `template` _[PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)_ | Template describes the data a pod should have when created from a template |  | Schemaless: {} <br /> |




#### UpdateStrategySpec



UpdateStrategySpec defines the update strategy for the unit set.



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of update strategy (e.g., RollingUpdate) |  |  |
| `rollingUpdate` _[RollingUpdateSpec](#rollingupdatespec)_ | RollingUpdate Rolling update configuration |  |  |


