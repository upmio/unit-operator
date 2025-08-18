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
	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
// var unitlog = logf.Log.WithName("unit-resource")

type unitAdmission struct {
	client client.Reader
}

// SetupUnitWebhookWithManager will setup the manager to manage the webhooks
// func (r *upmv1alpha2.Unit) SetupWebhookWithManager(mgr ctrl.Manager) error {
func SetupUnitWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&upmv1alpha2.Unit{}).
		WithValidator(&unitAdmission{client: mgr.GetAPIReader()}).
		WithDefaulter(&unitAdmission{client: mgr.GetAPIReader()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-upm-syntropycloud-io-v1alpha2-unit,mutating=true,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=units,verbs=create;update,versions=v1alpha2,name=munit.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &unitAdmission{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *unitAdmission) Default(ctx context.Context, obj runtime.Object) error {
	//unitlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

// +kubebuilder:webhook:path=/validate-upm-syntropycloud-io-v1alpha2-unit,mutating=false,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=units,verbs=create;update;delete,versions=v1alpha2,name=vunit.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &unitAdmission{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *unitAdmission) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	//unitlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *unitAdmission) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	//unitlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *unitAdmission) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	//unitlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
