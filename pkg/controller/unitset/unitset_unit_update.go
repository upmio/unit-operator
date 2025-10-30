package unitset

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

const updateStrategyRollingUpdate = "RollingUpdate"

// These timings are variables so tests can adjust them to avoid lengthy sleeps.
var (
	unitUpdateGracePeriod = 12 * time.Second
	unitReadyPollInterval = 10 * time.Second
	unitReadyPollTimeout  = 90 * time.Second
)

var errUnitFailedState = errors.New("unit entered failed state")

func sortUnitNamesByOrdinal(units []string) []string {
	sorted := make([]string, len(units))
	copy(sorted, units)
	sort.SliceStable(sorted, func(i, j int) bool {
		iOrdinal, iHasOrdinal := extractUnitOrdinal(sorted[i])
		jOrdinal, jHasOrdinal := extractUnitOrdinal(sorted[j])

		switch {
		case iHasOrdinal && jHasOrdinal:
			if iOrdinal != jOrdinal {
				return iOrdinal > jOrdinal
			}
			return sorted[i] > sorted[j]
		case iHasOrdinal:
			return true
		case jHasOrdinal:
			return false
		default:
			return sorted[i] > sorted[j]
		}
	})
	return sorted
}

func extractUnitOrdinal(unitName string) (int, bool) {
	idx := strings.LastIndex(unitName, "-")
	if idx == -1 || idx == len(unitName)-1 {
		return 0, false
	}

	ordinal, err := strconv.Atoi(unitName[idx+1:])
	if err != nil {
		return 0, false
	}

	return ordinal, true
}

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
		return fmt.Errorf("failed to get template pod template [%s/%s]: %w", vars.ManagerNamespace, unitset.TemplatePodTemplateName(), err)
	}

	needUpdate := false
	if !equality.Semantic.DeepEqual(podTemplate.Template, templatePodTemplate.Template) {
		podTemplate.Template = *templatePodTemplate.Template.DeepCopy()
		needUpdate = true
	}

	if needUpdate {

		volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitset)

		units, _ := unitset.UnitNames()
		if len(units) == 0 {
			return nil
		}

		units = sortUnitNamesByOrdinal(units)

		updateUnit := func(unit string) error {
			return r.updateUnitImageVersion(ctx, req, unitset, podTemplate, ports, volumeMounts, volumes, envVars, pvcs, unit)
		}

		var (
			updateErr    error
			upgradeReady bool
		)
		if strings.EqualFold(unitset.Spec.UpdateStrategy.Type, updateStrategyRollingUpdate) {
			upgradeReady, updateErr = r.performRollingUpdate(ctx, req, unitset, units, updateUnit)
		} else {
			updateErr = r.performParallelUpdate(ctx, units, updateUnit)
			upgradeReady = updateErr == nil
		}

		if updateErr != nil {
			return fmt.Errorf("reconcileImageVersion failed: %w", updateErr)
		}

		if !upgradeReady {
			return nil
		}

		// Update the old version pod template only after all units are successfully updated
		oldVersionPodTemplate := v1.PodTemplate{}
		oldVersionPodTemplateNamespacedName := client.ObjectKey{Name: unitset.PodTemplateName(), Namespace: req.Namespace}
		err = r.Get(ctx, oldVersionPodTemplateNamespacedName, &oldVersionPodTemplate)
		if err != nil {
			return fmt.Errorf("failed to get old version pod template [%s/%s]: %w", req.Namespace, unitset.PodTemplateName(), err)
		}

		oldVersionPodTemplate.Template = *templatePodTemplate.Template.DeepCopy()

		err = r.Update(ctx, &oldVersionPodTemplate)
		if err != nil {
			return fmt.Errorf("failed to update pod template [%s/%s]: %w", req.Namespace, unitset.PodTemplateName(), err)
		}

		err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset)
		if err != nil {
			return fmt.Errorf("[reconcileImageVersion] update unitset status error:[%s]", err.Error())
		}
	}

	return nil
}

// performRollingUpdate handles rolling update with state tracking
func (r *UnitSetReconciler) performRollingUpdate(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	units []string,
	updateUnit func(string) error) (bool, error) {

	units = sortUnitNamesByOrdinal(units)

	// Check current upgrade state
	currentUpgradeUnit := unitset.Status.InUpdate

	// If no upgrade in progress, start with the first unit
	if currentUpgradeUnit == "" {
		if len(units) == 0 {
			return true, nil
		}
		currentUpgradeUnit = units[0]

		// Update UnitSet status to mark upgrade start
		if err := r.updateInUpdateStatus(ctx, req, unitset, currentUpgradeUnit); err != nil {
			return false, fmt.Errorf("failed to start upgrade for unit [%s]: %w", currentUpgradeUnit, err)
		}
	}

	// Find the current unit in the sorted list
	currentIndex := -1
	for i, unit := range units {
		if unit == currentUpgradeUnit {
			currentIndex = i
			break
		}
	}

	// If current unit is not in the list, something is wrong
	if currentIndex == -1 {
		// Reset and start from beginning
		currentUpgradeUnit = units[0]
		currentIndex = 0
		if err := r.updateInUpdateStatus(ctx, req, unitset, currentUpgradeUnit); err != nil {
			return false, fmt.Errorf("failed to reset upgrade for unit [%s]: %w", currentUpgradeUnit, err)
		}
	}

	// Check if current unit needs upgrade
	needsUpgrade, err := r.checkUnitNeedsUpgrade(ctx, req, unitset, currentUpgradeUnit)
	if err != nil {
		return false, fmt.Errorf("failed to check if unit [%s] needs upgrade: %w", currentUpgradeUnit, err)
	}

	if needsUpgrade {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("rolling update cancelled while upgrading unit [%s]: %w", currentUpgradeUnit, ctx.Err())
		default:
		}

		// Upgrade current unit
		if err := updateUnit(currentUpgradeUnit); err != nil {
			return false, fmt.Errorf("rolling update failed for unit [%s]: %w", currentUpgradeUnit, err)
		}
	}

	// Move to next unit
	nextIndex := currentIndex + 1
	if nextIndex < len(units) {
		nextUnit := units[nextIndex]
		if err := r.updateInUpdateStatus(ctx, req, unitset, nextUnit); err != nil {
			return false, fmt.Errorf("failed to advance to next unit [%s]: %w", nextUnit, err)
		}

		// Requeue to continue with next unit via status update event
		return false, nil
	}

	// All units processed, clear InUpdate status
	if err := r.updateInUpdateStatus(ctx, req, unitset, ""); err != nil {
		return false, fmt.Errorf("failed to clear upgrade status: %w", err)
	}

	return true, nil
}

// updateInUpdateStatus updates the InUpdate field in UnitSet status
func (r *UnitSetReconciler) updateInUpdateStatus(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	unitName string) error {

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// Get latest UnitSet
		latest := &upmiov1alpha2.UnitSet{}
		if err := r.Get(ctx, client.ObjectKey{Name: unitset.Name, Namespace: req.Namespace}, latest); err != nil {
			return fmt.Errorf("failed to get latest unitset: %w", err)
		}

		// Update InUpdate status
		latest.Status.InUpdate = unitName

		// Update status
		if err := r.Status().Update(ctx, latest); err != nil {
			return fmt.Errorf("failed to update InUpdate status: %w", err)
		}

		return nil
	})
}

// checkUnitNeedsUpgrade checks if a unit needs to be upgraded
func (r *UnitSetReconciler) checkUnitNeedsUpgrade(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	unitName string) (bool, error) {

	// Get the unit
	unit := &upmiov1alpha2.Unit{}
	if err := r.Get(ctx, client.ObjectKey{Name: unitName, Namespace: req.Namespace}, unit); err != nil {
		return false, fmt.Errorf("failed to get unit [%s]: %w", unitName, err)
	}

	// Check if unit version annotation matches unitset version
	if unit.Annotations == nil {
		return true, nil
	}

	currentVersion, exists := unit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]
	if !exists || currentVersion != unitset.Spec.Version {
		return true, nil
	}

	// Check if unit is ready
	//if unit.Status.Phase != upmiov1alpha2.UnitReady {
	//	return true, nil
	//}

	return false, nil
}

//// performRollingUpdate handles rolling update strategy (one by one)
//func (r *UnitSetReconciler) performRollingUpdate(ctx context.Context, units []string, updateUnit func(string) error) error {
//	for i, unit := range units {
//		// Check if context is cancelled before processing each unit
//		select {
//		case <-ctx.Done():
//			return fmt.Errorf("rolling update cancelled at unit %d/%d [%s]: %w", i+1, len(units), unit, ctx.Err())
//		default:
//		}
//
//		if err := updateUnit(unit); err != nil {
//			return fmt.Errorf("rolling update failed at unit %d/%d [%s]: %w", i+1, len(units), unit, err)
//		}
//	}
//	return nil
//}

// performParallelUpdate handles parallel update strategy (all at once)
func (r *UnitSetReconciler) performParallelUpdate(ctx context.Context, units []string, updateUnit func(string) error) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(units))

	for _, unit := range units {
		wg.Add(1)
		go func(unit string) {
			defer wg.Done()

			if err := updateUnit(unit); err != nil {
				errChan <- fmt.Errorf("parallel update failed for unit [%s]: %w", unit, err)
			}
		}(unit)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect all errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return utilerrors.NewAggregate(errs)
	}

	return nil
}

func (r *UnitSetReconciler) updateUnitImageVersion(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	podTemplate *v1.PodTemplate,
	ports []v1.ContainerPort,
	volumeMounts []v1.VolumeMount,
	volumes []v1.Volume,
	envVars []v1.EnvVar,
	pvcs []v1.PersistentVolumeClaim,
	unit string) error {

	// get old unit
	original := upmiov1alpha2.Unit{}
	if err := r.Get(ctx, client.ObjectKey{Name: unit, Namespace: req.Namespace}, &original); err != nil {
		return fmt.Errorf("failed to get unit [%s/%s]: %w", req.Namespace, unit, err)
	}

	// merge desired template and update unit spec
	updated := mergePodTemplate(ctx, req, original, unitset, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)

	// Use retry mechanism for unit update to handle conflicts
	updateErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return r.Update(ctx, &updated)
	})
	if updateErr != nil {
		return fmt.Errorf("failed to update unit [%s/%s]: %w", req.Namespace, unit, updateErr)
	}

	// Grace period for unit to start updating
	if unitUpdateGracePeriod > 0 {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during grace period for unit [%s]: %w", unit, ctx.Err())
		case <-time.After(unitUpdateGracePeriod):
		}
	}

	// Update annotation to mark the version
	updateAnnotationErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		current := &upmiov1alpha2.Unit{}
		if err := r.Get(ctx, client.ObjectKey{Name: unit, Namespace: req.Namespace}, current); err != nil {
			return fmt.Errorf("failed to get unit for annotation update: %w", err)
		}

		if current.Annotations == nil {
			current.Annotations = map[string]string{}
		}
		current.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version
		return r.Update(ctx, current)
	})

	if updateAnnotationErr != nil {
		return fmt.Errorf("failed to update version annotation for unit [%s]: %w", unit, updateAnnotationErr)
	}

	return nil
}

func (r *UnitSetReconciler) waitForUnitReady(ctx context.Context, req ctrl.Request, unitName string) error {
	waitErr := wait.PollUntilContextTimeout(ctx, unitReadyPollInterval, unitReadyPollTimeout, true, func(ctx context.Context) (bool, error) {
		current := &upmiov1alpha2.Unit{}
		if err := r.Get(ctx, client.ObjectKey{Name: unitName, Namespace: req.Namespace}, current); err != nil {
			return false, nil
		}

		switch current.Status.Phase {
		case upmiov1alpha2.UnitReady:
			return true, nil
		case upmiov1alpha2.UnitFailed:
			return false, errUnitFailedState
		default:
			return false, nil
		}
	})

	if waitErr != nil {
		current := &upmiov1alpha2.Unit{}
		currentPhase := "unknown"
		if err := r.Get(ctx, client.ObjectKey{Name: unitName, Namespace: req.Namespace}, current); err == nil {
			currentPhase = string(current.Status.Phase)
		}

		if errors.Is(waitErr, errUnitFailedState) {
			return fmt.Errorf("unit [%s] entered failed state (phase: %s): %w", unitName, currentPhase, waitErr)
		}

		return fmt.Errorf("unit [%s] did not become ready within timeout %v (current phase: %s): %w", unitName, unitReadyPollTimeout, currentPhase, waitErr)
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

	// only storage type volume needs pvc
	if len(unitset.Spec.Storages) != 0 && len(unit.Spec.Template.Spec.Volumes) != 0 {
		for i := range unit.Spec.Template.Spec.Volumes {
			for j := range unitset.Spec.Storages {
				if unit.Spec.Template.Spec.Volumes[i].Name == unitset.Spec.Storages[j].Name {
					unit.Spec.Template.Spec.Volumes[i].PersistentVolumeClaim =
						&v1.PersistentVolumeClaimVolumeSource{
							ClaimName: upmiov1alpha2.PersistentVolumeClaimName(
								unit, unit.Spec.Template.Spec.Volumes[i].Name),
						}
				}
			}
		}
	}

	if len(unit.Spec.Template.Spec.Volumes) != 0 {
		for i := range unit.Spec.Template.Spec.Volumes {
			if unit.Spec.Template.Spec.Volumes[i].Name == "certificate" {
				certificateSecretName := fmt.Sprintf(
					"%s-%s-%s",
					unit.Name,
					upmiov1alpha2.CertmanagerCertificateSuffix,
					upmiov1alpha2.CertmanagerSecretNameSuffix)

				unit.Spec.Template.Spec.Volumes[i].Secret = &v1.SecretVolumeSource{
					SecretName: certificateSecretName,
				}
			}
		}
	}

	return *unit
}

// reconcile resources request
func (r *UnitSetReconciler) reconcileResources(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	kUnits, err := r.unitsBelongUnitset(ctx, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileResources] error getting units: [%s]", err.Error())
	}

	if len(kUnits) == 0 {
		return nil
	}

	unitMap := make(map[string]*upmiov1alpha2.Unit, len(kUnits))
	for _, unit := range kUnits {
		unitMap[unit.Name] = unit
	}

	unitNames, _ := unitset.UnitNames()
	unitNames = sortUnitNamesByOrdinal(unitNames)

	if strings.EqualFold(unitset.Spec.UpdateStrategy.Type, updateStrategyRollingUpdate) {
		current := unitset.Status.InUpdate
		if current == "" {
			for _, unitName := range unitNames {
				unit, exists := unitMap[unitName]
				if !exists {
					continue
				}
				if needsResourceUpdate(unit, unitset) {
					if err := r.updateInUpdateStatus(ctx, req, unitset, unitName); err != nil {
						return fmt.Errorf("failed to start resource update for unit [%s]: %w", unitName, err)
					}
					return nil
				}
			}
			if err := r.updateInUpdateStatus(ctx, req, unitset, ""); err != nil {
				return fmt.Errorf("failed to clear resource update status: %w", err)
			}
			return nil
		}

		unit, exists := unitMap[current]
		if !exists || !needsResourceUpdate(unit, unitset) {
			next := ""
			for i, name := range unitNames {
				if name == current && i+1 < len(unitNames) {
					for j := i + 1; j < len(unitNames); j++ {
						nextUnit := unitMap[unitNames[j]]
						if needsResourceUpdate(nextUnit, unitset) {
							next = unitNames[j]
							break
						}
					}
					break
				}
			}
			if err := r.updateInUpdateStatus(ctx, req, unitset, next); err != nil {
				return fmt.Errorf("failed to advance resource update status: %w", err)
			}
			return nil
		}

		if unit.Status.Phase != upmiov1alpha2.UnitReady {
			return nil
		}

		if err := r.updateUnitResources(ctx, req, unitset, current); err != nil {
			return err
		}

		allReady := true
		for _, unitName := range unitNames {
			unit, exists := unitMap[unitName]
			if !exists || needsResourceUpdate(unit, unitset) || unit.Status.Phase != upmiov1alpha2.UnitReady {
				allReady = false
				break
			}
		}
		if allReady {
			err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset)
			if err != nil {
				return fmt.Errorf("[reconcileResources] update unitset status error:[%s]", err.Error())
			}
		}
		return nil
	}

	var (
		wg   sync.WaitGroup
		errs []error
		mu   sync.Mutex
	)

	ifNeedUpdateObservedGeneration := false

	for _, unitName := range unitNames {
		unit, exists := unitMap[unitName]
		if !exists {
			continue
		}

		if !needsResourceUpdate(unit, unitset) {
			continue
		}

		ifNeedUpdateObservedGeneration = true

		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			if err := r.updateUnitResources(ctx, req, unitset, name); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(unitName)
	}

	wg.Wait()
	if len(errs) > 0 {
		return fmt.Errorf("[reconcileResources] error:[%s]", utilerrors.NewAggregate(errs))
	}

	if ifNeedUpdateObservedGeneration {
		// update observedGeneration of unitset status
		err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset)
		if err != nil {
			return fmt.Errorf("[reconcileResources] update unitset status error:[%s]", err.Error())
		}
	}

	return nil
}

func (r *UnitSetReconciler) updateResourceInUpdateStatus(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	unitName string) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latest := &upmiov1alpha2.UnitSet{}
		if err := r.Get(ctx, client.ObjectKey{Name: unitset.Name, Namespace: req.Namespace}, latest); err != nil {
			return fmt.Errorf("failed to get latest unitset: %w", err)
		}
		latest.Status.InUpdate = unitName
		if err := r.Status().Update(ctx, latest); err != nil {
			return fmt.Errorf("failed to update ResourceInUpdate status: %w", err)
		}
		return nil
	})
}

func (r *UnitSetReconciler) updateUnitResources(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	unitName string) error {
	updated := false
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		current := &upmiov1alpha2.Unit{}
		if err := r.Get(ctx, client.ObjectKey{Name: unitName, Namespace: req.Namespace}, current); err != nil {
			return err
		}
		if !needsResourceUpdate(current, unitset) {
			return nil
		}
		updatedUnit := mergeResources(*current, unitset)
		updated = true
		return r.Update(ctx, &updatedUnit)
	})
	if err != nil {
		return fmt.Errorf("[reconcileResources] update unit:[%s/%s] err:[%w]", req.Namespace, unitName, err)
	}
	if !updated {
		return nil
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
	ifNeedUpdateObservedGeneration := false

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
		ifNeedUpdateObservedGeneration = true

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

	if ifNeedUpdateObservedGeneration {
		if err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset); err != nil {
			return fmt.Errorf("[reconcileStorage] error: [%s]", err.Error())
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

	ifNeedUpdateObservedGeneration := false

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

				if latest.Labels[upmiov1alpha2.LabelUnitsCount] != strconv.Itoa(unitset.Spec.Units) {
					latest.Labels[upmiov1alpha2.LabelUnitsCount] = strconv.Itoa(unitset.Spec.Units)
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

				ifNeedUpdateObservedGeneration = true

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

	if ifNeedUpdateObservedGeneration {
		err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset)
		if err != nil {
			return fmt.Errorf("[reconcileUnitLabelsAnnotations] error: [%s]", err.Error())
		}
	}

	return nil
}

func needsResourceUpdate(unit *upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) bool {
	for i := range unit.Spec.Template.Spec.Containers {
		if unit.Spec.Template.Spec.Containers[i].Name != unitset.Spec.Type {
			continue
		}
		container := unit.Spec.Template.Spec.Containers[i]
		desired := unitset.Spec.Resources

		if container.Resources.Limits.Cpu().Cmp(*desired.Limits.Cpu()) != 0 {
			return true
		}
		if container.Resources.Limits.Memory().Cmp(*desired.Limits.Memory()) != 0 {
			return true
		}
		if container.Resources.Requests.Cpu().Cmp(*desired.Requests.Cpu()) != 0 {
			return true
		}
		if container.Resources.Requests.Memory().Cmp(*desired.Requests.Memory()) != 0 {
			return true
		}
	}
	return false
}
