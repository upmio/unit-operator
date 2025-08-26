package unitset

import (
	"context"
	"encoding/json"
	"fmt"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/utils/patch"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcilePatchUnitset patch unitset: backfill NodeNameMap into annotation upm.io/node-name-map
func (r *UnitSetReconciler) reconcilePatchUnitset(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	units, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcilePatchUnitset] error getting units: [%s]", err.Error())
	}

	// patch unitset: backfill NodeNameMap into annotation upm.io/node-name-map
	if len(units) != 0 {
		originalUnitset := unitset.DeepCopy()
		current := getNodeNameMapFromAnnotations(unitset)
		if current == nil {
			current = map[string]string{}
		}

		for _, one := range units {
			if one.Status.NodeName == "" {
				continue
			}
			if current[one.Name] == upmiov1alpha2.NoneSetFlag {
				continue
			}
			current[one.Name] = one.Status.NodeName
		}

		_ = setNodeNameMapToAnnotations(unitset, current)
		if originalUnitset.Annotations == nil || originalUnitset.Annotations[upmiov1alpha2.AnnotationUnitsetNodeNameMap] != unitset.Annotations[upmiov1alpha2.AnnotationUnitsetNodeNameMap] {
			_, _ = r.patchUnitset(ctx, originalUnitset, unitset)
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
	if len(units) != 0 {
		for _, one := range units {
			if one.Status.Phase == upmiov1alpha2.UnitReady {
				unitset.Status.ReadyUnits++
			}
		}
	}

	unitPVCSyncedCount := 0
	unitImageSyncedCount := 0
	unitResourceSyncedCount := 0

	if len(units) != 0 {
		for _, one := range units {
			if one.Status.Task != "" {
				if inTaskUnit == "" {
					inTaskUnit = one.Name
				} else {
					inTaskUnit = inTaskUnit + "," + one.Name
				}
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

	if inTaskUnit != "" {
		unitset.Status.InUpdate = inTaskUnit
	} else if unitImageSyncedCount == orig.Spec.Units && unitResourceSyncedCount == orig.Spec.Units {
		unitset.Status.InUpdate = ""
	} else {
		unitset.Status.InUpdate = ""
	}

	//if equality.Semantic.DeepEqual(orig.Status, unitset.Status) {
	//	return nil
	//}

	serviceList, err := r.listServiceBelongUnitset(ctx, unitset)
	if err != nil {
		klog.Errorf("[reconcileUnitsetStatus]failed to list service for unitset [%s], error: [%v]", req.String(), err.Error())
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

	// external service
	if unitset.Spec.ExternalService.Type != "" {
		for _, one := range serviceList {
			if _, ok := one.Labels[upmiov1alpha2.AnnotationExternalServiceType]; ok {
				unitset.Status.ExternalService.Name = one.Name
				break
			}
		}
	}

	// unit service
	if unitset.Spec.UnitService.Type != "" {
		unitServiceMap := make(map[string]string)

		for _, unit := range units {
			for _, service := range serviceList {
				if unit.Name == service.Labels[upmiov1alpha2.UnitName] {
					unitServiceMap[unit.Name] = service.Name
				}
			}
		}

		unitset.Status.UnitService.Name = unitServiceMap
	}

	if unitset.Status.Units != orig.Status.Units ||
		unitset.Status.ReadyUnits != orig.Status.ReadyUnits ||
		unitset.Status.ImageSyncStatus.Status != orig.Status.ImageSyncStatus.Status ||
		unitset.Status.ResourceSyncStatus.Status != orig.Status.ResourceSyncStatus.Status ||
		unitset.Status.PvcSyncStatus.Status != orig.Status.PvcSyncStatus.Status ||
		unitset.Status.ExternalService.Name != orig.Status.ExternalService.Name ||
		equality.Semantic.DeepEqual(unitset.Status.UnitService.Name, orig.Status.UnitService.Name) {

		return r.Status().Update(ctx, unitset)
	}

	return nil
}

func (r *UnitSetReconciler) patchUnitset(ctx context.Context, old, _new *upmiov1alpha2.UnitSet) (*upmiov1alpha2.UnitSet, error) {
	patchData, update, err := patch.GenerateMergePatch(old, _new, upmiov1alpha2.UnitSet{})
	if err != nil || !update {
		return old, err
	}

	//r.Recorder.Eventf(old, coreV1.EventTypeNormal, "ResourceCheck", "patch unitset ok~ (data: %s)", patchData)

	err = r.Patch(ctx, old, client.RawPatch(types.MergePatchType, patchData), &client.PatchOptions{})
	if err != nil {
		r.Recorder.Eventf(old, coreV1.EventTypeWarning, "PatchFailed", "patch unitset error: (data: %s), [ERROR:%s]", patchData, err.Error())
		return nil, err
	}

	return _new, nil
}

// getNodeNameMapFromAnnotations reads the UnitSet annotation upm.io/node-name-map
// and returns the parsed map. Returns empty map when absent or invalid.
func getNodeNameMapFromAnnotations(unitset *upmiov1alpha2.UnitSet) map[string]string {
	out := map[string]string{}
	if unitset == nil {
		return out
	}
	if unitset.Annotations == nil {
		return out
	}
	data, ok := unitset.Annotations[upmiov1alpha2.AnnotationUnitsetNodeNameMap]
	if !ok || data == "" {
		return out
	}
	_ = json.Unmarshal([]byte(data), &out)
	return out
}

// setNodeNameMapToAnnotations marshals and stores the map into UnitSet annotations.
func setNodeNameMapToAnnotations(unitset *upmiov1alpha2.UnitSet, m map[string]string) error {
	if unitset.Annotations == nil {
		unitset.Annotations = map[string]string{}
	}

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	unitset.Annotations[upmiov1alpha2.AnnotationUnitsetNodeNameMap] = string(b)
	return nil
}
