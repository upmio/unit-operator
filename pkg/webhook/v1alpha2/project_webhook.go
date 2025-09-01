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
	"context"
	"fmt"
	"strings"

	"github.com/upmio/unit-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var projectlog = logf.Log.WithName("project-resource")

type projectAdmission struct {
	client client.Reader
}

// SetupProjectWebhookWithManager will setup the manager to manage the webhooks
func SetupProjectWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha2.Project{}).
		WithValidator(&projectAdmission{client: mgr.GetAPIReader()}).
		WithDefaulter(&projectAdmission{client: mgr.GetAPIReader()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-upm-syntropycloud-io-v1alpha2-project,mutating=true,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=projects,verbs=create;update;delete,versions=v1alpha2,name=mproject.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &projectAdmission{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *projectAdmission) Default(ctx context.Context, obj runtime.Object) error {
	//projectlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

// +kubebuilder:webhook:path=/validate-upm-syntropycloud-io-v1alpha2-project,mutating=false,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=projects,verbs=create;update;delete,versions=v1alpha2,name=vproject.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &projectAdmission{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *projectAdmission) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	//projectlog.Info("validate create", "name", obj.GetObjectKind()..Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *projectAdmission) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	//projectlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *projectAdmission) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {

	// TODO(user): fill in your validation logic upon object deletion.
	project, ok := obj.(*v1alpha2.Project)
	if !ok {
		return nil, fmt.Errorf("object type assertion to Project failed")
	}

	projectlog.Info("validate delete", "name", project.Name)

	namespace := project.Namespace

	// Check UnitSet
	var unitsetList v1alpha2.UnitSetList
	if err := r.client.List(ctx, &unitsetList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list UnitSet in namespace [%s]: [%v]", namespace, err)
	}
	if len(unitsetList.Items) > 0 {
		names := make([]string, 0, len(unitsetList.Items))
		for _, us := range unitsetList.Items {
			names = append(names, us.Name)
		}
		return nil, fmt.Errorf("cannot delete Project: UnitSet(s) [%s] exist in namespace [%s]",
			strings.Join(names, ", "), namespace)
	}

	// Check Unit
	var unitList v1alpha2.UnitList
	if err := r.client.List(ctx, &unitList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list Unit in namespace [%s]: [%v]", namespace, err)
	}
	if len(unitList.Items) > 0 {
		names := make([]string, 0, len(unitList.Items))
		for _, u := range unitList.Items {
			names = append(names, u.Name)
		}
		return nil, fmt.Errorf("cannot delete Project: Unit(s) [%s] exist in namespace [%s]",
			strings.Join(names, ", "), namespace)
	}

	return nil, nil
}
