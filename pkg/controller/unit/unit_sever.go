package unit

import (
	"context"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	internalAgent "github.com/upmio/unit-operator/pkg/client/unit-agent"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcileUnitServer(ctx context.Context, unit *upmiov1alpha2.Unit) error {
	maintenanceValue, ok := unit.GetAnnotations()[upmiov1alpha2.AnnotationMaintenance]
	if ok && maintenanceValue == "true" {
		klog.Info("[reconcileUnitServer] unit is in maintenance mode, skip reconcile")
		r.Recorder.Eventf(unit, v1.EventTypeNormal, "ResourceCheck", "unit is in maintenance mode, skip reconcile")
		return nil
	}

	pod := &v1.Pod{}
	podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
	err := r.Get(ctx, podNamespacedName, pod)
	if err != nil {
		return err
	}

	if !podutil.IsPodInitialized(pod) || !podutil.IsPodScheduled(pod) {
		klog.Errorf("[reconcileUnitServer] pod not initialized or scheduled, not support unit lifecycle management")
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "pod not initialized or scheduled, not support unit lifecycle management")
		return nil
	}

	// container [unit-agent] not ready, not support unit lifecycle management
	if !podutil.IsContainerRunningAndReady(pod, vars.UnitAgentName) {
		klog.Errorf("[reconcileUnitConfig] container [unit-agent] not ready, not support unit lifecycle management")
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "container [unit-agent] not ready, not support unit lifecycle management")
		return nil
	}

	if len(pod.Status.PodIPs) == 0 {
		//r.EventRecorder.Eventf(unit, corev1.EventTypeWarning, ErrResourceExists, "unit lifecycle management failed: [no pod ip to use]")
		return fmt.Errorf("unit lifecycle management failed: [no pod ip to use]")
	}

	agentHost := ""
	switch vars.UnitAgentHostType {
	case "domain":
		agentHost = unit.Name
	case "ip":
		agentHost = pod.Status.PodIPs[0].IP
	}

	if unit.Spec.Startup {
		if podutil.IsContainerRunningAndReady(pod, unit.MainContainerName()) &&
			(unit.Status.ProcessState == "running" || unit.Status.ProcessState == "starting") {
			return nil
		}

		resp, err := internalAgent.ServiceLifecycleManagement(
			vars.UnitAgentHostType,
			upmiov1alpha2.UnitsetHeadlessSvcName(unit),
			agentHost,
			unit.Namespace,
			"2214",
			"start")

		if err != nil {
			//r.EventRecorder.Eventf(unit, corev1.EventTypeWarning, ErrResourceExists, "fail to start unit: message:[%s], error:[%s]", resp, err.Error())
			return fmt.Errorf("fail to start unit: message:[%s], error:[%s]", resp, err.Error())
		}

		return nil
	}

	if !podutil.IsPodReady(pod) &&
		unit.Status.ProcessState != "running" &&
		unit.Status.ProcessState != "starting" {
		return nil
	}

	resp, err := internalAgent.ServiceLifecycleManagement(
		vars.UnitAgentHostType,
		upmiov1alpha2.UnitsetHeadlessSvcName(unit),
		agentHost,
		unit.Namespace,
		"2214",
		"stop")
	if err != nil {
		return fmt.Errorf("fail to stop unit: message:[%s], error:[%s]", resp, err.Error())
	}

	return nil
}
