package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitReconciler) reconcilePod(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {

	unitGetErr := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: req.Namespace}, unit)
	if unitGetErr != nil {
		return fmt.Errorf("[reconcilePod] get unit error:[%s]", unitGetErr.Error())
	}

	pod := &v1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: req.Namespace}, pod)
	if apierrors.IsNotFound(err) {

		// if not found, generate from template
		pod, _ = r.convert2Pod(ctx, unit)

		err = r.Create(ctx, pod)
		if err != nil {
			return err
		}

		return nil

	} else if err != nil {
		return err
	}

	if !pod.DeletionTimestamp.IsZero() {
		return fmt.Errorf("pod [%s] is marked for deleted", pod.Name)
	}

	// update mem,cpu,image,env or node_affinity fail will trigger recreate pod
	reason, needUpgradePod := ifNeedUpgradePod(unit, pod)
	if needUpgradePod {
		klog.Infof("need upgrade pod, reason: %s", reason)

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			orig := unit.DeepCopy()

			orig.Status.Phase = ""
			orig.Status.HostIP = ""
			orig.Status.PodIPs = nil
			orig.Status.Task = reason
			orig.Status.ObservedGeneration = unit.Generation

			return r.Status().Update(ctx, orig)

		})
		if err != nil {
			return fmt.Errorf("[reconcilePod] update unit status fail before [upgradePod], error: [%s]", err.Error())
		}

		// mem,cpu support update without recreate pod, in place vertical scaling
		if (reason == "cpu changed" || reason == "memory changed") && vars.InPlacePodVerticalScalingEnabled {
			err := r.resizePod(ctx, unit, pod)
			if err != nil {
				return fmt.Errorf("[reconcilePod] resize pod error: [%s]", err.Error())
			}
			r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessUpdated", "resize pod [%s] ok~", pod.Name)

			return nil
		}

		err = r.upgradePod(ctx, req, unit, pod, reason)
		if err != nil {
			return err
		}
	}

	// sync label, not image here
	ifNeedUpdateObservedGeneration := false
	patch, need, err := ifNeedPatchPod(unit, pod)

	if need {
		ifNeedUpdateObservedGeneration = true
		err = r.Patch(ctx, pod, client.RawPatch(types.StrategicMergePatchType, patch))
		if err == nil {
			r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessUpdated", "patch pod [%s] ok~ (patch data: %s)", pod.Name, string(patch))
		} else {
			r.Recorder.Eventf(unit, v1.EventTypeWarning, "ErrResourceExists", "Patch pod [%s] fail:[%s] (patch data: %s)", pod.Name, err.Error(), string(patch))
		}
	}

	if err != nil {
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ErrResourceExists", "check patch pod fail:[%s]", err.Error())
	}

	if ifNeedUpdateObservedGeneration {
		err := r.reconcileUnitObservedGeneration(ctx, req, unit)
		if err != nil {
			return fmt.Errorf("[update pvc] error: [%s]", err.Error())
		}
	}

	return nil
}

func (r *UnitReconciler) resizePod(ctx context.Context, unit *upmiov1alpha2.Unit, pod *v1.Pod) error {
	mainContainerResources := v1.ResourceRequirements{}
	for _, container := range unit.Spec.Template.Spec.Containers {
		if container.Name == unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] {
			mainContainerResources = container.Resources
			break
		}
	}

	resizePolicy := []v1.ContainerResizePolicy{
		{
			ResourceName:  "cpu",
			RestartPolicy: v1.RestartContainer,
		},
		{
			ResourceName:  "memory",
			RestartPolicy: v1.RestartContainer,
		},
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []map[string]interface{}{
				{
					"name":         unit.Annotations[upmiov1alpha2.AnnotationMainContainerName],
					"resources":    mainContainerResources,
					"resizePolicy": resizePolicy,
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	return r.SubResource("resize").Patch(ctx,
		&v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name}},
		client.RawPatch(types.MergePatchType, patchBytes),
	)
}

func ifNeedPatchPod(unit *upmiov1alpha2.Unit, pod *v1.Pod) ([]byte, bool, error) {

	updatePod := generatePatchPod(unit, pod)

	modJson, err := json.Marshal(updatePod)
	if err != nil {
		return []byte{}, false, err
	}

	curJson, err := json.Marshal(pod)
	if err != nil {
		return []byte{}, false, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, v1.Pod{})
	if err != nil {
		return []byte{}, false, err
	}

	if len(patch) == 0 || string(patch) == "{}" {
		return patch, false, nil
	}
	return patch, true, nil
}

func generatePatchPod(unit *upmiov1alpha2.Unit, curPod *v1.Pod) *v1.Pod {

	clone := curPod.DeepCopy()

	for key, value := range unit.Labels {
		if clone.Labels == nil {
			clone.Labels = make(map[string]string)
		}

		if clone.Labels[key] != value {
			clone.Labels[key] = value
		}
	}

	for key, value := range unit.Annotations {
		if clone.Annotations == nil {
			clone.Annotations = make(map[string]string)
		}
		if clone.Annotations[key] != value {
			clone.Annotations[key] = value
		}
	}

	if unit.Spec.Template.Spec.NodeName == "" && curPod.Spec.NodeName != "" {
		clone.Spec.NodeName = curPod.Spec.NodeName
	} else if unit.Spec.Template.Spec.NodeName != "" && unit.Spec.Template.Spec.NodeName != curPod.Spec.NodeName {
		clone.Spec.NodeName = unit.Spec.Template.Spec.NodeName
	}

	for i := range unit.Spec.Template.Spec.Containers {
		for j := range clone.Spec.Containers {
			if unit.Spec.Template.Spec.Containers[i].Name == clone.Spec.Containers[j].Name {
				// Update non-main container images
				if unit.Spec.Template.Spec.Containers[i].Name != unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] &&
					clone.Spec.Containers[j].Image != unit.Spec.Template.Spec.Containers[i].Image {
					clone.Spec.Containers[j].Image = unit.Spec.Template.Spec.Containers[i].Image
				}

				// Sync environment variables for all containers (including main container)
				// This ensures that environment variable changes are applied without pod recreation
				if !envVarsEqual(unit.Spec.Template.Spec.Containers[i].Env, clone.Spec.Containers[j].Env) {
					clone.Spec.Containers[j].Env = make([]v1.EnvVar, len(unit.Spec.Template.Spec.Containers[i].Env))
					copy(clone.Spec.Containers[j].Env, unit.Spec.Template.Spec.Containers[i].Env)
				}
			}
		}
	}

	// Sync environment variables for init containers
	for i := range unit.Spec.Template.Spec.InitContainers {
		for j := range clone.Spec.InitContainers {
			if unit.Spec.Template.Spec.InitContainers[i].Name == clone.Spec.InitContainers[j].Name {
				// Sync environment variables for init containers
				if !envVarsEqual(unit.Spec.Template.Spec.InitContainers[i].Env, clone.Spec.InitContainers[j].Env) {
					clone.Spec.InitContainers[j].Env = make([]v1.EnvVar, len(unit.Spec.Template.Spec.InitContainers[i].Env))
					copy(clone.Spec.InitContainers[j].Env, unit.Spec.Template.Spec.InitContainers[i].Env)
				}
			}
		}
	}

	return clone
}

// envVarsEqual compares two slices of EnvVar and returns true if they are equal
func envVarsEqual(a, b []v1.EnvVar) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps for efficient comparison
	aMap := make(map[string]v1.EnvVar)
	bMap := make(map[string]v1.EnvVar)

	for _, env := range a {
		aMap[env.Name] = env
	}

	for _, env := range b {
		bMap[env.Name] = env
	}

	// Check if all env vars in a exist in b with same values
	for name, envA := range aMap {
		envB, exists := bMap[name]
		if !exists {
			return false
		}

		if envA.Value != envB.Value {
			return false
		}

		if !reflect.DeepEqual(envA.ValueFrom, envB.ValueFrom) {
			return false
		}
	}

	return true
}

func (r *UnitReconciler) upgradePod(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit, pod *v1.Pod, upgradeReason string) error {

	r.Recorder.Eventf(unit, v1.EventTypeNormal, "ResourceCheck", "[%s] trigger regenerate pod: stop service -> delete pod -> regenerate pod", upgradeReason)

	// stop service
	tmpUint := unit.DeepCopy()
	tmpUint.Spec.Startup = false

	err := r.reconcileUnitServer(ctx, req, tmpUint)
	if err != nil && !apierrors.IsNotFound(err) {
		r.Recorder.Eventf(unit, v1.EventTypeWarning, "ErrResourceExists", "ignore: stop server fail [%s]", err.Error())
		// return err
	}

	// delete pod
	err = r.Delete(ctx, pod)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// wait for pod delete
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 40*time.Second, true, func(ctx context.Context) (bool, error) {
		pod := &v1.Pod{}
		err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, pod)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, fmt.Errorf("[upgradePod]wait pod deleted: get pod fail, error: [%s]", err.Error())
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("[upgradePod] error waiting for pod deleted: [%s]", err.Error())
	}

	// create
	pod, err = r.convert2Pod(ctx, unit)
	if err != nil {
		return fmt.Errorf("convert unit to pod error:[%s]", err.Error())
	}

	err = r.Create(ctx, pod)
	if err == nil {
		r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessCreated", "regenerate pod [%s] ok", pod.Name)
	}

	return err
}

func (r *UnitReconciler) convert2Pod(ctx context.Context, unit *upmiov1alpha2.Unit) (*v1.Pod, error) {
	unitGetErr := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}, unit)
	if unitGetErr != nil {
		return nil, fmt.Errorf("[upgradePod] get unit error:[%s]", unitGetErr.Error())
	}

	ref := metav1.NewControllerRef(unit, controllerKind)
	desiredLabels := getPodsLabelSet(unit)
	//desiredFinalizers := getPodsFinalizers(&unit.Spec.Template)
	desiredAnnotations := getPodsAnnotationSet(unit)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            unit.Name,
			Namespace:       unit.Namespace,
			Labels:          desiredLabels,
			Annotations:     desiredAnnotations,
			OwnerReferences: []metav1.OwnerReference{*ref},
		},
	}

	unit.Spec.Template.Spec.DeepCopyInto(&pod.Spec)

	return &pod, nil
}

func getPodsLabelSet(unit *upmiov1alpha2.Unit) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range unit.Spec.Template.Labels {
		desiredLabels[k] = v
	}

	for k, v := range unit.Labels {
		if _, ok := desiredLabels[k]; !ok {
			desiredLabels[k] = v
		}
	}

	return desiredLabels
}

// Commenting out unused function to fix lint errors
/*
func getPodsFinalizers(template *v1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}
*/

func getPodsAnnotationSet(unit *upmiov1alpha2.Unit) labels.Set {
	desiredAnnotations := make(labels.Set)
	for k, v := range unit.Spec.Template.Annotations {
		desiredAnnotations[k] = v
	}

	for k, v := range unit.Annotations {
		if _, ok := desiredAnnotations[k]; !ok {
			desiredAnnotations[k] = v
		}
	}

	return desiredAnnotations
}

// main container
func ifNeedUpgradePod(unit *upmiov1alpha2.Unit, pod *v1.Pod) (upgradeReason string, needUpgrade bool) {
	mainContainerName := unit.Annotations[upmiov1alpha2.AnnotationMainContainerName]

	for _, unitContainer := range unit.Spec.Template.Spec.Containers {
		for _, podContainer := range pod.Spec.Containers {
			if unitContainer.Name != podContainer.Name {
				continue
			}

			if unitContainer.Name == mainContainerName {
				// main container image
				if unitContainer.Image != podContainer.Image {
					return "image changed", true
				}

				// cpu,mem
				if unitContainer.Resources.Requests.Cpu().MilliValue() != podContainer.Resources.Requests.Cpu().MilliValue() ||
					unitContainer.Resources.Limits.Cpu().MilliValue() != podContainer.Resources.Limits.Cpu().MilliValue() {
					return "cpu changed", true
				}

				if unitContainer.Resources.Requests.Memory().Value() != podContainer.Resources.Requests.Memory().Value() ||
					unitContainer.Resources.Limits.Memory().Value() != podContainer.Resources.Limits.Memory().Value() {
					return "memory changed", true
				}

				// env - main container must restart to apply changes
				if !LoopCompareEnv(unitContainer.Env, podContainer.Env) {
					return "env changed", true
				}
			} else {
				// Any non-main container env drift requires recreation to keep pod in sync with Unit spec
				if !LoopCompareEnv(unitContainer.Env, podContainer.Env) {
					return fmt.Sprintf("container %s env changed", unitContainer.Name), true
				}
			}

			break
		}
	}

	// Check init containers for environment variable consistency
	for _, unitInitContainer := range unit.Spec.Template.Spec.InitContainers {
		for _, podInitContainer := range pod.Spec.InitContainers {
			if unitInitContainer.Name == podInitContainer.Name {
				if !LoopCompareEnv(unitInitContainer.Env, podInitContainer.Env) {
					// Init containers typically need restart for env changes
					return "init container env changed", true
				}
			}
		}
	}

	// status:
	// message: Pod Predicate NodeAffinity failed
	// phase: Failed
	// reason: NodeAffinity
	if pod.Spec.NodeName != "" && pod.Status.Reason == "NodeAffinity" && pod.Status.Phase == v1.PodFailed {
		klog.Warningf("%s find Pod Predicate NodeAffinity failed ,trigger recreate", pod.Name)
		return "Pod Predicate NodeAffinity failed", true
	}

	return "", false
}

// LoopCompareEnv compare two env slice
func LoopCompareEnv(unitEnvs, podEnvs []v1.EnvVar) bool {
	if (unitEnvs == nil) != (podEnvs == nil) {
		return false
	}

	if len(unitEnvs) == 0 && len(podEnvs) == 0 {
		return true
	}

	matched := true

	if len(unitEnvs) != len(podEnvs) {
		klog.Infof("[LoopCompareEnv] env count mismatch, unit len:%d, pod len:%d", len(unitEnvs), len(podEnvs))
		matched = false
	}

	podEnvMap := make(map[string]v1.EnvVar, len(podEnvs))
	for _, podEnv := range podEnvs {
		podEnvMap[podEnv.Name] = podEnv
	}

	for i := range unitEnvs {
		podEnv, exists := podEnvMap[unitEnvs[i].Name]
		if !exists {
			klog.Infof("[LoopCompareEnv] env name:%s exists in unit but not found in pod", unitEnvs[i].Name)
			matched = false
			continue
		}

		if unitEnvs[i].Value != "" && unitEnvs[i].Value != podEnv.Value {
			klog.Infof("[LoopCompareEnv] [value] env name:%s, unit value:%s, pod value:%s", unitEnvs[i].Name, unitEnvs[i].Value, podEnv.Value)
			matched = false
		} else if unitEnvs[i].ValueFrom != nil && !reflect.DeepEqual(unitEnvs[i].ValueFrom, podEnv.ValueFrom) {
			klog.Infof("[LoopCompareEnv] [valueFrom] env name:%s, unit valueFrom:%v, pod valueFrom:%v", unitEnvs[i].Name, unitEnvs[i].ValueFrom, podEnv.ValueFrom)
			matched = false
		} else if unitEnvs[i].Value == "" && unitEnvs[i].ValueFrom == nil {
			if podEnv.Value != "" {
				klog.Infof("[LoopCompareEnv] [empty value] env name:%s, unit value is empty but pod value:%s", unitEnvs[i].Name, podEnv.Value)
				matched = false
			} else if podEnv.ValueFrom != nil {
				klog.Infof("[LoopCompareEnv] [empty value] env name:%s, unit value is empty but pod has valueFrom:%v", unitEnvs[i].Name, podEnv.ValueFrom)
				matched = false
			}
		}

		delete(podEnvMap, unitEnvs[i].Name)
	}

	for name := range podEnvMap {
		klog.Infof("[LoopCompareEnv] env name:%s exists in pod but not found in unit", name)
		matched = false
	}

	return matched
}

func (r *UnitReconciler) waitUntilPodScheduled(ctx context.Context, podName, namespace string) (*v1.Pod, error) {
	// wait pod scheduled

	pod := &v1.Pod{}
	err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		podNamespacedName := client.ObjectKey{Name: podName, Namespace: namespace}
		err := r.Get(ctx, podNamespacedName, pod)
		if err != nil {
			return false, nil
		}

		if pod.Spec.NodeName == "" || !podutil.IsCreated(pod) {
			return false, nil
		}

		return true, nil

	})

	if err != nil {
		err = fmt.Errorf("waitUntilPodScheduled %s fail: %s", podName, err.Error())
	}

	return pod, err
}

func (r *UnitReconciler) podAutoRecovery(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {
	pod := &v1.Pod{}
	podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
	err := r.Get(ctx, podNamespacedName, pod)
	if err != nil && apierrors.IsNotFound(err) {
		//klog.Infof("[podAutoRecovery]:pod:[%s] not found, no need trrigger recovery", unit.Name)
		return nil
	}

	if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
		if unit.Spec.FailedPodRecoveryPolicy != nil {
			if !unit.Spec.FailedPodRecoveryPolicy.Enabled {
				klog.Infof("[podAutoRecovery]:pod:[%s] is failed, but recovery policy is disabled, no need trrigger recovery", unit.Name)
				return nil
			}
		}
	}

	if pod.Status.Phase == v1.PodFailed || pod.Status.Phase == v1.PodSucceeded {
		waitErr := r.waitPodFailed(ctx, unit)
		if waitErr == nil {
			return nil
		}

		klog.Infof("[podAutoRecovery]:pod:[%s], trrigger recreate", unit.Name)

		// delete pod
		err = r.Delete(ctx, pod)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		// wait delete
		err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 10*time.Second, true, func(ctx context.Context) (bool, error) {
			pod := &v1.Pod{}
			podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
			err := r.Get(ctx, podNamespacedName, pod)
			if err != nil && apierrors.IsNotFound(err) {
				return true, nil
			}

			return false, nil
		})

		if err != nil {
			return fmt.Errorf("[podAutoRecovery]: wait pod delete fail:%s", err.Error())
		}

		// create
		newPod, _ := r.convert2Pod(ctx, unit)

		err = r.Create(ctx, newPod)
		if err == nil {
			r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessCreated", "[podAutoRecovery]: recreate pod [%s] ok", pod.Name)
		}

		return err
	}

	return nil
}

func (r *UnitReconciler) waitPodFailed(ctx context.Context, unit *upmiov1alpha2.Unit) error {

	err := wait.PollUntilContextTimeout(ctx, 10*time.Second,
		time.Duration(10*int(unit.Spec.FailedPodRecoveryPolicy.ReconcileThreshold))*time.Second,
		true, func(ctx context.Context) (bool, error) {

			pod := &v1.Pod{}
			podNamespacedName := client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}
			err := r.Get(ctx, podNamespacedName, pod)
			if err != nil {
				return false, nil
			}

			if podutil.IsFailed(pod) || podutil.IsPodSucceeded(pod) {
				return false, nil
			}

			return true, nil

		})

	if err != nil {
		err = fmt.Errorf("waitPodFailed %s fail: %s", unit.Name, err.Error())
	}

	return err
}
