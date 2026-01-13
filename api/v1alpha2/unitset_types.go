/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UnitSetSpec defines the desired state of UnitSet
type UnitSetSpec struct {

	// Type is the type of the unitset
	Type string `json:"type,omitempty"`

	// Edition of the unit set
	Edition string `json:"edition,omitempty"`

	// Version of the unit set, e.g.:8.0.40
	Version string `json:"version,omitempty"`

	// Units Number of units in the unitset
	Units int `json:"units,omitempty"`

	// Resources Resource requirements for the units
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Env Environment variables for the units
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// ExternalService Configuration for external services
	// +optional
	ExternalService ExternalServiceSpec `json:"externalService,omitempty"`

	// UnitService Configuration for unit services
	// +optional
	UnitService UnitServiceSpec `json:"unitService,omitempty"`

	// UpdateStrategy Strategy for updating the unit set
	// +optional
	UpdateStrategy UpdateStrategySpec `json:"updateStrategy,omitempty"`

	//NodeAffinityPreset  Node affinity rules
	// +optional
	NodeAffinityPreset []NodeAffinityPresetSpec `json:"nodeAffinityPreset,omitempty"`

	// PodAntiAffinityPreset Pod anti-affinity policy
	// +optional
	PodAntiAffinityPreset string `json:"podAntiAffinityPreset,omitempty"`

	// Storages defines the configuration for storage
	// +optional
	Storage []StorageSpec `json:"storage,omitempty"`

	// EmptyDir defines the configuration for emptyDir
	// +optional
	EmptyDir []EmptyDirSpec `json:"emptyDir,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	ExtraVolume []ExtraVolumeInfo `json:"extraVolume,omitempty"`

	// CertificateProfile defines the configuration for certificate profile
	// +optional
	CertificateProfile CertificateProfile `json:"certificateProfile,omitempty"`

	// PodMonitor defines the configuration for pod monitor
	// +optional
	PodMonitor PodMonitorInfo `json:"podMonitor,omitempty"`
}

type ExtraVolumeInfo struct {

	// Volume defines the configuration for volume
	// +optional
	Volume corev1.Volume `json:"volume,omitempty"`

	// VolumeMountPath Volume mount path
	// +optional
	VolumeMountPath string `json:"volumeMountPath,omitempty"`
}

type PodMonitorInfo struct {

	// Enable define if need create pod monitor
	// default: false
	// +optional
	Enable bool `json:"enable,omitempty"`
}

type CertificateProfile struct {

	// Organizations List of organization names
	// +optional
	Organizations []string `json:"organizations,omitempty"`

	// RootSecret Root secret name
	// +optional
	RootSecret string `json:"rootSecret,omitempty"`
}
type SecretInfo struct {

	// Name of the secret
	// +optional
	Name string `json:"name,omitempty"`

	// MountPath Mount path of the secret
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

// ExternalServiceSpec defines the configuration for external services.
type ExternalServiceSpec struct {

	// Type of the external service (e.g., NodePort)
	// +optional
	Type string `json:"type,omitempty"`
}

// UnitServiceSpec defines the configuration for unit services.
type UnitServiceSpec struct {

	// Type of the unit service (e.g., ClusterIP)
	// +optional
	Type string `json:"type,omitempty"`
}

// CertificateSecretSpec defines the configuration for certificate secrets.
type CertificateSecretSpec struct {

	// Organization name for the certificate
	// +optional
	Organization string `json:"organization,omitempty"`

	// Name of the certificate secret
	// +optional
	Name string `json:"name,omitempty"`
}

// UpdateStrategySpec defines the update strategy for the unit set.
type UpdateStrategySpec struct {

	// Type of update strategy (e.g., RollingUpdate)
	// +optional
	Type string `json:"type,omitempty"`

	// RollingUpdate Rolling update configuration
	// +optional
	RollingUpdate RollingUpdateSpec `json:"rollingUpdate,omitempty"`
}

// RollingUpdateSpec defines the rolling update configuration.
type RollingUpdateSpec struct {

	// Partition Number of partitions for the update
	// +optional
	Partition int32 `json:"partition,omitempty"`

	// MaxUnavailable Maximum number of unavailable pods during update
	// +optional
	MaxUnavailable int32 `json:"maxUnavailable,omitempty"`
}

// NodeAffinityPresetSpec defines node affinity rules.
type NodeAffinityPresetSpec struct {

	// Key for the node affinity
	// +optional
	Key string `json:"key,omitempty"`

	// Values for the node affinity
	// +optional
	Values []string `json:"values,omitempty"`
}

// StorageSpec defines the configuration for storage.
type StorageSpec struct {

	// Name of the storage
	// +optional
	Name string `json:"name,omitempty"`

	// Size of the storage
	// +optional
	Size string `json:"size,omitempty"`

	// StorageClassName storage class name
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// MountPath Mount path
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

type EmptyDirSpec struct {

	// Name of the storage
	// +optional
	Name string `json:"name,omitempty"`

	// Size of the storage
	// +optional
	Size string `json:"size,omitempty"`

	// MountPath Mount path
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

// UnitSetStatus defines the observed state of UnitSet
type UnitSetStatus struct {

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions is an array of conditions.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed for this UnitSet. It corresponds to the
	// UnitSet's generation, which is updated on mutation by the API Server.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Units the number of units
	// +optional
	Units int `json:"units,omitempty"`

	// ReadyUnits the number of ready units
	// +optional
	ReadyUnits int `json:"readyUnits"`

	// PvcSyncStatus defines the status of the pvc sync
	// +optional
	PvcSyncStatus PvcSyncStatus `json:"unitPVCSynced,omitempty"`

	// ImageSyncStatus defines the status of the image sync
	// +optional
	ImageSyncStatus ImageSyncStatus `json:"unitImageSynced,omitempty"`

	// ResourceSyncStatus defines the status of the resource sync
	// +optional
	ResourceSyncStatus ResourceSyncStatus `json:"unitResourceSynced,omitempty"`

	// InUpdate used to mark if a mirror upgrade or resource change is in progress
	// +optional
	InUpdate string `json:"inUpdate,omitempty"`

	// ExternalService the information of unitset external service
	// +optional
	ExternalService ExternalServiceStatus `json:"externalService,omitempty"`

	// UnitService the information of unit service
	// +optional
	UnitService UnitServiceStatus `json:"unitService,omitempty"`
}

type ExternalServiceStatus struct {

	// Name the name of unitset external service
	// +optional
	Name string `json:"name,omitempty"`
}

type UnitServiceStatus struct {

	// Name which is a map, the key is unit name, the value is unit service name
	// +optional
	Name map[string]string `json:"name,omitempty"`
}

// ResourceSyncStatus defines the observed state of ResourceSync
type ResourceSyncStatus struct {

	// LastTransitionTime the last transition time
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Status the status of the resource sync
	// enum: True, False
	// +optional
	Status string `json:"status,omitempty"`
}

// ImageSyncStatus defines the observed state of ImageSync
type ImageSyncStatus struct {

	// LastTransitionTime the last transition time
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Status the status of the image sync
	// enum: True, False
	// +optional
	Status string `json:"status,omitempty"`
}

// PvcSyncStatus defines the observed state of PvcSync
type PvcSyncStatus struct {

	// LastTransitionTime the last transition time
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Status the status of the pvc sync
	// enum: True, False
	// +optional
	Status string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=us
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.type",priority=1
// +kubebuilder:printcolumn:name="VERSION",type="string",JSONPath=".spec.version",priority=1
// +kubebuilder:printcolumn:name="EXPECTED",type=integer,JSONPath=`.spec.units`
// +kubebuilder:printcolumn:name="CURRENT",type=integer,JSONPath=`.status.units`
// +kubebuilder:printcolumn:name="READY",type=integer,JSONPath=`.status.readyUnits`
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// UnitSet is the Schema for the unitsets API
type UnitSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UnitSetSpec   `json:"spec,omitempty"`
	Status UnitSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UnitSetList contains a list of UnitSet
type UnitSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UnitSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UnitSet{}, &UnitSetList{})
}
