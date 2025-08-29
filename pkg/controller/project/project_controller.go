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

package project

import (
	"context"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	upmv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

var (
	controllerKind          = upmiov1alpha2.GroupVersion.WithKind("Project")
	maxConcurrentReconciles = 10
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Project object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	startTime := time.Now()

	project := &upmiov1alpha2.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{
				RequeueAfter: 3 * time.Second,
			}, nil
		}

		klog.Errorf("unable to fetch project [%s], error: [%v]", req.String(), err.Error())
		return ctrl.Result{
			RequeueAfter: 3 * time.Second,
		}, err
	}

	retErr = r.reconcileProject(ctx, req, project)

	defer func() {
		// If retErr is empty, print log
		// If retErr is not empty, print log and update error to Project's event
		if retErr == nil {
			klog.Infof("finished reconciling Project [%s], duration [%v]", req.String(), time.Since(startTime))
		} else {
			klog.Errorf("failed to reconcile Project [%s], error: [%v]", req.String(), retErr)
			// Update project's event
			r.Recorder.Eventf(project, v1.EventTypeWarning, "FailedReconcile", retErr.Error())
		}
	}()

	return ctrl.Result{
		RequeueAfter: 3 * time.Second,
	}, retErr
}

func (r *ProjectReconciler) reconcileProject(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) (err error) {
	klog.Infof("start reconciling Project [%s]", req.String())

	err = r.reconcileNamespace(ctx, req, project)
	if err != nil {
		return err
	}

	err = r.reconcileServiceAccount(ctx, req, project)
	if err != nil {
		return err
	}

	err = r.reconcileSecret(ctx, req, project)
	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&upmv1alpha2.Project{}).
		Owns(&rbacV1.Role{}).
		Owns(&rbacV1.RoleBinding{}).
		Owns(&v1.Service{}).
		Owns(&v1.ServiceAccount{}).
		Owns(&v1.Namespace{}).
		Owns(&v1.Secret{}).
		Owns(&v1.ConfigMap{}).
		WithOptions(
			controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles},
		).
		Complete(r)
}
