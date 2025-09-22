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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcileUnitConfig(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
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

	var lastTimeMessage string
	var syncConfigErr error

	waitErr := wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {

		lastTimeMessage, syncConfigErr = internalAgent.SyncConfig(
			vars.UnitAgentHostType,
			upmiov1alpha2.UnitsetHeadlessSvcName(unit),
			agentHost,
			"2214",
			unit.Namespace,
			unit.Spec.ConfigTemplateName,
			unit.Spec.ConfigValueName,
			unit.MainContainerName(),
			extendConfigName)

		if syncConfigErr != nil {
			//return false,fmt.Errorf("sync unit config failed: message:[%s], error:[%s]", message, err.Error())
			return false, nil
		}

		return true, nil

	})

	if waitErr != nil {
		klog.Warningf("[reconcileUnitConfig] unit:[%s] wait for sync config timeout, message:[%s], error:[%s], will trrigger recreate pod",
			req.String(), lastTimeMessage, syncConfigErr.Error())

		r.Recorder.Eventf(unit, v1.EventTypeWarning, "SyncConfigFailed",
			"[reconcileUnitConfig] wait for sync config timeout, message:[%s], error:[%s], will trrigger recreate pod",
			lastTimeMessage, syncConfigErr.Error())

		// delete pod
		err = r.Delete(ctx, pod)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		// wait delete
		err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			pod := &v1.Pod{}
			podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
			err := r.Get(ctx, podNamespacedName, pod)
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, nil
		})

		if err != nil {
			return fmt.Errorf("[reconcileUnitConfig]: wait pod delete fail:%s", err.Error())
		}

		// create
		newPod, _ := r.convert2Pod(ctx, unit)

		err = r.Create(ctx, newPod)
		if err == nil {
			r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessCreated", "[reconcileUnitConfig]: recreate pod [%s] ok", pod.Name)
		}

		return err
	}

	return nil
}
