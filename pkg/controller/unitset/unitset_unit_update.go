package unitset

import (
	"context"
	"fmt"
	"sync"
	"time"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcileImageVersion(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	podTemplate *v1.PodTemplate,
	ports []v1.ContainerPort) error {

	// Two ways to change
	// 1. Update unitset directly
	// 2. Update podtemplate // Not considered for now

	// new version template
	templatePodTemplate := v1.PodTemplate{}
	templatePodTemplateNamespacedName := client.ObjectKey{Name: unitset.TemplatePodTemplateName(), Namespace: vars.ManagerNamespace}
	err := r.Get(ctx, templatePodTemplateNamespacedName, &templatePodTemplate)
	if err != nil {
		return err
	}

	needUpdate := false
	if !equality.Semantic.DeepEqual(podTemplate.Template, templatePodTemplate.Template) {
		podTemplate.Template = *templatePodTemplate.Template.DeepCopy()
		needUpdate = true
	}

	if needUpdate {
		volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitset)

		units, _ := unitset.UnitNames()
		errs := []error{}
		var wg sync.WaitGroup
		var errsMutex sync.Mutex

		for _, unit := range units {

			wg.Add(1)
			go func(unit string) {
				defer wg.Done()

				// get old
				kUnit := upmiov1alpha2.Unit{}
				err := r.Get(ctx, client.ObjectKey{Name: unit, Namespace: req.Namespace}, &kUnit)
				if err != nil {
					errsMutex.Lock()
					errs = append(errs, err)
					errsMutex.Unlock()
					return
				}

				// merge
				newUnit := mergePodTemplate(ctx, req, kUnit, unitset, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)

				// update
				err = r.Update(ctx, &newUnit)
				if err != nil {
					errsMutex.Lock()
					errs = append(errs, err)
					errsMutex.Unlock()
					return
				}

				time.Sleep(12 * time.Second)

				// wait for unit ready, and update annotation
				waitErr := wait.PollUntilContextTimeout(ctx, 10*time.Second, 90*time.Second, true, func(ctx context.Context) (bool, error) {

					newKUnit := &upmiov1alpha2.Unit{}
					err := r.Get(ctx, client.ObjectKey{Name: unit, Namespace: req.Namespace}, newKUnit)
					if err != nil {
						return false, nil
					}

					if newKUnit.Status.Phase != upmiov1alpha2.UnitReady {
						return false, nil
					}

					return true, nil

				})

				if waitErr != nil {
					errsMutex.Lock()
					errs = append(errs, fmt.Errorf("wait unit ready [%s] fail: %s", unit, waitErr.Error()))
					errsMutex.Unlock()
					return
				}

				// update annotation
				newUnit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version
				err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
					return r.Update(ctx, &newUnit)
				})

				if err != nil {
					errsMutex.Lock()
					errs = append(errs, fmt.Errorf("update unit annotation [%s] fail: %s", unit, err.Error()))
					errsMutex.Unlock()
					return
				}

			}(unit)

		}
		wg.Wait()

		err = utilerrors.NewAggregate(errs)
		if err != nil {
			return fmt.Errorf("reconcileImageVersion error:[%s]", err.Error())
		}

		oldVersionPodTemplate := v1.PodTemplate{}
		oldVersionPodTemplateNamespacedName := client.ObjectKey{Name: unitset.PodTemplateName(), Namespace: req.Namespace}
		err = r.Get(ctx, oldVersionPodTemplateNamespacedName, &oldVersionPodTemplate)
		if err != nil {
			return err
		}

		oldVersionPodTemplate.Template = *templatePodTemplate.Template.DeepCopy()

		err = r.Update(ctx, &oldVersionPodTemplate)
		if err != nil {
			return fmt.Errorf("[reconcileImageVersion] update podtemplate:[%s/%s] err:[%s]", req.Namespace, unitset.PodTemplateName(), err.Error())
		}
	}

	return nil
}

func mergePodTemplate(
	ctx context.Context,
	req ctrl.Request,
	kUnit upmiov1alpha2.Unit,
	unitset *upmiov1alpha2.UnitSet,
	podTemplate *v1.PodTemplate,
	ports []v1.ContainerPort,
	volumeMounts []v1.VolumeMount,
	volumes []v1.Volume,
	envVars []v1.EnvVar,
	pvcs []v1.PersistentVolumeClaim) upmiov1alpha2.Unit {

	unit := kUnit.DeepCopy()

	unit.Spec.Template = podTemplate.Template
	unit.Spec.Template.Spec.Subdomain = unitset.HeadlessServiceName()

	enableServiceLinks := true
	unit.Spec.Template.Spec.EnableServiceLinks = &enableServiceLinks

	unit.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("%s-serviceaccount", req.Namespace)

	unit.Spec.Template.Spec.Hostname = unit.Name

	fillVolumeMountsAndVolumes(unit, volumeMounts, volumes)
	fillEnvs(unit, unitset, envVars, ports)
	fillResourcesToDefaultContainer(unit, unitset)
	fillNodeAffinity(unit, unitset)
	fillPodAffinity(unit, unitset)
	//fillPortToDefaultContainer(unit, unitset, ports)

	// if NodeNameMap (from annotations) not empty, fill node name to unit.spec and unit.annotation
	nodeNameMap := getNodeNameMapFromAnnotations(unitset)
	if len(nodeNameMap) != 0 {
		nodeName, ok := nodeNameMap[unit.Name]
		if ok && nodeName != upmiov1alpha2.NoneSetFlag {
			//unit.Annotations[upmiov1alpha2.AnnotationLastUnitBelongNode] = nodeName

			if unit.Spec.Template.Spec.Affinity == nil {
				unit.Spec.Template.Spec.Affinity = &v1.Affinity{}
			}

			matchExpressions := v1.NodeSelectorRequirement{
				Key:      "kubernetes.io/hostname",
				Operator: v1.NodeSelectorOpIn,
				Values:   []string{nodeName},
			}

			// append matchExpressions
			if unit.Spec.Template.Spec.Affinity.NodeAffinity == nil {
				unit.Spec.Template.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{}
			}

			if unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
				unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								matchExpressions,
							},
						},
					},
				}
			} else {
				unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
					unit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
					v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{
							matchExpressions,
						},
					})
			}
		}
	}

	// emptyDir doesn't need pvc
	if len(unitset.Spec.Storages) != 0 {
		if len(unit.Spec.Template.Spec.Volumes) != 0 {
			for i := range unit.Spec.Template.Spec.Volumes {
				// Non-secret
				if unit.Spec.Template.Spec.Volumes[i].Name != "secret" {
					unit.Spec.Template.Spec.Volumes[i].PersistentVolumeClaim =
						&v1.PersistentVolumeClaimVolumeSource{
							ClaimName: upmiov1alpha2.PersistentVolumeClaimName(
								unit, unit.Spec.Template.Spec.Volumes[i].Name),
						}
				}
			}
		}
	}

	return *unit
}

// reconcile resources request
func (r *UnitSetReconciler) reconcileResources(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	//units, _ := unitset.UnitNames()

	kUnits, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileResources] error getting units: [%s]", err.Error())
	}

	if len(kUnits) == 0 {
		return nil
	}

	needUpdate := false
	for _, unit := range kUnits {
		// cpu memory
		for i := range unit.Spec.Template.Spec.Containers {
			if unit.Spec.Template.Spec.Containers[i].Name == unitset.Spec.Type {
				if unit.Spec.Template.Spec.Containers[i].Resources.Limits.Cpu().
					Cmp(*unitset.Spec.Resources.Limits.Cpu()) != 0 ||
					unit.Spec.Template.Spec.Containers[i].Resources.Limits.Memory().
						Cmp(*unitset.Spec.Resources.Limits.Memory()) != 0 ||
					unit.Spec.Template.Spec.Containers[i].Resources.Requests.Cpu().
						Cmp(*unitset.Spec.Resources.Requests.Cpu()) != 0 ||
					unit.Spec.Template.Spec.Containers[i].Resources.Requests.Memory().
						Cmp(*unitset.Spec.Resources.Requests.Memory()) != 0 {
					needUpdate = true
				}
			}
		}
	}

	errs := []error{}
	if needUpdate {
		var wg sync.WaitGroup
		for _, unit := range kUnits {
			wg.Add(1)
			go func(unit upmiov1alpha2.Unit) {
				defer wg.Done()

				err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					newUnit := mergeResources(unit, unitset)
					err = r.Update(ctx, &newUnit)
					if err != nil {
						//errs = append(errs, fmt.Errorf("[reconcileResources] update unit:[%s/%s] err:[%s]", req.Namespace, unit.Name, err.Error()))
						return err
					}

					return nil
				})

				//newUnit := mergeResources(unit, unitset)
				//err = r.Update(ctx, &newUnit)
				if err != nil {
					errs = append(errs, fmt.Errorf("[reconcileResources] update unit:[%s/%s] err:[%s]", req.Namespace, unit.Name, err.Error()))
					return
				}

			}(*unit)
		}
		wg.Wait()

		err = utilerrors.NewAggregate(errs)
		if err != nil {
			return fmt.Errorf("[reconcileResources] error:[%s]", err.Error())
		}
	}

	return nil
}

func mergeResources(kUnit upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) upmiov1alpha2.Unit {

	unit := kUnit.DeepCopy()

	for i := range unit.Spec.Template.Spec.Containers {
		if unit.Spec.Template.Spec.Containers[i].Name == unitset.Spec.Type {
			unit.Spec.Template.Spec.Containers[i].Resources = unitset.Spec.Resources
		}
	}

	return *unit
}

func (r *UnitSetReconciler) reconcileStorage(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	kUnits, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileStorage] error getting units: [%s]", err.Error())
	}

	if len(kUnits) == 0 {
		return nil
	}

	needUpdate := false

	// resource request
	for _, unit := range kUnits {
		for _, unitsetStorage := range unitset.Spec.Storages {
			for _, unitVolumeClaimTemplate := range unit.Spec.VolumeClaimTemplates {
				if unitVolumeClaimTemplate.Name == unitsetStorage.Name {

					if unitVolumeClaimTemplate.Spec.Resources.Requests.Storage().
						Cmp(resource.MustParse(unitsetStorage.Size)) < 0 {
						needUpdate = true
					}
				}
			}
		}
	}

	errs := []error{}
	if needUpdate {
		var wg sync.WaitGroup
		for _, unit := range kUnits {
			wg.Add(1)
			go func(unit upmiov1alpha2.Unit) {
				defer wg.Done()

				err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					newUnit := mergeStorage(unit, unitset)
					err = r.Update(ctx, &newUnit)
					if err != nil {
						return err
					}

					return nil
				})

				if err != nil {
					errs = append(errs, fmt.Errorf("[reconcileStorage] update unit:[%s/%s] err:[%s]", req.Namespace, unit.Name, err.Error()))
					return
				}

			}(*unit)
		}
		wg.Wait()

		err = utilerrors.NewAggregate(errs)
		if err != nil {
			return fmt.Errorf("[reconcileStorage] error:[%s]", err.Error())
		}
	}

	return nil
}

func mergeStorage(kUnit upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) upmiov1alpha2.Unit {

	unit := kUnit.DeepCopy()

	for i := range unit.Spec.VolumeClaimTemplates {
		for _, unitsetStorage := range unitset.Spec.Storages {
			if unit.Spec.VolumeClaimTemplates[i].Name == unitsetStorage.Name {
				unit.Spec.VolumeClaimTemplates[i].Spec.Resources.Requests["storage"] = resource.MustParse(unitsetStorage.Size)
			}
		}
	}

	return *unit
}

// reconcileUnitLabelsAnnotations ensures that UnitSet metadata (labels/annotations)
// are propagated to all Units it manages. It merges keys from UnitSet into each Unit
// without removing pre-existing Unit-specific keys.
func (r *UnitSetReconciler) reconcileUnitLabelsAnnotations(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
) error {
	kUnits, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileUnitLabelsAnnotations] error getting units: [%s]", err.Error())
	}

	if len(kUnits) == 0 {
		return nil
	}

	errs := []error{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, unit := range kUnits {
		wg.Add(1)
		go func(unit upmiov1alpha2.Unit) {
			defer wg.Done()

			updateErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// Get latest
				latest := &upmiov1alpha2.Unit{}
				if err := r.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: unit.Namespace}, latest); err != nil {
					return err
				}

				needUpdate := false

				if latest.Labels == nil {
					latest.Labels = map[string]string{}
					needUpdate = true
				}
				if latest.Annotations == nil {
					latest.Annotations = map[string]string{}
					needUpdate = true
				}

				// Always ensure Unit is labeled with UnitSet name
				if latest.Labels[upmiov1alpha2.UnitsetName] != unitset.Name {
					latest.Labels[upmiov1alpha2.UnitsetName] = unitset.Name
					needUpdate = true
				}

				// Merge labels from UnitSet
				for k, v := range unitset.Labels {
					if cur, ok := latest.Labels[k]; !ok || cur != v {
						latest.Labels[k] = v
						needUpdate = true
					}
				}

				// Merge annotations from UnitSet
				for k, v := range unitset.Annotations {
					if cur, ok := latest.Annotations[k]; !ok || cur != v {
						latest.Annotations[k] = v
						needUpdate = true
					}
				}

				if !needUpdate {
					return nil
				}

				return r.Update(ctx, latest)
			})

			if updateErr != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("[reconcileUnitLabelsAnnotations] update unit [%s/%s] err: [%s]", unit.Namespace, unit.Name, updateErr.Error()))
				mu.Unlock()
			}
		}(*unit)
	}

	wg.Wait()

	if agg := utilerrors.NewAggregate(errs); agg != nil {
		return fmt.Errorf("[reconcileUnitLabelsAnnotations] error: [%s]", agg.Error())
	}

	return nil
}
