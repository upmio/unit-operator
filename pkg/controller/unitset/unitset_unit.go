package unitset

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcileUnit(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	podTemplate *v1.PodTemplate,
	ports []v1.ContainerPort) error {

	unitNames, unitNamesWithIndex := unitset.UnitNames()
	klog.Infof("reconcileUnit units len:[%d],[%v]", len(unitNames), unitNames)

	// volume's PVC name needs to be filled in during unit generation
	volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitset)

	// Create missing units with bounded concurrency + per-unit timeout to avoid goroutine leak.
	var needUpdateObservedGeneration atomic.Bool
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, unitName := range unitNames {
		unitName := unitName
		g.Go(func() error {
			// bound each unit operation
			uctx, cancel := context.WithTimeout(gctx, 5*time.Second)
			defer cancel()

			kUnit := upmiov1alpha2.Unit{}
			err := r.Get(uctx, client.ObjectKey{Name: unitName, Namespace: req.Namespace}, &kUnit)
			if apierrors.IsNotFound(err) {
				unitTemplate, err := r.generateUnitTemplate(uctx, req, unitName, unitset, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)
				if err != nil {
					return fmt.Errorf("[reconcileUnit] generateUnitTemplate: unitName:[%s] err:[%v]", unitName, err)
				}

				unit := fillUnitPersonalizedInfo(unitTemplate, unitset, unitNamesWithIndex, unitName)
				needUpdateObservedGeneration.Store(true)

				err = r.Create(uctx, unit)
				if err != nil && !apierrors.IsAlreadyExists(err) {
					return err
				}
				return nil
			}
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcileUnit error:[%s]", err.Error())
	}

	// remove unit
	kUnits, listErr := r.unitsBelongUnitset(ctx, unitset)
	if listErr != nil {
		return fmt.Errorf("[reconcileUnit] list units err:[%v]", listErr)
	}

	if len(kUnits) != 0 && len(kUnits) > unitset.Spec.Units {
		// remove units
		_, rmErr := r.removeUnits(ctx, unitset, kUnits)
		if rmErr != nil {
			return rmErr
		}
		needUpdateObservedGeneration.Store(true)
	}

	if needUpdateObservedGeneration.Load() {
		// update observedGeneration of unitset status
		err := r.reconcileUnitsetObservedGeneration(ctx, req, unitset)
		if err != nil {
			return fmt.Errorf("[reconcileUnit] update unitset status error:[%s]", err.Error())
		}
	}

	return nil
}

func (r *UnitSetReconciler) removeUnits(ctx context.Context, unitset *upmiov1alpha2.UnitSet, kUnits []*upmiov1alpha2.Unit) ([]*upmiov1alpha2.Unit, error) {
	expectedCount := unitset.Spec.Units

	if len(kUnits) == expectedCount {
		return kUnits, nil
	}

	out := []*upmiov1alpha2.Unit{}
	for _, one := range kUnits {
		serialNumber, err := strconv.Atoi(one.Labels[upmiov1alpha2.UnitSn])
		if err != nil {
			return nil, fmt.Errorf("[removeUnits] get unit:[%s] serial number error:[%s]", one.Name, err.Error())
		}

		if serialNumber+1 <= expectedCount {
			out = append(out, one)
			continue
		}

		err = r.Delete(ctx, one)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return out, nil
			}

			return nil, fmt.Errorf("[removeUnits] delete unit:[%s] error:[%s]", one.Name, err.Error())
		}
	}

	return out, nil
}

func fillUnitPersonalizedInfo(
	unitTemplate upmiov1alpha2.Unit,
	unitset *upmiov1alpha2.UnitSet,
	unitNamesWithIndex map[string]string,
	unitName string) *upmiov1alpha2.Unit {

	unit := unitTemplate.DeepCopy()

	//unit.Name = unitName
	unit.Spec.Template.Spec.Hostname = unitName

	if unit.Labels == nil {
		unit.Labels = make(map[string]string)
	}

	for k, v := range unitset.Labels {
		unit.Labels[k] = v
	}

	unit.Labels[upmiov1alpha2.UnitSn] = unitNamesWithIndex[unitName]
	unit.Labels[upmiov1alpha2.UnitName] = unitName

	if unit.Annotations == nil {
		unit.Annotations = make(map[string]string)
	}

	for k, v := range unitset.Annotations {
		unit.Annotations[k] = v
	}

	unit.Spec.ConfigValueName = unitset.ConfigValueName(unitName)

	// if NodeNameMap (from annotations) not empty, fill node name to node affinity
	nodeNameMap := getNodeNameMapFromAnnotations(unitset)
	if len(nodeNameMap) != 0 {
		nodeName, ok := nodeNameMap[unitName]
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
	if len(unitset.Spec.Storage) != 0 && len(unit.Spec.Template.Spec.Volumes) != 0 {
		for i := range unit.Spec.Template.Spec.Volumes {
			for j := range unitset.Spec.Storage {
				if unit.Spec.Template.Spec.Volumes[i].Name == unitset.Spec.Storage[j].Name {
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

	return unit
}

func (r *UnitSetReconciler) generateUnitTemplate(
	ctx context.Context,
	req ctrl.Request,
	unitName string,
	unitset *upmiov1alpha2.UnitSet,
	podTemplate *v1.PodTemplate,
	ports []v1.ContainerPort,
	volumeMounts []v1.VolumeMount,
	volumes []v1.Volume,
	envVars []v1.EnvVar,
	pvcs []upmiov1alpha2.UnitVolumeClaimTemplate) (upmiov1alpha2.Unit, error) {

	if unitset == nil {
		return upmiov1alpha2.Unit{}, fmt.Errorf("[generateUnitTemplate] unitset is nil")
	}

	if podTemplate == nil {
		return upmiov1alpha2.Unit{}, fmt.Errorf("[generateUnitTemplate] podTemplate is nil")
	}

	// no name, ConfigValueName
	ref := metav1.NewControllerRef(unitset, controllerKind)

	failedPodRecoveryPolicy := upmiov1alpha2.FailedPodRecoveryPolicy{
		Enabled:            true,
		ReconcileThreshold: 6,
	}

	unit := upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:            unitName,
			Namespace:       req.Namespace,
			Labels:          make(map[string]string),
			Annotations:     make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{*ref},
			Finalizers: []string{
				upmiov1alpha2.FinalizerPodDelete,
				upmiov1alpha2.FinalizerPvcDelete,
			},
		},
		Spec: upmiov1alpha2.UnitSpec{
			Startup:                 true,
			ConfigTemplateName:      unitset.ConfigTemplateName(),
			Template:                upmiov1alpha2.UnitPodTemplateSpec{},
			FailedPodRecoveryPolicy: &failedPodRecoveryPolicy,
		},
	}

	if len(pvcs) != 0 {
		for _, pvc := range pvcs {

			klog.Infof("[generateUnitTemplate] append PVC info [%s]", pvc.Name)
			unit.Spec.VolumeClaimTemplates = append(unit.Spec.VolumeClaimTemplates, pvc)
		}
	}

	if len(unitset.Labels) != 0 {
		for k, v := range unitset.Labels {
			unit.Labels[k] = v
		}
	}
	unit.Labels[upmiov1alpha2.UnitsetName] = unitset.Name
	unit.Labels[upmiov1alpha2.LabelUnitsCount] = strconv.Itoa(unitset.Spec.Units)

	if len(unitset.Annotations) != 0 {
		for k, v := range unitset.Annotations {
			unit.Annotations[k] = v
		}
	}

	unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] = unitset.Spec.Type
	unit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = unitset.Spec.Version

	if unit.Spec.Template.Labels == nil {
		unit.Spec.Template.Labels = make(map[string]string)
	}
	unit.Spec.Template.Labels = podTemplate.Template.Labels

	if unit.Spec.Template.Annotations == nil {
		unit.Spec.Template.Annotations = make(map[string]string)
	}
	unit.Spec.Template.Annotations = podTemplate.Template.Annotations

	unit.Spec.Template.Spec = *podTemplate.Template.Spec.DeepCopy()

	unit.Spec.Template.Spec.Subdomain = unitset.HeadlessServiceName()
	enableServiceLinks := true
	unit.Spec.Template.Spec.EnableServiceLinks = &enableServiceLinks
	unit.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("%s-serviceaccount", req.Namespace)

	fillAnnotations(&unit, unitset)
	fillVolumeMountsAndVolumes(&unit, volumeMounts, volumes)
	fillEnvs(&unit, unitset, envVars, ports)
	fillResourcesToDefaultContainer(&unit, unitset)
	fillNodeAffinity(&unit, unitset)
	fillPodAffinity(&unit, unitset)

	return unit, nil
}

// fillAnnotations fills annotations from unitset to unit
// unitset.Spec.Storages
// unitset.Spec.EmptyDir
// unitset.Spec.CertificateProfile.Organizations
// unitset.Spec.ExtraVolume
func fillAnnotations(unit *upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) {
	if unit == nil || unitset == nil {
		return
	}

	if unit.Annotations == nil {
		unit.Annotations = make(map[string]string)
	}

	// unitset.Spec.Storages
	for _, one := range unitset.Spec.Storage {
		key := upmiov1alpha2.AnnotationStoragePrefix + one.Name + ".mountPath"
		unit.Annotations[key] = one.MountPath
	}

	// unitset.Spec.EmptyDir
	for _, one := range unitset.Spec.EmptyDir {
		key := upmiov1alpha2.AnnotationStoragePrefix + one.Name + ".mountPath"
		unit.Annotations[key] = one.MountPath
	}

	// unitset.Spec.CertificateProfile.Organizations
	if len(unitset.Spec.CertificateProfile.Organizations) != 0 && unitset.Spec.CertificateProfile.RootSecret != "" {
		unit.Annotations[upmiov1alpha2.AnnotationCertificatePrefix+"certMount"] = "/CERT_MOUNT"

		organizationsStr, _ := json.Marshal(unitset.Spec.CertificateProfile.Organizations)
		unit.Annotations[upmiov1alpha2.AnnotationCertificatePrefix+"certORG"] = string(organizationsStr)

	}

	// unitset.Spec.ExtraVolume
	for _, one := range unitset.Spec.ExtraVolume {
		switch volumeKind(one.Volume) {
		case "Secret":
			key := upmiov1alpha2.AnnotationExtraVolumePrefix + one.Volume.Name + ".secretName"
			unit.Annotations[key] = one.Volume.Secret.SecretName
		case "ConfigMap":
			key := upmiov1alpha2.AnnotationExtraVolumePrefix + one.Volume.Name + ".configMapName"
			unit.Annotations[key] = one.Volume.ConfigMap.Name
		default:
			klog.Errorf("unsupported volume kind: %s", volumeKind(one.Volume))
		}

		key := upmiov1alpha2.AnnotationExtraVolumePrefix + one.Volume.Name + ".volumeMountPath"
		unit.Annotations[key] = one.VolumeMountPath
	}
}

// volumeKind returns the type of volume as a string based on its underlying Kubernetes volume source.
//
// Parameters:
//   - vol: a v1.Volume object representing a Kubernetes volume
//
// Returns:
//   - string: the volume type ("Secret", "ConfigMap", or "Other")
func volumeKind(vol v1.Volume) string {
	switch {
	case vol.Secret != nil:
		return "Secret"
	case vol.ConfigMap != nil:
		return "ConfigMap"
	default:
		return "Other"
	}
}

//func volumeKindReflect(vol v1.Volume) string {
//	vsVal := reflect.ValueOf(vol.VolumeSource)
//	if vsVal.Kind() == reflect.Ptr || vsVal.Kind() == reflect.Interface {
//		if vsVal.IsNil() {
//			return "Other"
//		}
//		vsVal = vsVal.Elem()
//	}
//	vsType := vsVal.Type()
//	for i := 0; i < vsVal.NumField(); i++ {
//		f := vsVal.Field(i)
//		if f.Kind() == reflect.Ptr && !f.IsNil() {
//			return vsType.Field(i).Name
//		}
//	}
//	return "Other"
//}

func fillNodeAffinity(unit *upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) {
	matchExpressions := []v1.NodeSelectorRequirement{}
	if len(unitset.Spec.NodeAffinityPreset) != 0 {
		for i := range unitset.Spec.NodeAffinityPreset {
			matchExpressions = append(matchExpressions, v1.NodeSelectorRequirement{
				Key:      unitset.Spec.NodeAffinityPreset[i].Key,
				Operator: v1.NodeSelectorOpIn,
				Values:   unitset.Spec.NodeAffinityPreset[i].Values,
			})
		}
	}

	if len(matchExpressions) != 0 {
		if unit.Spec.Template.Spec.Affinity == nil {
			unit.Spec.Template.Spec.Affinity = &v1.Affinity{}
		}

		unit.Spec.Template.Spec.Affinity.NodeAffinity = &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: matchExpressions,
					},
				},
			},
		}
	}
}

func fillPodAffinity(unit *upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) {
	if unitset.Spec.PodAntiAffinityPreset != "" {
		switch unitset.Spec.PodAntiAffinityPreset {
		case "soft":
			matchExpressions := metav1.LabelSelectorRequirement{
				Key:      upmiov1alpha2.UnitsetName,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{unitset.Name},
			}

			if unit.Spec.Template.Spec.Affinity == nil {
				unit.Spec.Template.Spec.Affinity = &v1.Affinity{}
			}

			unit.Spec.Template.Spec.Affinity.PodAntiAffinity = &v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								matchExpressions,
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			}

			unit.Spec.Template.Spec.TopologySpreadConstraints =
				append(unit.Spec.Template.Spec.TopologySpreadConstraints, v1.TopologySpreadConstraint{
					MaxSkew:           1,
					TopologyKey:       "upm.api/node-group",
					WhenUnsatisfiable: v1.ScheduleAnyway,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							upmiov1alpha2.UnitsetName: unitset.Name,
						},
					},
				})

		case "hard":
			matchExpressions := metav1.LabelSelectorRequirement{
				Key:      upmiov1alpha2.UnitsetName,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{unitset.Name},
			}

			if unit.Spec.Template.Spec.Affinity == nil {
				unit.Spec.Template.Spec.Affinity = &v1.Affinity{}
			}

			unit.Spec.Template.Spec.Affinity.PodAntiAffinity = &v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								matchExpressions,
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			}

			unit.Spec.Template.Spec.TopologySpreadConstraints =
				append(unit.Spec.Template.Spec.TopologySpreadConstraints, v1.TopologySpreadConstraint{
					MaxSkew:           1,
					TopologyKey:       "upm.api/node-group",
					WhenUnsatisfiable: v1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							upmiov1alpha2.UnitsetName: unitset.Name,
						},
					},
				})
		}
	}
}

func fillResourcesToDefaultContainer(unit *upmiov1alpha2.Unit, unitset *upmiov1alpha2.UnitSet) {
	for i := range unit.Spec.Template.Spec.Containers {
		if unit.Spec.Template.Spec.Containers[i].Name == unitset.Spec.Type {
			unit.Spec.Template.Spec.Containers[i].Resources = unitset.Spec.Resources

			// Set ResizePolicy for the main container to enable in-place vertical scaling.
			// This is an immutable field and must be set at Pod creation time.
			if len(unitset.Spec.ResizePolicy) > 0 {
				unit.Spec.Template.Spec.Containers[i].ResizePolicy = unitset.Spec.ResizePolicy
			}
		}
	}
}

func fillEnvs(
	unit *upmiov1alpha2.Unit,
	unitset *upmiov1alpha2.UnitSet,
	mountEnvs []v1.EnvVar,
	ports []v1.ContainerPort) {

	klog.V(4).Infof("---------------------------------------")

	// Order: unitset, volume mount (including shared config), pod template
	// All containers need these

	firstEnvs := getFirstEnvs(unitset)
	secondEnvs := getSecondEnvs(mountEnvs)

	klog.V(4).Infof("---[fillEnvs] unit:[%s], [firstEnvs] len:[%d], Envs:[%v]", unit.Name, len(firstEnvs), getEnvNames(firstEnvs))
	klog.V(4).Infof("---[fillEnvs] unit:[%s], [secondEnvs] len:[%d],Envs:[%v]", unit.Name, len(secondEnvs), getEnvNames(secondEnvs))

	updateContainerEnvs(unit, firstEnvs, secondEnvs)

	klog.V(4).Infof("---------------------------------------")
}

func updateContainerEnvs(unit *upmiov1alpha2.Unit, firstEnvs, secondEnvs []v1.EnvVar) {
	for i := range unit.Spec.Template.Spec.InitContainers {

		var thirdEnvs = []v1.EnvVar{}
		if len(unit.Spec.Template.Spec.InitContainers[i].Env) != 0 {
			thirdEnvs = append(thirdEnvs, unit.Spec.Template.Spec.InitContainers[i].Env...)
		}

		klog.V(4).Infof("---[fillEnvs] unit:[%s] Container:[%s], [thirdEnvs] len:[%d], Envs:[%v]",
			unit.Name, unit.Spec.Template.Spec.InitContainers[i].Name, len(thirdEnvs), getEnvNames(thirdEnvs))

		unit.Spec.Template.Spec.InitContainers[i].Env = []v1.EnvVar{}

		needEnvs := []v1.EnvVar{}

		for _, env := range firstEnvs {
			needEnvs = addEnvVar(needEnvs, env)
		}
		for _, env := range secondEnvs {
			needEnvs = addEnvVar(needEnvs, env)
		}
		if len(thirdEnvs) != 0 {
			for _, env := range thirdEnvs {
				needEnvs = addEnvVar(needEnvs, env)
			}
		}

		unit.Spec.Template.Spec.InitContainers[i].Env = needEnvs

		klog.V(4).Infof("------[FILLENVS] unit:[%s] CONTAINERS:[%s], [ALL ENVS] len:[%d], Envs:[%v]------",
			unit.Name,
			unit.Spec.Template.Spec.InitContainers[i].Name,
			len(unit.Spec.Template.Spec.InitContainers[i].Env),
			getEnvNames(unit.Spec.Template.Spec.InitContainers[i].Env))
	}

	for i := range unit.Spec.Template.Spec.Containers {
		var thirdEnvs = []v1.EnvVar{}
		//thirdEnvs := unit.Spec.Template.Spec.Containers[i].Env
		if len(unit.Spec.Template.Spec.Containers[i].Env) != 0 {
			thirdEnvs = append(thirdEnvs, unit.Spec.Template.Spec.Containers[i].Env...)
		}
		klog.V(4).Infof("---[fillEnvs] unit:[%s] Container:[%s], [thirdEnvs] len:[%d], Envs:[%v]",
			unit.Name, unit.Spec.Template.Spec.Containers[i].Name, len(thirdEnvs), getEnvNames(thirdEnvs))

		unit.Spec.Template.Spec.Containers[i].Env = []v1.EnvVar{}

		needEnvs := []v1.EnvVar{}

		for _, env := range firstEnvs {
			needEnvs = addEnvVar(needEnvs, env)
		}
		for _, env := range secondEnvs {
			needEnvs = addEnvVar(needEnvs, env)
		}
		if len(thirdEnvs) != 0 {
			for _, env := range thirdEnvs {
				needEnvs = addEnvVar(needEnvs, env)
			}
		}

		unit.Spec.Template.Spec.Containers[i].Env = needEnvs

		klog.V(4).Infof("------[FILLENVS] unit:[%s] CONTAINERS:[%s], [ALL ENVS] len:[%d], Envs:[%v]------",
			unit.Name,
			unit.Spec.Template.Spec.Containers[i].Name,
			len(unit.Spec.Template.Spec.Containers[i].Env),
			getEnvNames(unit.Spec.Template.Spec.Containers[i].Env))
	}
}

func getFirstEnvs(unitset *upmiov1alpha2.UnitSet) []v1.EnvVar {

	firstEnvs := make([]v1.EnvVar, 0)
	if len(unitset.Spec.Env) != 0 {
		firstEnvs = append(firstEnvs, unitset.Spec.Env...)
	}

	return firstEnvs
}

func getSecondEnvs(mountEnvs []v1.EnvVar) []v1.EnvVar {

	secondEnvs := make([]v1.EnvVar, 0)
	if len(mountEnvs) != 0 {
		secondEnvs = append(secondEnvs, mountEnvs...)
	}

	return secondEnvs
}

func addEnvVar(envs []v1.EnvVar, newEnv v1.EnvVar) []v1.EnvVar {
	for _, env := range envs {
		if env.Name == newEnv.Name {
			return envs
		}
	}
	return append(envs, newEnv)
}

func getEnvNames(envs []v1.EnvVar) []string {
	names := make([]string, len(envs))
	for i, env := range envs {
		names[i] = env.Name
	}

	return names
}

func fillVolumeMountsAndVolumes(
	unit *upmiov1alpha2.Unit,
	volumeMounts []v1.VolumeMount,
	volumes []v1.Volume) {

	klog.V(4).Infof("fillVolumeMountsAndVolumes: volumeMounts len:[%d]", len(volumeMounts))

	if len(volumes) != 0 {
		unit.Spec.Template.Spec.Volumes = append(unit.Spec.Template.Spec.Volumes, volumes...)
	}

	// Need to deduplicate volumeMounts first
	volumeMountsMap := make(map[string]v1.VolumeMount)
	for _, mount := range volumeMounts {
		volumeMountsMap[mount.Name] = mount
	}
	volumeMounts = make([]v1.VolumeMount, 0)
	for _, mount := range volumeMountsMap {
		volumeMounts = append(volumeMounts, mount)
	}

	if len(volumeMounts) != 0 {
		if len(unit.Spec.Template.Spec.InitContainers) != 0 {
			for i := range unit.Spec.Template.Spec.InitContainers {
				for j := range volumeMounts {
					containerAddMounter(&unit.Spec.Template.Spec.InitContainers[i], volumeMounts[j])
				}
			}
		}

		if len(unit.Spec.Template.Spec.Containers) != 0 {
			for i := range unit.Spec.Template.Spec.Containers {
				for j := range volumeMounts {
					containerAddMounter(&unit.Spec.Template.Spec.Containers[i], volumeMounts[j])
				}
			}
		}
	}
}

func containerAddMounter(container *v1.Container, mounter v1.VolumeMount) {
	if container.VolumeMounts == nil {
		container.VolumeMounts = []v1.VolumeMount{mounter}
	} else {
		for _, existingMount := range container.VolumeMounts {
			if existingMount.Name == mounter.Name {
				return
			}
		}
		container.VolumeMounts = append(container.VolumeMounts, mounter)
	}
}

func generateVolumeMountsAndEnvs(unitset *upmiov1alpha2.UnitSet) ([]v1.VolumeMount, []v1.Volume, []v1.EnvVar, []upmiov1alpha2.UnitVolumeClaimTemplate) {

	var volumeClaimTemplates []upmiov1alpha2.UnitVolumeClaimTemplate
	var volumeMount []v1.VolumeMount
	var volumes []v1.Volume
	var envs []v1.EnvVar

	if len(unitset.Spec.Storage) != 0 {
		for _, one := range unitset.Spec.Storage {
			storageInfo := one
			//klog.Infof("generateVolumeMountsAndEnvs: storageInfo: %v", storageInfo)
			//klog.Infof("generateVolumeMountsAndEnvs: storage name: %s", storageInfo.Name)

			sc := storageInfo.StorageClassName
			volumeClaimTemplate := upmiov1alpha2.UnitVolumeClaimTemplate{
				Name: storageInfo.Name,
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					Resources: v1.VolumeResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceStorage: resource.MustParse(storageInfo.Size),
						},
					},
					StorageClassName: &sc,
				},
			}
			//klog.Infof("generateVolumeMountsAndEnvs: generated volumeClaimTemplate name: [%s]", volumeClaimTemplate.Name)
			volumeClaimTemplates = append(volumeClaimTemplates, volumeClaimTemplate)

			volumeMount = append(volumeMount, v1.VolumeMount{
				Name:      storageInfo.Name,
				MountPath: storageInfo.MountPath,
			})

			volumes = append(volumes, v1.Volume{
				Name: storageInfo.Name,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: "",
					},
				},
			})
		}
	}

	if len(unitset.Spec.EmptyDir) != 0 {
		for _, emptyDirInfo := range unitset.Spec.EmptyDir {
			volumeMount = append(volumeMount, v1.VolumeMount{
				Name:      emptyDirInfo.Name,
				MountPath: emptyDirInfo.MountPath,
			})

			size := resource.MustParse(emptyDirInfo.Size)
			volumes = append(volumes, v1.Volume{
				Name: emptyDirInfo.Name,
				VolumeSource: v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{
						SizeLimit: &size,
					},
				},
			})
		}
	}

	if len(unitset.Spec.CertificateProfile.Organizations) != 0 && unitset.Spec.CertificateProfile.RootSecret != "" {
		// certificate volume SecretName not here, will be filled in Unit level
		volume := v1.SecretVolumeSource{
			SecretName: "",
		}
		volumes = append(volumes, v1.Volume{
			Name: "certificate",
			VolumeSource: v1.VolumeSource{
				Secret: &volume,
			},
		})

		volumeMount = append(volumeMount, v1.VolumeMount{
			Name:      "certificate",
			ReadOnly:  true,
			MountPath: "/CERT_MOUNT",
		})
	}

	if len(unitset.Spec.ExtraVolume) != 0 {
		for _, extraVolumeInfo := range unitset.Spec.ExtraVolume {
			volumeMount = append(volumeMount, v1.VolumeMount{
				Name:      extraVolumeInfo.Volume.Name,
				MountPath: extraVolumeInfo.VolumeMountPath,
			})

			volumes = append(volumes, extraVolumeInfo.Volume)
		}
	}

	return volumeMount, volumes, envs, volumeClaimTemplates
}

func (r *UnitSetReconciler) getPodTemplate(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet) (v1.PodTemplate, error) {

	podTemplate := v1.PodTemplate{}
	err := r.Get(ctx, client.ObjectKey{Name: unitset.PodTemplateName(), Namespace: req.Namespace}, &podTemplate)
	if apierrors.IsNotFound(err) {
		templatePodTemplate := v1.PodTemplate{}
		templatePodTemplateNamespacedName := client.ObjectKey{Name: unitset.TemplatePodTemplateName(), Namespace: vars.ManagerNamespace}
		err = r.Get(ctx, templatePodTemplateNamespacedName, &templatePodTemplate)
		if err != nil {
			return podTemplate, err
		}

		// check the ports in main container (container name= unitset.spec.type)
		// ports are REQUIRED VALUE
		for _, one := range templatePodTemplate.Template.Spec.Containers {
			if one.Name == unitset.Spec.Type {
				if one.Ports == nil || len(one.Ports) == 0 {
					return v1.PodTemplate{}, fmt.Errorf("not found ports in pod template:[%s], "+
						"ports are Required value",
						templatePodTemplateNamespacedName.String())
				}
			}
		}

		ref := metav1.NewControllerRef(unitset, controllerKind)
		podTemplate = v1.PodTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:            unitset.PodTemplateName(),
				Namespace:       req.Namespace,
				Labels:          make(map[string]string),
				Annotations:     make(map[string]string),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
			Template: *templatePodTemplate.Template.DeepCopy(),
		}

		err = r.Create(ctx, &podTemplate)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return podTemplate, err
		}

	} else if err != nil {
		return podTemplate, err
	}

	return podTemplate, nil
}

func getPortsFromPodtemplate(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
	template v1.PodTemplate) []v1.ContainerPort {

	out := []v1.ContainerPort{}

	if template.Template.Spec.Containers == nil {
		return out
	}

	if len(template.Template.Spec.Containers) == 0 {
		return out
	}

	for _, one := range template.Template.Spec.Containers {
		if one.Name == unitset.Spec.Type {
			out = one.Ports
			break
		}
	}

	return out
}
