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

package unit

import (
	"context"
	"fmt"
	"sync"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/utils/patch"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

// UnitReconciler reconciles a Unit object
type UnitReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

var (
	controllerKind          = upmiov1alpha2.GroupVersion.WithKind("Unit")
	maxConcurrentReconciles = 10
)

// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=units,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=units/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=units/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Unit object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *UnitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, retErr error) {
	startTime := time.Now()

	unit := &upmiov1alpha2.Unit{}
	if err := r.Get(ctx, req.NamespacedName, unit); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{
				RequeueAfter: 3 * time.Second,
			}, nil
		}

		klog.Errorf("unable to fetch Unit [%s], error: [%v]", req.String(), err.Error())
		return ctrl.Result{
			RequeueAfter: 3 * time.Second,
		}, err
	}

	retErr = r.reconcileUnit(ctx, req, unit)

	defer func() {
		if retErr == nil {
			klog.Infof("finished reconciling Unit [%s], duration [%v]", req.String(), time.Since(startTime))
		} else {
			klog.Errorf("failed to reconcile Unit [%s], error: [%v]", req.String(), retErr)
			r.Recorder.Eventf(unit, v1.EventTypeWarning, "Failed", retErr.Error())
		}
	}()

	return ctrl.Result{
		RequeueAfter: 3 * time.Second,
	}, retErr
}

func (r *UnitReconciler) reconcileUnit(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	klog.Infof("start reconciling Unit [%s]", req.String())

	var err error
	// examine DeletionTimestamp to determine if object is under deletion
	if unit.DeletionTimestamp != nil || !unit.DeletionTimestamp.IsZero() {

		klog.Infof("Unit [%s] is being deleted, finalizers: %v", req.String(), unit.GetFinalizers())

		errs := []error{}
		var wg sync.WaitGroup

		toRemoveFinalizer := unit.GetFinalizers()

		// The object is being deleted
		for _, myFinalizerName := range toRemoveFinalizer {
			wg.Add(1)
			go func(finalizer string) {
				defer wg.Done()
				// our finalizer is present, so lets handle any external dependency
				if deleteResourcesErr := r.deleteResources(ctx, req, unit, finalizer); deleteResourcesErr != nil {
					// if fail to delete the external dependency here, return with error
					// so that it can be retried.
					errs = append(errs, deleteResourcesErr)
					return
				}
			}(myFinalizerName)
		}
		wg.Wait()

		// remove our finalizer from the list and update it.
		err := utilerrors.NewAggregate(errs)
		if err != nil {
			return fmt.Errorf("UNIT DELETING: failed to delete external resources: [%v]", err.Error())
		}

		// Stop reconciliation as the item is being deleted
		return nil
	}

	err = r.reconcilePersistentVolumeClaims(ctx, req, unit)
	if err != nil {
		klog.Errorf("failed to reconcile PersistentVolumeClaims [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to reconcile PersistentVolumeClaims [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcileUnitStatus(ctx, req, unit)
	})
	if err != nil {
		klog.Errorf("failed to reconcile UnitStatus [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile UnitStatus, err: [%v]", err.Error())
	}

	err = r.podAutoRecovery(ctx, req, unit)
	if err != nil {
		klog.Errorf("failed to auto recovery Pod [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to auto recovery Pod [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	err = r.reconcilePod(ctx, req, unit)
	if err != nil {
		klog.Errorf("failed to reconcile Pod [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to reconcile Pod [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	_, err = r.waitUntilPodScheduled(ctx, unit.Name, unit.GetNamespace())
	if err != nil {
		klog.Errorf("failed to waitUntilPodScheduled [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to waitUntilPodScheduled [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	//{
	//	belongNode := unit.Annotations[upmiov1alpha2.AnnotationLastUnitBelongNode]
	//	if belongNode != pod.Spec.NodeName {
	//		updateUnit := unit.DeepCopy()
	//		updateUnit.Annotations[upmiov1alpha2.AnnotationLastUnitBelongNode] = pod.Spec.NodeName
	//		_, err = r.patchUnit(ctx, unit, updateUnit)
	//		if err != nil {
	//			return fmt.Errorf("patchUnit fail: [%s]", err.Error())
	//		}
	//	}
	//}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcileUnitStatus(ctx, req, unit)
	})
	if err != nil {
		klog.Errorf("failed to reconcile UnitStatus [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile UnitStatus, err: [%v]", err.Error())
	}

	err = r.reconcileUnitConfig(ctx, unit)
	if err != nil {
		klog.Errorf("failed to reconcile UnitConfig [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to reconcile UnitConfig [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	err = r.reconcileUnitServer(ctx, unit)
	if err != nil {
		klog.Errorf("failed to reconcile UnitServer [%s], err: [%v]", req.NamespacedName, err.Error())
		return fmt.Errorf("failed to reconcile UnitServer [%s], err: [%v]", req.NamespacedName, err.Error())
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcileUnitStatus(ctx, req, unit)
	})
	if err != nil {
		klog.Errorf("failed to reconcile UnitStatus [%s], err: [%v]", req.String(), err.Error())
		return fmt.Errorf("failed to reconcile UnitStatus, err: [%v]", err.Error())
	}

	return nil
}

func (r *UnitReconciler) patchUnit(ctx context.Context, old, _new *upmiov1alpha2.Unit) (*upmiov1alpha2.Unit, error) {
	patch, update, err := patch.GenerateMergePatch(old, _new, upmiov1alpha2.Unit{})
	if err != nil || !update {
		return old, err
	}

	r.Recorder.Eventf(old, v1.EventTypeNormal, "ResourceCheck", "patch unit ok~ (data: %s)", patch)

	err = r.Patch(ctx, old, client.RawPatch(types.MergePatchType, patch), &client.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return _new, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UnitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.TODO(),
		&v1.PersistentVolume{},
		".spec.claimRef.name",
		func(rawObj client.Object) []string {
			pv := rawObj.(*v1.PersistentVolume)
			return []string{pv.Spec.ClaimRef.Name}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&upmiov1alpha2.Unit{}).
		Owns(&v1.Pod{}).
		Owns(&v1.PersistentVolumeClaim{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}
