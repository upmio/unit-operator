package unitset

import (
	"context"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/utils/patch"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcilePatchUnitset(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	units, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcilePatchUnitset] error getting units: [%s]", err.Error())
	}

	// patch unitset
	if units != nil && len(units) != 0 {
		originalUnitset := unitset.DeepCopy()
		needPatch := false
		if unitset.Spec.NodeNameMap == nil {
			needPatch = true
			unitset.Spec.NodeNameMap = make(map[string]string)
		}

		for _, one := range units {
			if one.Status.NodeName == "" {
				continue
			}

			if unitset.Spec.NodeNameMap[one.Name] == upmiov1alpha2.NoneSetFlag {
				continue
			}

			unitset.Spec.NodeNameMap[one.Name] = one.Status.NodeName
		}

		if needPatch || mapsEqual(originalUnitset.Spec.NodeNameMap, unitset.Spec.NodeNameMap) == false {
			_, err = r.patchUnitset(ctx, originalUnitset, unitset)
		}
	}

	return nil
}
func (r *UnitSetReconciler) reconcileUnitsetStatus(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	orig := unitset.DeepCopy()

	units, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		klog.Errorf("[reconcileUnitsetStatus]failed to get units for unitset [%s], error: [%v]", req.String(), err.Error())
		unitset.Status = upmiov1alpha2.UnitSetStatus{
			Units:      0,
			ReadyUnits: 0,
			ImageSyncStatus: upmiov1alpha2.ImageSyncStatus{
				LastTransitionTime: v1.Now(),
				Status:             "False",
			},
			ResourceSyncStatus: upmiov1alpha2.ResourceSyncStatus{
				LastTransitionTime: v1.Now(),
				Status:             "False",
			},
			PvcSyncStatus: upmiov1alpha2.PvcSyncStatus{
				LastTransitionTime: v1.Now(),
				Status:             "False",
			},
		}
	}

	unitset.Status.Units = len(units)
	inTaskUnit := ""
	unitset.Status.ReadyUnits = 0
	if units != nil && len(units) != 0 {
		for _, one := range units {
			if one.Status.Phase == upmiov1alpha2.UnitReady {
				unitset.Status.ReadyUnits++
			}
		}
	}

	unitPVCSyncedCount := 0
	unitImageSyncedCount := 0
	unitResourceSyncedCount := 0

	if units != nil && len(units) != 0 {
		for _, one := range units {
			//if one.Status.Phase == upmiov1alpha2.UnitReady {
			//	unitset.Status.ReadyUnits++
			//}

			if one.Status.Task != "" {
				inTaskUnit = inTaskUnit + "_" + one.Name
			}

			for _, pvc := range one.Status.PersistentVolumeClaim {
				for _, expect := range unitset.Spec.Storages {
					if pvc.Name == upmiov1alpha2.PersistentVolumeClaimName(one, expect.Name) {
						if pvc.Capacity.Storage == resource.MustParse(expect.Size) {
							unitPVCSyncedCount++
						}
					}
				}
			}

			for i := range one.Spec.Template.Spec.Containers {
				if one.Spec.Template.Spec.Containers[i].Name == one.Annotations[upmiov1alpha2.AnnotationMainContainerName] {
					if one.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu().
						Cmp(*unitset.Spec.Resources.Limits.Cpu()) == 0 &&
						one.Spec.Template.Spec.Containers[i].Resources.Limits.Memory().
							Cmp(*unitset.Spec.Resources.Limits.Memory()) == 0 &&
						one.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu().
							Cmp(*unitset.Spec.Resources.Requests.Cpu()) == 0 &&
						one.Spec.Template.Spec.Containers[i].Resources.Requests.Memory().
							Cmp(*unitset.Spec.Resources.Requests.Memory()) == 0 {
						unitResourceSyncedCount++
					}
				}
			}

			if one.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] == unitset.Spec.Version {
				unitImageSyncedCount++
			}
		}
	}

	if unitImageSyncedCount == orig.Spec.Units {
		unitset.Status.ImageSyncStatus = upmiov1alpha2.ImageSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "True",
		}
	} else {
		unitset.Status.ImageSyncStatus = upmiov1alpha2.ImageSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "False",
		}
	}

	if len(unitset.Spec.Storages)*orig.Spec.Units == unitPVCSyncedCount {
		unitset.Status.PvcSyncStatus = upmiov1alpha2.PvcSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "True",
		}
	} else {
		unitset.Status.PvcSyncStatus = upmiov1alpha2.PvcSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "False",
		}
	}

	if unitResourceSyncedCount == orig.Spec.Units {
		unitset.Status.ResourceSyncStatus = upmiov1alpha2.ResourceSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "True",
		}
	} else {
		unitset.Status.ResourceSyncStatus = upmiov1alpha2.ResourceSyncStatus{
			LastTransitionTime: v1.Now(),
			Status:             "False",
		}
	}

	if unitImageSyncedCount == orig.Spec.Units && unitResourceSyncedCount == orig.Spec.Units {
		unitset.Status.InUpdate = ""
	} else {
		unitset.Status.InUpdate = inTaskUnit
	}

	//if equality.Semantic.DeepEqual(orig.Status, unitset.Status) {
	//	return nil
	//}

	if unitset.Status.Units != orig.Status.Units ||
		unitset.Status.ReadyUnits != orig.Status.ReadyUnits ||
		unitset.Status.ImageSyncStatus.Status != orig.Status.ImageSyncStatus.Status ||
		unitset.Status.ResourceSyncStatus.Status != orig.Status.ResourceSyncStatus.Status ||
		unitset.Status.PvcSyncStatus.Status != orig.Status.PvcSyncStatus.Status {

		return r.Status().Update(ctx, unitset)
	}

	return nil
}

func (r *UnitSetReconciler) patchUnitset(ctx context.Context, old, _new *upmiov1alpha2.UnitSet) (*upmiov1alpha2.UnitSet, error) {
	patchData, update, err := patch.GenerateMergePatch(old, _new, upmiov1alpha2.UnitSet{})
	if err != nil || !update {
		return old, err
	}

	r.Recorder.Eventf(old, coreV1.EventTypeNormal, "ResourceCheck", "patch unitset ok~ (data: %s)", patchData)

	err = r.Patch(ctx, old, client.RawPatch(types.MergePatchType, patchData), &client.PatchOptions{})
	if err != nil {
		return nil, err
	}

	return _new, nil
}

func mapsEqual(a, b map[string]string) bool {
	// check length
	if len(a) != len(b) {
		return false
	}

	// check keys
	for k, value1 := range a {
		if value2, exists := b[k]; !exists || value2 != value1 {
			return false // not equal
		}
	}
	return true
}
