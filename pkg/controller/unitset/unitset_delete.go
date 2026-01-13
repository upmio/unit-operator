package unitset

import (
	"context"
	"fmt"
	"sync"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *UnitSetReconciler) deleteResources(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet, finalizer string) error {
	switch finalizer {
	case upmiov1alpha2.FinalizerUnitDelete:
		return r.deleteUnitWithFinalizer(ctx, req, unitset, finalizer)
	case upmiov1alpha2.FinalizerConfigMapDelete:
		return r.deleteConfigMapWithFinalizer(ctx, req, unitset, finalizer)
	}

	return nil
}

func (r *UnitSetReconciler) unitsBelongUnitset(ctx context.Context, unitset *upmiov1alpha2.UnitSet) ([]*upmiov1alpha2.Unit, error) {

	kUnits := &upmiov1alpha2.UnitList{}
	err := r.List(ctx, kUnits, client.InNamespace(unitset.Namespace), client.MatchingLabels{upmiov1alpha2.UnitsetName: unitset.Name})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	if len(kUnits.Items) == 0 {
		return nil, nil
	}

	out := []*upmiov1alpha2.Unit{}
	for i := range kUnits.Items {
		out = append(out, kUnits.Items[i].DeepCopy())
	}

	return out, nil
}

func (r *UnitSetReconciler) deleteUnitWithFinalizer(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	finalizer string) error {
	klog.Infof("unitset:[%s] start delete units and remove finalizer:[%s]", req.String(), finalizer)

	kUnits, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[deleteUnitWithFinalizer] error getting units: [%s]", err.Error())
	}

	if len(kUnits) == 0 {
		if controllerutil.ContainsFinalizer(unitset, finalizer) {
			controllerutil.RemoveFinalizer(unitset, finalizer)

			if err := r.Update(ctx, unitset); err != nil {
				return fmt.Errorf("[deleteUnitWithFinalizer] error removing finalizer: [%s]", err.Error())
			}
		}

		return nil
	}

	for _, one := range kUnits {
		err := r.Delete(ctx, one)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("[deleteUnitWithFinalizer] error deleting unit: [%s]", err.Error())
		}
	}

	// wait for units deleted
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 28*time.Second, true, func(ctx context.Context) (bool, error) {
		kUnits, err := r.unitsBelongUnitset(ctx, unitset)
		if err != nil {
			return false, fmt.Errorf("[deleteUnitWithFinalizer] wait for units deleted: error getting units: [%s]", err.Error())
		}

		if len(kUnits) == 0 {
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error waiting for units deleted: [%s]", err.Error())
	}

	// remove finalizer
	if controllerutil.ContainsFinalizer(unitset, finalizer) {
		controllerutil.RemoveFinalizer(unitset, finalizer)

		if err := r.Update(ctx, unitset); err != nil {
			return fmt.Errorf("[deleteUnitWithFinalizer] error removing finalizer[%s]: [%s]", finalizer, err.Error())
		}

	}

	return nil
}

func (r *UnitSetReconciler) deleteConfigMapWithFinalizer(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	finalizer string) error {
	klog.Infof("unitset:[%s] start delete configmaps and remove finalizer:[%s]", req.String(), finalizer)

	needDeleteConfigmapName := []string{}
	needDeleteConfigmapName = append(needDeleteConfigmapName, unitset.ConfigTemplateName())
	unitNames, _ := unitset.UnitNames()
	for i := range unitNames {
		needDeleteConfigmapName = append(needDeleteConfigmapName, unitset.ConfigValueName(unitNames[i]))
	}

	errs := []error{}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, configmapName := range needDeleteConfigmapName {
		if configmapName == "" {
			continue
		}

		wg.Add(1)
		go func(configmapName string) {
			defer wg.Done()
			klog.Infof("[deleteConfigMapWithFinalizer] delete configmap:[%s]...", configmapName)
			cm := v1.ConfigMap{}
			cm.SetNamespace(unitset.Namespace)
			cm.SetName(configmapName)

			err := r.Delete(ctx, &cm)
			if err != nil && !apierrors.IsNotFound(err) {
				mu.Lock()
				errs = append(errs, fmt.Errorf("error deleting configmap:[%s]: [%s]", configmapName, err.Error()))
				mu.Unlock()
				return
			}

		}(configmapName)

	}
	wg.Wait()

	deleteErr := utilerrors.NewAggregate(errs)
	if deleteErr != nil {
		return fmt.Errorf("[deleteConfigMapWithFinalizer] error deleting configmap: [%s]", deleteErr.Error())
	}

	// wait for cm deleted (no hard-coded sleep)
	waitErrs := []error{}
	var (
		waitWg   sync.WaitGroup
		waitMu   sync.Mutex
		interval = 200 * time.Millisecond
		timeout  = 10 * time.Second
	)
	for _, configmapName := range needDeleteConfigmapName {
		if configmapName == "" {
			continue
		}

		waitWg.Add(1)
		go func(configmapName string) {
			defer waitWg.Done()

			err := wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
				configmap := &v1.ConfigMap{}
				err := r.Get(ctx, client.ObjectKey{Name: configmapName, Namespace: unitset.Namespace}, configmap)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					}

					return false, fmt.Errorf("error waiting for configmap:[%s] deleted: %s", configmapName, err.Error())
				}

				return false, nil
			})

			if err != nil {
				waitMu.Lock()
				waitErrs = append(waitErrs, err)
				waitMu.Unlock()
				return
			}

		}(configmapName)
	}
	waitWg.Wait()

	waitErr := utilerrors.NewAggregate(waitErrs)
	if waitErr != nil {
		return fmt.Errorf("[deleteConfigMapWithFinalizer] error waiting for configmap deleted: [%s]", waitErr.Error())
	}

	if controllerutil.ContainsFinalizer(unitset, finalizer) {
		controllerutil.RemoveFinalizer(unitset, finalizer)

		if err := r.Update(ctx, unitset); err != nil {
			return fmt.Errorf("[deleteConfigMapWithFinalizer] error removing finalizer: [%s]", err.Error())
		}

	}

	return nil
}
