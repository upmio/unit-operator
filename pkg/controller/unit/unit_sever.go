package unit

import (
	"context"
	"fmt"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	internalAgent "github.com/upmio/unit-operator/pkg/client/unit-agent"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcileUnitServer(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	maintenanceValue, ok := unit.GetAnnotations()[upmiov1alpha2.AnnotationMaintenance]
	if ok && maintenanceValue == "true" {
		klog.Info("[reconcileUnitServer] unit is in maintenance mode, skip reconcile")
		r.Recorder.Eventf(unit, v1.EventTypeNormal, "ResourceCheck", "unit is in maintenance mode, skip reconcile")
		return nil
	}

	getUnitErr := r.Get(ctx, req.NamespacedName, unit)
	if getUnitErr != nil {
		return getUnitErr
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
		klog.Errorf("[reconcileUnitServer] container [unit-agent] not ready, not support unit lifecycle management")
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "container [unit-agent] not ready, not support unit lifecycle management")
		return nil
	}

	if len(pod.Status.PodIPs) == 0 {
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

		klog.Infof("[reconcileUnitServer] unit:[%s] unit.spec.startup=true, will execute [start]",
			req.NamespacedName.String())

		resp, startErr := internalAgent.ServiceLifecycleManagement(
			vars.UnitAgentHostType,
			upmiov1alpha2.UnitsetHeadlessSvcName(unit),
			agentHost,
			unit.Namespace,
			"2214",
			"start")

		if startErr != nil {
			klog.Errorf("[reconcileUnitServer] unit:[%s] EXECUTE [start] error:[%s]",
				req.NamespacedName.String(), startErr.Error())

			return fmt.Errorf("fail to start unit: message:[%s], error:[%s]", resp, startErr.Error())
		}

		var timeout int
		var periodSeconds int
		for _, one := range pod.Spec.Containers {
			if one.Name == unit.MainContainerName() {
				periodSeconds = int(one.ReadinessProbe.PeriodSeconds)
				timeout = int(one.ReadinessProbe.PeriodSeconds*one.ReadinessProbe.SuccessThreshold + one.ReadinessProbe.TimeoutSeconds)
				break
			}
		}

		klog.Infof("[reconcileUnitServer] unit:[%s] EXECUTE [start] ok, will wait for pod running and ready, [%d]s/[%d]s",
			req.String(), periodSeconds, timeout)

		waitErr := wait.PollUntilContextTimeout(ctx, time.Duration(periodSeconds)*time.Second, time.Duration(timeout)*time.Second, false, func(ctx context.Context) (bool, error) {
			newPod := &v1.Pod{}
			newPodErr := r.Get(ctx, req.NamespacedName, newPod)
			if newPodErr != nil {
				return false, nil
			}

			if !podutil.IsRunningAndReady(newPod) {
				return false, nil
			}

			return true, nil
		})

		if waitErr != nil {
			klog.Warningf("[reconcileUnitServer] [start service] unit:[%s] start up timeout, will trrigger [stop service] and then redo [start up]",
				req.String())

			r.Recorder.Eventf(unit, v1.EventTypeWarning, "StartUp", "[start up] timeout, will trrigger [stop] and then redo [start]")

			stopMessage, stopErr := internalAgent.ServiceLifecycleManagement(
				vars.UnitAgentHostType,
				upmiov1alpha2.UnitsetHeadlessSvcName(unit),
				agentHost,
				unit.Namespace,
				"2214",
				"stop")

			if stopErr != nil {
				return fmt.Errorf("fail to stop unit: message:[%s], error:[%s]", stopMessage, stopErr.Error())
			}

			return nil
		}

		klog.Infof("[reconcileUnitServer] unit:[%s] execute [start] ok~ ",
			req.NamespacedName.String())

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
