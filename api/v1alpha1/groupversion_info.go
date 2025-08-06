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

// Package v1alpha1 contains API Schema definitions for the upm v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=upm.syntropycloud.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "upm.syntropycloud.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
