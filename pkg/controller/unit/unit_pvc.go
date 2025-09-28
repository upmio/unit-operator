package unit

import (
	"context"
	"fmt"
	"strings"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcilePersistentVolumeClaims(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	if len(unit.Spec.VolumeClaimTemplates) == 0 {
		return nil
	}

	for _, each := range unit.Spec.VolumeClaimTemplates {
		pvcName := upmiov1alpha2.PersistentVolumeClaimName(unit, each.Name)
		claim := &v1.PersistentVolumeClaim{}

		err := r.Get(ctx, client.ObjectKey{Name: pvcName, Namespace: unit.Namespace}, claim)
		if apierrors.IsNotFound(err) {
			// spec.claimRef.name == pvcName: list the pv corresponding to the pvc to be created.
			pvList := &v1.PersistentVolumeList{}

			// Note: envtest does not support custom field selectors on PVs. List all and filter in-memory.
			// Keep listOps var to avoid unused import of fields if refactored later.
			_ = &client.ListOptions{FieldSelector: fields.OneTermEqualSelector(".spec.claimRef.name", pvcName)}

			err := r.List(ctx, pvList)
			if err != nil {
				return fmt.Errorf("list pv error:[%s]", err.Error())
			}

			// if a pv exists, the new pvc is not allowed to be built,
			// an error is returned, and the error message prints that a pv already exists.
			if len(pvList.Items) != 0 {
				pvNames := []string{}
				for _, one := range pvList.Items {
					if one.Spec.ClaimRef != nil && one.Spec.ClaimRef.Name == pvcName {
						pvNames = append(pvNames, one.Name)
					}
				}
				if len(pvNames) != 0 {
					return fmt.Errorf("pv [%s] already exists, please delete them first", strings.Join(pvNames, ","))
				}
			}

			// no pv exists, create pvc
			claim, err = convert2PVC(unit, each)
			if err != nil {
				return err
			}

			err = r.Create(ctx, claim)
			// todo event
			if err != nil {
				return err
			}

		} else if err != nil {
			return err
		}

		if each.Spec.Resources.Requests.Storage().Cmp(*claim.Spec.Resources.Requests.Storage()) != 0 {
			newClaim := claim.DeepCopy()
			newClaim.Spec.Resources.Requests = each.Spec.Resources.Requests

			err = r.Update(ctx, newClaim)
			if err != nil {
				return fmt.Errorf("update pvc:[%s] error:[%s]", claim.Name, err.Error())
				//} else {
				//r.EventRecorder.Eventf(unit, corev1.EventTypeNormal, SuccessUpdated, "update pvc [%s] ok~", pvcName)
			}
		}
	}

	//klog.Infof("reconcilePersistentVolumeClaims ok, unit name: [%s]", unit.Name)

	return nil
}

func convert2PVC(unit *upmiov1alpha2.Unit, persistentVolumeClaim v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {

	//ref := metav1.NewControllerRef(unit, controllerKind)

	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: unit.Namespace,
			Name:      upmiov1alpha2.PersistentVolumeClaimName(unit, persistentVolumeClaim.Name),
			Labels:    make(map[string]string),
			//OwnerReferences: []metav1.OwnerReference{*ref},
		},

		Spec: persistentVolumeClaim.Spec,
	}

	claim.Labels = unit.Labels

	return claim, nil
}
