package unit

import (
	"context"
	"encoding/json"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	podutil "github.com/upmio/unit-operator/pkg/utils/pod"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func (r *UnitReconciler) reconcilePod(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit) error {

	pod := &v1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: req.Namespace}, pod)
	if apierrors.IsNotFound(err) {
		// if not found, generate from template
		pod, _ = convert2Pod(unit)

		err = r.Create(ctx, pod)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
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

			return r.Status().Update(ctx, orig)

		})
		if err != nil {
			return fmt.Errorf("[reconcilePod] update unit status fail before [upgradePod], error: [%s]", err.Error())
		}

		err = r.upgradePod(ctx, req, unit, pod, reason)
		if err != nil {
			return err
		}
	}

	// sync label, not image here
	patch, need, err := ifNeedPatchPod(unit, pod)

	if need {
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

	return nil
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
			if unit.Spec.Template.Spec.Containers[i].Name != unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] &&
				unit.Spec.Template.Spec.Containers[i].Name == clone.Spec.Containers[j].Name &&
				clone.Spec.Containers[j].Image != unit.Spec.Template.Spec.Containers[i].Image {
				clone.Spec.Containers[j].Image = unit.Spec.Template.Spec.Containers[i].Image
			}
		}
	}

	return clone
}

func (r *UnitReconciler) upgradePod(ctx context.Context, req ctrl.Request, unit *upmiov1alpha2.Unit, pod *v1.Pod, upgradeReason string) error {

	r.Recorder.Eventf(unit, v1.EventTypeNormal, "ResourceCheck", "[%s] trigger regenerate pod: stop service -> delete pod -> regenerate pod", upgradeReason)

	// stop service
	tmpuint := unit.DeepCopy()
	tmpuint.Spec.Startup = false

	err := r.reconcileUnitServer(ctx, tmpuint)
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
	pod, err = convert2Pod(unit)
	if err != nil {
		return fmt.Errorf("convert unit to pod error:[%s]", err.Error())
	}

	err = r.Create(ctx, pod)
	if err == nil {
		r.Recorder.Eventf(unit, v1.EventTypeNormal, "SuccessCreated", "regenerate pod [%s] ok", pod.Name)
	}

	return err
}

func convert2Pod(unit *upmiov1alpha2.Unit) (*v1.Pod, error) {
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

	pod.Spec = *unit.Spec.Template.Spec.DeepCopy()

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

func getPodsFinalizers(template *v1.PodTemplateSpec) []string {
	desiredFinalizers := make([]string, len(template.Finalizers))
	copy(desiredFinalizers, template.Finalizers)
	return desiredFinalizers
}

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
	for _, unitContainer := range unit.Spec.Template.Spec.Containers {
		for _, podContainer := range pod.Spec.Containers {

			if unitContainer.Name == podContainer.Name && podContainer.Name == unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] {
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

				// env
				if !LoopCompareEnv(unitContainer.Env, podContainer.Env) {
					return "env changed", true
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

	// Compare only the env's in the unit.
	// i：If it exists in unit but not in pod, return false
	// ii：If it exists in unit and it exists in pod, but the value is not the same, then it returns false

	findEnv := false

	for i := range unitEnvs {
		for j := range podEnvs {
			if unitEnvs[i].Name == podEnvs[j].Name {

				findEnv = true

				if unitEnvs[i].Value != "" && unitEnvs[i].Value != podEnvs[j].Value {
					klog.Infof("[LoopCompareEnv] [value] env name:%s, unit value:%s, pod value:%s", unitEnvs[i].Name, unitEnvs[i].Value, podEnvs[j].Value)
					return false
				} else if unitEnvs[i].ValueFrom != nil && !reflect.DeepEqual(unitEnvs[i].ValueFrom, podEnvs[j].ValueFrom) {
					klog.Infof("[LoopCompareEnv] [valueFrom] env name:%s, unit valueFrom:%v, pod valueFrom:%v", unitEnvs[i].Name, unitEnvs[i].ValueFrom, podEnvs[j].ValueFrom)
					return false
				}

			}
		}
	}

	if !findEnv {
		return false
	}

	return true
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
