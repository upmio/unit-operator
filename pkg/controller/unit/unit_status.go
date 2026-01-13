package unit

import (
	"context"
	"fmt"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcileUnitStatus(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	orig := unit.DeepCopy()

	pod, pvcs, node, err := r.unitManagedResources(ctx, req, unit)
	if err != nil {
		klog.Errorf("[reconcileUnitStatus] get unit:[%s] managed resources failed, error: [%s]", req.String(), err.Error())

		unit.Status = upmiov1alpha2.UnitStatus{
			Phase:                 "",
			NodeReady:             "",
			NodeName:              "",
			ProcessState:          "",
			HostIP:                "",
			PodIPs:                nil,
			PersistentVolumeClaim: nil,
		}
	}

	if pod != nil && pod.DeletionTimestamp.IsZero() {
		unit.Status.HostIP = pod.Status.HostIP
		unit.Status.PodIPs = pod.Status.PodIPs

		if pod.Status.Phase == v1.PodRunning {
			if podutil.IsPodReady(pod) {
				unit.Status.Phase = upmiov1alpha2.UnitReady
				unit.Status.Task = ""
			} else {
				unit.Status.Phase = upmiov1alpha2.UnitRunning
			}
		} else {
			unit.Status.Phase = upmiov1alpha2.UnitPhase(pod.Status.Phase)
		}

		agentHost := ""
		switch vars.UnitAgentHostType {
		case "domain":
			agentHost = unit.Name
		case "ip":
			agentHost = pod.Status.PodIPs[0].IP
		}

		processState := ""
		if podutil.IsContainerRunningAndReady(pod, vars.UnitAgentName) {
			agent := r.Agent
			if agent == nil {
				agent = defaultUnitAgentClient{}
			}

			processState, err = agent.GetServiceProcessState(
				vars.UnitAgentHostType,
				upmiov1alpha2.UnitsetHeadlessSvcName(unit),
				agentHost,
				req.Namespace,
				"2214")

			if err != nil {
				klog.Errorf("get unit agent process state failed, error: [%s]", err.Error())
			}
		}

		unit.Status.ProcessState = processState

	}

	if node != nil {
		nodeReadyStatus := "False"
		for _, each := range node.Status.Conditions {
			if each.Type == v1.NodeReady {
				if each.Status == v1.ConditionTrue {
					nodeReadyStatus = "True"
					break
				}
			}
		}

		unit.Status.NodeReady = nodeReadyStatus
		unit.Status.NodeName = node.Name
	}

	if len(pvcs) != 0 {
		pvcInfo := []upmiov1alpha2.PvcInfo{}
		for _, claim := range pvcs {
			onePvc := upmiov1alpha2.PvcInfo{
				Name:        claim.Name,
				VolumeName:  claim.Spec.VolumeName,
				AccessModes: claim.Spec.AccessModes,
				Capacity: upmiov1alpha2.PvcCapacity{
					Storage: *claim.Status.Capacity.Storage(),
				},
				Phase: claim.Status.Phase,
			}
			pvcInfo = append(pvcInfo, onePvc)
		}

		unit.Status.PersistentVolumeClaim = pvcInfo
	}

	// ConfigTemplateName/ConfigValueName are required for Unit status sync checks.
	// If they are empty, fail fast with a clear error (and avoid k8s client "resource name may not be empty").
	if unit.Spec.ConfigTemplateName == "" || unit.Spec.ConfigValueName == "" {
		unit.Status.ConfigSyncStatus = upmiov1alpha2.ConfigSyncStatus{
			LastTransitionTime: metaV1.Now(),
			Status:             "False",
		}

		if hasUnitStatusChanged(orig.Status, unit.Status) {
			_ = r.Status().Update(ctx, unit)
		}

		return fmt.Errorf("[reconcileUnitStatus] required config names are empty: configTemplateName=%q configValueName=%q", unit.Spec.ConfigTemplateName, unit.Spec.ConfigValueName)
	}

	configTemplateCm := v1.ConfigMap{}
	configTemplateCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigTemplateName, Namespace: req.Namespace}, &configTemplateCm)
	if configTemplateCmErr != nil {
		return fmt.Errorf("[reconcileUnitStatus] get configTemplate cm:[%s] failed: %s", unit.Spec.ConfigTemplateName, configTemplateCmErr.Error())
	}

	configValueCm := v1.ConfigMap{}
	configValueCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigValueName, Namespace: req.Namespace}, &configValueCm)
	if configValueCmErr != nil {
		return fmt.Errorf("[reconcileUnitStatus] get configValue cm:[%s] failed: %s", unit.Spec.ConfigValueName, configValueCmErr.Error())
	}

	newConfigSyncStatus := "True"
	// If annotations are missing/empty, we cannot prove the unit has synced configs; treat as not synced.
	if unit.Annotations == nil {
		newConfigSyncStatus = "False"
	} else {
		tv, okT := unit.Annotations[upmiov1alpha2.AnnotationConfigTemplateVersion]
		vv, okV := unit.Annotations[upmiov1alpha2.AnnotationConfigValueVersion]
		if !okT || !okV {
			newConfigSyncStatus = "False"
		} else if tv != configTemplateCm.ResourceVersion || vv != configValueCm.ResourceVersion {
			newConfigSyncStatus = "False"
		}
	}

	if orig.Status.ConfigSyncStatus.Status != newConfigSyncStatus {
		unit.Status.ConfigSyncStatus = upmiov1alpha2.ConfigSyncStatus{
			LastTransitionTime: metaV1.Now(),
			Status:             newConfigSyncStatus,
		}
	} else {
		unit.Status.ConfigSyncStatus = orig.Status.ConfigSyncStatus
	}

	if hasUnitStatusChanged(orig.Status, unit.Status) {
		return r.Status().Update(ctx, unit)
	}

	return nil
}

// hasStatusChanged 比较状态是否真正发生变化，忽略时间戳字段的差异
func hasUnitStatusChanged(origStatus, newStatus upmiov1alpha2.UnitStatus) bool {
	origCopy := origStatus.DeepCopy()
	newCopy := newStatus.DeepCopy()

	if origCopy.ConfigSyncStatus.Status == newCopy.ConfigSyncStatus.Status {
		newCopy.ConfigSyncStatus.LastTransitionTime = origCopy.ConfigSyncStatus.LastTransitionTime
	}

	return !equality.Semantic.DeepEqual(origCopy, newCopy)
}

func (r *UnitReconciler) reconcileUnitObservedGeneration(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &upmiov1alpha2.Unit{}
		if err := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: req.Namespace}, latest); err != nil {
			return err
		}

		latest.Status.ObservedGeneration = latest.Generation
		return r.Status().Update(ctx, latest)
	})

	if retryErr != nil {
		return fmt.Errorf("update unit ObservedGeneration error: [%s]", retryErr.Error())
	}

	return nil
}

func (r *UnitReconciler) unitManagedResources(
	ctx context.Context,
	req ctrl.Request,
	unit *upmiov1alpha2.Unit) (
	*v1.Pod,
	[]*v1.PersistentVolumeClaim,
	*v1.Node,
	error) {

	// Pod
	po := &v1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, po)
	if err != nil {
		if apierrors.IsNotFound(err) {
			po = nil
		} else {
			return nil, nil, nil, fmt.Errorf("get unit pod failed, error: [%s]", err.Error())
		}
	}

	// Node
	var node *v1.Node
	if po != nil && po.Spec.NodeName != "" {
		node = &v1.Node{}
		err = r.Get(ctx, client.ObjectKey{Name: po.Spec.NodeName}, node)
		if err != nil {
			if apierrors.IsNotFound(err) {
				node = nil
			} else {
				return nil, nil, nil, fmt.Errorf("get unit node failed, error: [%s]", err.Error())
			}
		}
	}

	// PVCs
	var claims []*v1.PersistentVolumeClaim
	if unit.Spec.VolumeClaimTemplates != nil {
		for _, template := range unit.Spec.VolumeClaimTemplates {
			claim := &v1.PersistentVolumeClaim{}
			pvcName := upmiov1alpha2.PersistentVolumeClaimName(unit, template.Name)

			err = r.Get(ctx, client.ObjectKey{Name: pvcName, Namespace: unit.Namespace}, claim)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				} else {
					return nil, nil, nil, fmt.Errorf("get unit pvc:[%s] failed, error: [%s]", pvcName, err.Error())
				}
			}
			claims = append(claims, claim)
		}
	}

	return po, claims, node, nil
}
