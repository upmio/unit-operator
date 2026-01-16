/*
 * UPM for Enterprise
 *
 * Copyright (c) 2009-2025 SYNTROPY Pte. Ltd.
 * All rights reserved.
 *
 * This software is the confidential and proprietary information of
 * SYNTROPY Pte. Ltd. ("Confidential Information"). You shall not
 * disclose such Confidential Information and shall use it only in
 * accordance with the terms of the license agreement you entered
 * into with SYNTROPY.
 */

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CA contains information about the Certificate Authority configuration.
	// +optional
	CA CAInfo `json:"ca,omitempty"`
}

// CAInfo contains information about the Certificate Authority configuration.
type CAInfo struct {

	// Enabled indicates whether the CA is enabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// CommonName is the common name for the CA certificate.
	// +kubebuilder:validation:MinLength=1
	//+optional
	CommonName string `json:"commonName,omitempty"`

	// SecretName is the name of the Kubernetes secret storing the CA.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Duration is the validity period of the CA certificate.
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +optional
	Duration string `json:"duration,omitempty"`

	// RenewBefore is the time before expiration when the certificate should be renewed.
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +optional
	RenewBefore string `json:"renewBefore,omitempty"`

	// PrivateKey contains information about the CA's private key.
	// +optional
	PrivateKey PrivateKeyInfo `json:"privateKey,omitempty"`
}

// PrivateKeyInfo contains details about the private key used by the CA.
type PrivateKeyInfo struct {

	// Algorithm is the cryptographic algorithm used for the private key.
	// +kubebuilder:validation:Enum=RSA;ECDSA;Ed25519
	// +optional
	Algorithm string `json:"algorithm,omitempty"`

	// Size is the size of the private key in bits.
	// +kubebuilder:validation:Minimum=1
	// +optional
	Size int `json:"size,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pj

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
