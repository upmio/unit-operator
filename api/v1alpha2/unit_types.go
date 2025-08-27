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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// UnitSpec defines the desired state of Unit
type UnitSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Startup defines whether the service is started or not
	// +optional
	Startup bool `json:"startup,omitempty"`

	// ConfigTemplateName defines the config template name.
	// A unitset is instantiated as a config template for the unitset
	// by copying the corresponding version template.
	// one for a set of unitsets.
	// The unitset is then assigned a value to the field.
	// unitset is not processed logically
	// and is passed as a parameter when the unit agent is called.
	// +optional
	ConfigTemplateName string `json:"configTemplateName,omitempty"`

	// ConfigValueName defines the config value name.
	// unitset instantiates one for each unit by copying the corresponding version template.
	// The value is then assigned to the field.
	// unitset does no logical processing
	// and is passed as a parameter in the call to the unit agent
	// +optional
	ConfigValueName string `json:"configValueName,omitempty"`

	// VolumeClaimTemplates is a user's request for and claim to a persistent volume
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// Template describes the data a pod should have when created from a template
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
}

// UnitStatus defines the observed state of Unit
type UnitStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions is an array of conditions.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase is the current lifecycle phase of the unit.
	// +optional
	Phase UnitPhase `json:"phase"`

	// NodeReady is the state of node ready condition.
	// +optional
	NodeReady string `json:"nodeReady"`

	// NodeName is the node name of the unit.
	// +optional
	NodeName string `json:"nodeName"`

	// Task is the current task of the unit.
	// +optional
	Task string `json:"task"`

	// ProcessState is the current process state of the unit-operator.
	// +optional
	ProcessState string `json:"processState"`

	// HostIP represents a single IP address allocated to the host.
	// +optional
	HostIP string `json:"hostIP"`

	// PodIPs holds the IP addresses allocated to the pod.
	// +optional
	PodIPs []corev1.PodIP `json:"podIPs"`

	// ConfigSyncStatus represents the status of the config sync.
	// +optional
	ConfigSyncStatus ConfigSyncStatus `json:"configSynced,omitempty"`

	// PersistentVolumeClaim represents the current information/status of a persistent volume claim.
	// +optional
	PersistentVolumeClaim []PvcInfo `json:"persistentVolumeClaim"`
}

// PvcInfo represents the current information/status of a persistent volume claim.
type PvcInfo struct {

	// Name name of a persistent volume claim.
	// +optional
	Name string `json:"name"`

	// VolumeName name of volume
	// +optional
	VolumeName string `json:"volumeName"`

	// AccessModes contains the actual access modes the volume backing the PVC has.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes"`

	// Capacity represents the actual resources of the PVC.
	// +optional
	Capacity PvcCapacity `json:"capacity"`

	// Phase represents the current phase of PersistentVolumeClaim.
	// +optional
	Phase corev1.PersistentVolumeClaimPhase `json:"phase"`
}

// PvcCapacity represents the actual resources of the PVC.
type PvcCapacity struct {

	// Storage represents the actual resources of the PVC.
	// +optional
	Storage resource.Quantity `json:"storage"`
}

type ConfigSyncStatus struct {

	// LastTransitionTime the last transition time
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Status the status of the config sync
	// +optional
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=un
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="PROCESS STATE",type=string,JSONPath=`.status.processState`
// +kubebuilder:printcolumn:name="NODE NAME",type=string,JSONPath=`.status.nodeName`
// +kubebuilder:printcolumn:name="NODE READY",type=string,JSONPath=`.status.nodeReady`
// +kubebuilder:printcolumn:name="HOST IP",type=string,JSONPath=`.status.hostIP`,priority=1
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Unit is the Schema for the units API
type Unit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UnitSpec   `json:"spec,omitempty"`
	Status UnitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UnitList contains a list of Unit
type UnitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Unit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Unit{}, &UnitList{})
}
