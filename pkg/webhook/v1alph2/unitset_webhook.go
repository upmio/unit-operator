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

package v1alph2

import (
	"context"
	"github.com/upmio/unit-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
//var unitsetlog = logf.Log.WithName("unitset-resource")

type unitSetAdmission struct {
	client client.Reader
}

// SetupUnitsetWebhookWithManager will setup the manager to manage the webhooks
// func (r *v1alpha2.UnitSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
func SetupUnitsetWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha2.UnitSet{}).
		WithValidator(&unitSetAdmission{client: mgr.GetAPIReader()}).
		WithDefaulter(&unitSetAdmission{client: mgr.GetAPIReader()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-upm-syntropycloud-io-v1alpha2-unitset,mutating=true,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=unitsets,verbs=create;update,versions=v1alpha2,name=munitset.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &unitSetAdmission{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *unitSetAdmission) Default(ctx context.Context, obj runtime.Object) error {
	//unitsetlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.

	unitSet := obj.(*v1alpha2.UnitSet)

	// 如果是删除操作，则直接退出
	if unitSet.GetDeletionTimestamp() != nil {
		return nil
	}

	//打印日志
	klog.Infof("[WEBHOOK LOG] [default] name: [%s/%s] add finalizer", unitSet.Namespace, unitSet.Name)

	controllerutil.AddFinalizer(unitSet, v1alpha2.FinalizerUnitDelete)
	controllerutil.AddFinalizer(unitSet, v1alpha2.FinalizerConfigMapDelete)

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

// +kubebuilder:webhook:path=/validate-upm-syntropycloud-io-v1alpha2-unitset,mutating=false,failurePolicy=fail,sideEffects=None,groups=upm.syntropycloud.io,resources=unitsets,verbs=create;update;delete,versions=v1alpha2,name=vunitset.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &unitSetAdmission{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *unitSetAdmission) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	//unitsetlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *unitSetAdmission) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	//unitsetlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *unitSetAdmission) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	//unitsetlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
