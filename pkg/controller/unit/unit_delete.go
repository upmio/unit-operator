package unit

import (
	"context"
	"fmt"
	"sync"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *UnitReconciler) deleteResources(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit, finalizer string) error {
	switch finalizer {
	case upmiov1alpha2.FinalizerPodDelete:
		return r.deletePodWithFinalizer(ctx, req, unit, finalizer)
	case upmiov1alpha2.FinalizerPvcDelete:
		return r.deletePVCWithFinalizer(ctx, req, unit, finalizer)
	}

	return nil
}

func (r *UnitReconciler) deletePVCWithFinalizer(
	ctx context.Context,
	req ctrl.Request,
	unit *upmiov1alpha2.Unit,
	finalizer string) error {
	klog.Infof("unit:[%s] start delete pvc and remove finalizer:[%s]", req.String(), finalizer)

	needDeletePVCNames := []string{}
	if len(unit.Spec.VolumeClaimTemplates) != 0 {
		for _, one := range unit.Spec.VolumeClaimTemplates {
			pvcName := upmiov1alpha2.PersistentVolumeClaimName(unit, one.Name)
			needDeletePVCNames = append(needDeletePVCNames, pvcName)
		}
	}

	errs := []error{}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, pvcName := range needDeletePVCNames {
		if pvcName == "" {
			continue
		}

		wg.Add(1)
		go func(pvcName string) {
			defer wg.Done()

			klog.Infof("[deletePVCWithFinalizer] delete pvc:[%s]...", pvcName)

			pvc := v1.PersistentVolumeClaim{}
			pvc.SetNamespace(req.Namespace)
			pvc.SetName(pvcName)

			if forceDelete, ok := unit.Annotations[upmiov1alpha2.AnnotationForceDelete]; ok && forceDelete == "true" {
				// force delete
				second := int64(0)
				err := r.Delete(ctx, &pvc, &client.DeleteOptions{GracePeriodSeconds: &second})
				klog.Infof("[deletePVCWithFinalizer] excute to FORCE delete pvc: %s, unit:[%s]", pvcName, unit.Name)
				if err != nil && !apierrors.IsNotFound(err) {
					mu.Lock()
					errs = append(errs, fmt.Errorf("error force deleting pvc:[%s]: [%s]", pvc.Name, err.Error()))
					mu.Unlock()
					return
				}

				// force delete pv
				pvList := &v1.PersistentVolumeList{}

				listOps := &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector(".spec.claimRef.name", pvcName),
				}

				err = r.List(ctx, pvList, listOps)
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("when force deleting pv, list pv error:[%s]", err.Error()))
					mu.Unlock()
					return
				}

				if len(pvList.Items) != 0 {
					for _, one := range pvList.Items {
						pv := one.DeepCopy()
						err = r.Delete(ctx, pv, &client.DeleteOptions{GracePeriodSeconds: &second})
						klog.Infof("[deletePVCWithFinalizer] excute to FORCE delete pv: %s, unit:[%s]", pv.Name, unit.Name)
						if err != nil && !apierrors.IsNotFound(err) {
							mu.Lock()
							errs = append(errs, fmt.Errorf("error force deleting pv:[%s]: [%s]", pv.Name, err.Error()))
							mu.Unlock()
							return
						}
					}
				}

			} else {
				// only delete pvc, pv deleted by pvc
				err := r.Delete(ctx, &pvc)
				if err != nil && !apierrors.IsNotFound(err) {
					mu.Lock()
					errs = append(errs, fmt.Errorf("error deleting pvc:[%s]: [%s]", pvc.Name, err.Error()))
					mu.Unlock()
					return
				}
			}

		}(pvcName)

	}
	wg.Wait()

	deleteErr := utilerrors.NewAggregate(errs)
	if deleteErr != nil {
		return fmt.Errorf("[deletePVCWithFinalizer] error deleting pvc: [%s]", deleteErr.Error())
	}

	// wait for pvc or pv deleted (no hard-coded sleep; wait handles eventual consistency)
	waitErrs := []error{}
	var (
		waitWg   sync.WaitGroup
		waitMu   sync.Mutex
		interval = 200 * time.Millisecond
		timeout  = 10 * time.Second
	)
	for i := range needDeletePVCNames {
		waitWg.Add(1)
		go func(pvcName string) {
			defer waitWg.Done()

			err := wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
				pvc := &v1.PersistentVolumeClaim{}
				err := r.Get(ctx, client.ObjectKey{Name: pvcName, Namespace: req.Namespace}, pvc)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return true, nil
					}

					return false, fmt.Errorf("error waiting for pvc:[%s] deleted: %s", pvcName, err.Error())
				}

				return false, nil
			})

			if err != nil {
				waitMu.Lock()
				waitErrs = append(waitErrs, err)
				waitMu.Unlock()
				return
			}

			// wait for pv deleted
			if forceDelete, ok := unit.Annotations[upmiov1alpha2.AnnotationForceDelete]; ok && forceDelete == "true" {

				pvList := &v1.PersistentVolumeList{}

				listOps := &client.ListOptions{
					FieldSelector: fields.OneTermEqualSelector(".spec.claimRef.name", pvcName),
				}

				err := r.List(ctx, pvList, listOps)
				if err != nil {
					waitMu.Lock()
					waitErrs = append(waitErrs, fmt.Errorf("wait for pv deleted, list pv error:[%s]", err.Error()))
					waitMu.Unlock()
					return
				}

				errs := []error{}
				var (
					pvDeleteWG sync.WaitGroup
					pvErrMu    sync.Mutex
				)
				if len(pvList.Items) != 0 {
					for _, one := range pvList.Items {
						pvDeleteWG.Add(1)
						go func(pvName string) {
							defer pvDeleteWG.Done()

							// wait for pv deleted
							err = wait.PollUntilContextTimeout(ctx, interval, 15*time.Second, true, func(ctx context.Context) (bool, error) {
								pv := v1.PersistentVolume{}
								pvName := types.NamespacedName{Name: pvName}

								err := r.Get(ctx, pvName, &pv)
								if err != nil {
									if apierrors.IsNotFound(err) {
										return true, nil
									}
									return false, fmt.Errorf("error waiting for pv:[%s] deleted: %s", pvName.String(), err.Error())
								}

								return false, nil
							})

							if err != nil {
								pvErrMu.Lock()
								errs = append(errs, fmt.Errorf("error waiting for pv deleted: %s", err.Error()))
								pvErrMu.Unlock()
							}

						}(one.Name)
					}
					pvDeleteWG.Wait()

					err := utilerrors.NewAggregate(errs)
					if err != nil {
						waitMu.Lock()
						waitErrs = append(waitErrs, fmt.Errorf("[deletePVCWithFinalizer] error waiting for pv deleted: [%s]", err.Error()))
						waitMu.Unlock()
						return
					}
				}
			}

		}(needDeletePVCNames[i])

	}
	waitWg.Wait()

	waitErr := utilerrors.NewAggregate(waitErrs)
	if waitErr != nil {
		return fmt.Errorf("[deletePVCWithFinalizer] error waiting for pvc/pv deleted: [%s]", waitErr.Error())
	}

	if controllerutil.ContainsFinalizer(unit, finalizer) {
		controllerutil.RemoveFinalizer(unit, finalizer)

		if err := r.Update(ctx, unit); err != nil {
			return fmt.Errorf("[deletePVCWithFinalizer] error removing finalizer:[%s]: [%s]", finalizer, err.Error())
		}

	}

	return nil
}

func (r *UnitReconciler) deletePodWithFinalizer(
	ctx context.Context,
	req ctrl.Request,
	unit *upmiov1alpha2.Unit,
	finalizer string) error {
	klog.Infof("unit:[%s] start delete pod and remove finalizer:[%s]", req.String(), finalizer)

	pod := &v1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if controllerutil.ContainsFinalizer(unit, finalizer) {
				controllerutil.RemoveFinalizer(unit, finalizer)

				if err := r.Update(ctx, unit); err != nil {
					return fmt.Errorf("[deletePodWithFinalizer] error removing finalizer: [%s]", err.Error())
				}
			}

			return nil
		}

		return fmt.Errorf("[deletePodWithFinalizer] get pod fail, error: [%s]", err.Error())
	}

	if forceDelete, ok := unit.Annotations[upmiov1alpha2.AnnotationForceDelete]; ok && forceDelete == "true" {

		// force delete
		second := int64(0)
		err = r.Delete(ctx, pod, &client.DeleteOptions{GracePeriodSeconds: &second})
		if err != nil {
			return err
		}

	} else {

		if err := r.Delete(ctx, pod); err != nil {
			return fmt.Errorf("[deletePodWithFinalizer] error deleting pod: [%s]", err.Error())
		}

	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 40*time.Second, true, func(ctx context.Context) (bool, error) {
		pod := &v1.Pod{}
		err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, pod)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, fmt.Errorf("[deletePodWithFinalizer]wait pod deleted: get pod fail, error: [%s]", err.Error())
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("[deletePodWithFinalizer] error waiting for pod deleted: [%s]", err.Error())
	}

	// remove finalizer
	if controllerutil.ContainsFinalizer(unit, finalizer) {
		controllerutil.RemoveFinalizer(unit, finalizer)

		if err = r.Update(ctx, unit); err != nil {
			return fmt.Errorf("[deletePodWithFinalizer] error removing finalizer[%s]: [%s]", finalizer, err.Error())
		}
	}

	return nil
}
