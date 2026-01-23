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

package unitset

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	controllerKind          = upmiov1alpha2.GroupVersion.WithKind("UnitSet")
	maxConcurrentReconciles = 10
)

const (
	requeueAfter = 10 * time.Second
)

// UnitSetReconciler reconciles a UnitSet object
type UnitSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=unitsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=unitsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=unitsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=units,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=podtemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=redisreplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=podmonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the UnitSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *UnitSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	startTime := time.Now()

	unitset := &upmiov1alpha2.UnitSet{}
	if err := r.Get(ctx, req.NamespacedName, unitset); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{
				RequeueAfter: requeueAfter,
			}, nil
		}

		klog.Errorf("unable to fetch Unitset [%s], error: [%v]", req.String(), err.Error())
		return ctrl.Result{
			RequeueAfter: requeueAfter,
		}, err
	}

	retErr = r.reconcileUnitset(ctx, req, unitset)

	defer func() {
		// If retErr is empty, print log
		// If retErr is not empty, print log and update error to unitset's event
		if retErr == nil {
			klog.Infof("finished reconciling Unitset [%s], duration [%v]", req.String(), time.Since(startTime))
		} else {
			klog.Errorf("failed to reconcile Unitset [%s], error: [%v]", req.String(), retErr)
			// Update unitset's event
			r.Recorder.Eventf(unitset, v1.EventTypeWarning, "FailedReconcile", retErr.Error())
		}
	}()

	return ctrl.Result{
		RequeueAfter: requeueAfter,
	}, retErr
}

func (r *UnitSetReconciler) reconcileUnitset(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) (err error) {
	klog.Infof("start reconciling Unitset [%s]", req.String())

	// Handle deletion only when DeletionTimestamp is set and non-zero.
	// NOTE: The old `!= nil || !IsZero()` form can panic when DeletionTimestamp is nil.
	if !unitset.DeletionTimestamp.IsZero() {
		klog.Infof("Unitset [%s] is being deleted, finalizers: %v", req.String(), unitset.GetFinalizers())

		errs := []error{}
		var (
			wg sync.WaitGroup
			mu sync.Mutex
		)

		toRemoveFinalizer := unitset.GetFinalizers()

		// The object is being deleted
		for _, myFinalizerName := range toRemoveFinalizer {
			wg.Add(1)
			go func(finalizer string) {
				defer wg.Done()

				// our finalizer is present, so lets handle any external dependency
				if deleteResourcesErr := r.deleteResources(ctx, req, unitset, finalizer); deleteResourcesErr != nil {
					mu.Lock()
					errs = append(errs, deleteResourcesErr)
					mu.Unlock()
					return
				}

			}(myFinalizerName)
		}
		wg.Wait()

		// remove our finalizer from the list and update it.
		err = utilerrors.NewAggregate(errs)
		if err != nil {
			return fmt.Errorf("UNITSET DELETING:error [%s]", err.Error())
		}

		// Stop reconciliation as the item is being deleted
		return nil
	}

	err = r.reconcileConfigmap(ctx, req, unitset)
	if err != nil {
		return err
	}

	// the pod template is old version, when image update, it will be updated in updateImage func
	podTemplate, err := r.getPodTemplate(ctx, req, unitset)
	if err != nil {
		return fmt.Errorf("get pod template error:[%s]", err.Error())
	}

	ports := getPortsFromPodtemplate(ctx, req, unitset, podTemplate)
	if len(ports) == 0 {
		return fmt.Errorf("get ports in pod template error: [ not found ports in pod template, ports: Required value ]")
	}

	err = r.reconcileHeadlessService(ctx, req, unitset, ports)
	if err != nil {
		return err
	}

	err = r.reconcileExternalService(ctx, req, unitset, ports)
	if err != nil {
		return err
	}

	err = r.reconcileUnitService(ctx, req, unitset, ports)
	if err != nil {
		return err
	}

	err = r.reconcileUnitCertificates(ctx, req, unitset)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcileUnitsetStatus(ctx, req, unitset)
	})
	if err != nil {
		klog.Errorf("failed to reconcile UnitsetStatus [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile UnitsetStatus, err: [%v]", err.Error())
	}

	err = r.reconcileUnit(ctx, req, unitset, &podTemplate, ports)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcilePatchUnitset(ctx, req, unitset)
	})
	if err != nil {
		klog.Errorf("failed to reconcile PatchUnitset [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile PatchUnitset, err: [%v]", err.Error())
	}

	err = r.reconcilePodMonitor(ctx, req, unitset)
	if err != nil {
		return err
	}

	err = r.reconcileImageVersion(ctx, req, unitset, &podTemplate, ports)
	if err != nil {
		return err
	}

	err = r.reconcileResources(ctx, req, unitset)
	if err != nil {
		return err
	}

	// Reconcile ResizePolicy independently of resources
	// This allows ResizePolicy to be updated separately without requiring resource changes
	err = r.reconcileResizePolicy(ctx, req, unitset)
	if err != nil {
		return err
	}

	// Propagate UnitSet labels/annotations to managed Units
	err = r.reconcileUnitLabelsAnnotations(ctx, req, unitset)
	if err != nil {
		return err
	}

	err = r.reconcileStorage(ctx, req, unitset)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcileUnitsetStatus(ctx, req, unitset)
	})
	if err != nil {
		klog.Errorf("failed to reconcile UnitsetStatus [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile UnitsetStatus, err: [%v]", err.Error())
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UnitSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&upmiov1alpha2.UnitSet{}).
		Owns(&v1.Pod{}).
		Owns(&upmiov1alpha2.Unit{}).
		Owns(&rbacV1.Role{}).
		Owns(&rbacV1.RoleBinding{}).
		Owns(&v1.Service{}).
		Owns(&v1.ServiceAccount{}).
		Owns(&v1.Secret{}).
		Owns(&v1.ConfigMap{}).
		WithOptions(
			controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles},
		).
		Complete(r)
}
