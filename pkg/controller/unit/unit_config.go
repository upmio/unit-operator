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
	"k8s.io/client-go/util/retry"
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

	// check need reload config
	if podutil.IsRunning(pod) {
		reloadErr := r.reloadUnitConfig(ctx, req, unit, pod)
		if reloadErr != nil {
			klog.Errorf("[reconcileUnitConfig] reload unit config failed: %s", reloadErr.Error())
			return reloadErr
		}
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

	var timeout int
	var periodSeconds int
	for _, one := range pod.Spec.Containers {
		if one.Name == unit.MainContainerName() {
			periodSeconds = int(one.ReadinessProbe.PeriodSeconds)
			timeout = int(one.ReadinessProbe.PeriodSeconds*one.ReadinessProbe.SuccessThreshold + one.ReadinessProbe.TimeoutSeconds)
			break
		}
	}

	var lastTimeMessage string
	var syncConfigErr error

	waitErr := wait.PollUntilContextTimeout(ctx, time.Duration(periodSeconds)*time.Second, time.Duration(timeout)*time.Second, false, func(ctx context.Context) (bool, error) {

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
		if err != nil {
			return fmt.Errorf("[reconcileUnitConfig]:wait for sync config timeout, recreate pod failed: %s", err.Error())
		}

		return fmt.Errorf("[reconcileUnitConfig]: because sync config timeout, recreate pod ok, wait for next reconcile to sync config")
	}

	// sync ConfigTemplate and ConfigValue version to unit annotation
	// need get ConfigTemplate and ConfigValue version from configmap
	configTemplateCm := v1.ConfigMap{}
	configTemplateCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigTemplateName, Namespace: req.Namespace}, &configTemplateCm)
	if configTemplateCmErr != nil {
		return fmt.Errorf("[reconcileUnitConfig]: sync config success, but get configTemplate cm:[%s] failed: %s, patch config template version to unit failed", unit.Spec.ConfigTemplateName, configTemplateCmErr.Error())
	}

	configValueCm := v1.ConfigMap{}
	configValueCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigValueName, Namespace: req.Namespace}, &configValueCm)
	if configValueCmErr != nil {
		return fmt.Errorf("[reconcileUnitConfig]: sync config success, but get configValue cm:[%s] failed: %s, patch config value version to unit failed", unit.Spec.ConfigValueName, configValueCmErr.Error())
	}

	if unit.Annotations == nil {
		unit.Annotations = make(map[string]string)
	}

	unit.Annotations[upmiov1alpha2.AnnotationConfigTemplateVersion] = configTemplateCm.ResourceVersion
	unit.Annotations[upmiov1alpha2.AnnotationConfigValueVersion] = configValueCm.ResourceVersion

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.Update(ctx, unit)
	})
	if err != nil {
		return fmt.Errorf("[reconcileUnitConfig]: sync unit config failed: %s", err.Error())
	}

	return nil
}

func (r *UnitReconciler) reloadUnitConfig(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit, pod *v1.Pod) error {
	configTemplateVersion, configTemplateOk := unit.Annotations[upmiov1alpha2.AnnotationConfigTemplateVersion]
	configValueVersion, configValueOk := unit.Annotations[upmiov1alpha2.AnnotationConfigValueVersion]
	if !configTemplateOk || !configValueOk {
		// no annotation no need to update
		// reload func will be called after sync config succeed
		klog.Infof("[reloadUnitConfig] unit:[%s] no config version annotation, no need to reload config", req.String())
		return nil
	}

	configTemplateCm := v1.ConfigMap{}
	configTemplateCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigTemplateName, Namespace: req.Namespace}, &configTemplateCm)
	if configTemplateCmErr != nil {
		return fmt.Errorf("[reconcileUnitConfig]: sync config success, but get configTemplate cm:[%s] failed: %s, patch config template version to unit failed", unit.Spec.ConfigTemplateName, configTemplateCmErr.Error())
	}

	configValueCm := v1.ConfigMap{}
	configValueCmErr := r.Get(ctx, client.ObjectKey{Name: unit.Spec.ConfigValueName, Namespace: req.Namespace}, &configValueCm)
	if configValueCmErr != nil {
		return fmt.Errorf("[reconcileUnitConfig]: sync config success, but get configValue cm:[%s] failed: %s, patch config value version to unit failed", unit.Spec.ConfigValueName, configValueCmErr.Error())
	}

	if configTemplateVersion != configTemplateCm.ResourceVersion || configValueVersion != configValueCm.ResourceVersion {

		// container [unit-agent] not ready, not need sync config
		if !podutil.IsContainerRunningAndReady(pod, vars.UnitAgentName) {
			klog.Errorf("[reconcileUnitConfig-reloadUnitConfig] container [unit-agent] not ready, not support [reload] unit config")
			r.Recorder.Eventf(unit, v1.EventTypeWarning, "ResourceCheck", "container [unit-agent] not ready, not support [reload] unit config")
			return nil
		}

		if len(pod.Status.PodIPs) == 0 {
			return fmt.Errorf("[reloadUnitConfig] reload unit config failed: no pod ip to used")
		}

		agentHost := ""
		switch vars.UnitAgentHostType {
		case "domain":
			agentHost = unit.Name
		case "ip":
			agentHost = pod.Status.PodIPs[0].IP
		}

		message, syncConfigErr := internalAgent.SyncConfig(
			vars.UnitAgentHostType,
			upmiov1alpha2.UnitsetHeadlessSvcName(unit),
			agentHost,
			"2214",
			unit.Namespace,
			unit.Spec.ConfigTemplateName,
			unit.Spec.ConfigValueName,
			unit.MainContainerName(),
			[]string{})

		if syncConfigErr != nil {
			return fmt.Errorf("[reloadUnitConfig]: reload unit config failed:[%s], message:[%s]", syncConfigErr.Error(), message)
		}

		unit.Annotations[upmiov1alpha2.AnnotationConfigTemplateVersion] = configTemplateCm.ResourceVersion
		unit.Annotations[upmiov1alpha2.AnnotationConfigValueVersion] = configValueCm.ResourceVersion

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, unit)
		})
		if err != nil {
			return fmt.Errorf("[reconcileUnitConfig]: sync unit config failed: %s", err.Error())
		}

	}

	return nil
}
