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

func (r *UnitReconciler) reconcileUnitConfig(ctx context.Context, unit *upmiov1alpha2.Unit) error {
	maintenanceValue, ok := unit.GetAnnotations()[upmiov1alpha2.AnnotationMaintenance]
	if ok && maintenanceValue == "true" {
		klog.Info("[reconcileUnitConfig] unit is in maintenance mode, skip reconcile")
		r.Recorder.Eventf(unit, v1.EventTypeNormal, "ResourceCheck", "unit is in maintenance mode, skip reconcile")
		return nil
	}

	pod := &v1.Pod{}
	podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
	err := r.Get(ctx, podNamespacedName, pod)
	if err != nil {
		return err
	}

	if !podutil.IsPodInitialized(pod) {
		klog.Errorf("[reconcileUnitConfig] pod not initialized, not support sync unit config")
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "pod not initialized, not support sync unit config")
		return nil
	}

	// pod ready, not need sync config
	if podutil.IsContainerRunningAndReady(pod, unit.MainContainerName()) || !unit.Spec.Startup {
		return nil
	}

	// container [unit-agent] not ready, not need sync config
	if !podutil.IsContainerRunningAndReady(pod, vars.UnitAgentName) {
		klog.Errorf("[reconcileUnitConfig] container [unit-agent] not ready, not support sync unit config")
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "container [unit-agent] not ready, not support sync unit config")
		return nil
	}

	if len(pod.Status.PodIPs) == 0 {
		return fmt.Errorf("sync unit config failed: no pod ip to used")
	}

	agentHost := ""
	switch vars.UnitAgentHostType {
	case "domain":
		agentHost = unit.Name
	case "ip":
		agentHost = pod.Status.PodIPs[0].IP
	}

	extendConfigName := []string{}
	//extendConfigName = append(extendConfigName, unit.Spec.SharedConfigName) //}

	message, err := internalAgent.SyncConfig(
		vars.UnitAgentHostType,
		upmiov1alpha2.UnitsetHeadlessSvcName(unit),
		agentHost,
		"2214",
		unit.Namespace,
		unit.Spec.ConfigTemplateName,
		unit.Spec.ConfigValueName,
		unit.MainContainerName(),
		extendConfigName)
	if err != nil {
		//r.EventRecorder.Eventf(unit, corev1.EventTypeWarning, ErrResourceExists, "sync unit config failed: message:[%s], error:[%s]", message, err.Error())
		return fmt.Errorf("sync unit config failed: message:[%s], error:[%s]", message, err.Error())
	}

	return nil
}
