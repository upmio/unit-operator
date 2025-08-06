package unit

import (
	"context"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	internalAgent "github.com/upmio/unit-operator/pkg/client/unit-agent"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
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

	if pod != nil {
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
			processState, err = internalAgent.GetServiceProcessState(
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

	if pvcs != nil && len(pvcs) != 0 {
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

	if equality.Semantic.DeepEqual(orig.Status, unit.Status) {
		return nil
	}

	return r.Status().Update(ctx, unit)
}

func (r *UnitReconciler) unitManagedResources(
	ctx context.Context,
	req ctrl.Request,
	unit *upmiov1alpha2.Unit) (
	*v1.Pod,
	[]*v1.PersistentVolumeClaim,
	*v1.Node,
	error) {
	//pod, pvc, node
	po := &v1.Pod{}
	node := &v1.Node{}
	claims := []*v1.PersistentVolumeClaim{}

	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, po)
	if err != nil {
		if apierrors.IsNotFound(err) {
			po = nil
		}
		return nil, nil, nil, fmt.Errorf("get unit pod failed, error: [%s]", err.Error())
	}

	if po == nil {
		node = nil
	} else {
		nodeName := po.Spec.NodeName
		err = r.Get(ctx, client.ObjectKey{Name: nodeName}, node)
		if err != nil {
			if apierrors.IsNotFound(err) {
				node = nil
			}
			return nil, nil, nil, fmt.Errorf("get unit node failed, error: [%s]", err.Error())
		}
	}

	if unit.Spec.VolumeClaimTemplates != nil && len(unit.Spec.VolumeClaimTemplates) != 0 {
		for _, one := range unit.Spec.VolumeClaimTemplates {
			claim := &v1.PersistentVolumeClaim{}

			pvcName := upmiov1alpha2.PersistentVolumeClaimName(unit, one.Name)
			err = r.Get(ctx, client.ObjectKey{Name: pvcName, Namespace: unit.Namespace}, claim)
			if err != nil {
				if apierrors.IsNotFound(err) {
					claim = nil
				}
				return nil, nil, nil, fmt.Errorf("get unit pvc:[%s] failed, error: [%s]", pvcName, err.Error())
			}
			claims = append(claims, claim)
		}
	}

	return po, claims, node, nil
}
