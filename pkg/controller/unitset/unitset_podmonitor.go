package unitset

import (
	"context"
	"fmt"

	serviceMonitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
	apiextensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultPodMonitorEndpointPort = "metrics"

func (r *UnitSetReconciler) reconcilePodMonitor(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	if !unitset.Spec.PodMonitor.Enable {
		return nil
	}

	exceptedCrd := apiextensionsV1.CustomResourceDefinition{}
	err := r.Get(ctx, client.ObjectKey{Name: upmiov1alpha2.MonitorPodMonitorCrdName}, &exceptedCrd)
	if err != nil && errors.IsNotFound(err) {
		// no crd found, creation of service monitor not supported
		r.Recorder.Eventf(unitset, v1.EventTypeWarning, "PodMonitor",
			"[pod monitor enable=true], but not found podmonitor crd:[%s], ignore",
			upmiov1alpha2.MonitorPodMonitorCrdName)
		klog.Infof("[ensurePodMonitor] unitset:[%s], [pod monitor enable=true], "+
			"but not found podmonitor crd:[%s], ignore", req.String(), upmiov1alpha2.MonitorPodMonitorCrdName)
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("get crd:[%s] error:[%s]", upmiov1alpha2.MonitorPodMonitorCrdName, err.Error())
	}

	ref := metav1.NewControllerRef(unitset, controllerKind)
	podMonitorName := unitset.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix
	expectedSpec := buildExpectedPodMonitorSpec(unitset)

	exceptedPodMonitor := &serviceMonitorv1.PodMonitor{}
	exceptedPodMonitorNamespacedName := client.ObjectKey{Name: podMonitorName, Namespace: unitset.Namespace}
	err = r.Get(ctx, exceptedPodMonitorNamespacedName, exceptedPodMonitor)
	if err != nil && errors.IsNotFound(err) {
		pm := &serviceMonitorv1.PodMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:            podMonitorName,
				Namespace:       unitset.Namespace,
				Labels:          cloneStringMap(unitset.Labels),
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
			Spec: expectedSpec,
		}

		err = r.Create(ctx, pm)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create pod monitor:[%s] error:[%s]", podMonitorName, err.Error())
		}

		r.Recorder.Eventf(unitset, v1.EventTypeNormal, "PodMonitor create", "create pod monitor:[%s] ok~", podMonitorName)
		return nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("get pod monitor:[%s] error:[%s]", podMonitorName, err.Error())
	}

	if !equality.Semantic.DeepEqual(exceptedPodMonitor.Spec, expectedSpec) {
		exceptedPodMonitor.Spec = expectedSpec
		if err = r.Update(ctx, exceptedPodMonitor); err != nil {
			return fmt.Errorf("update pod monitor:[%s] error:[%s]", podMonitorName, err.Error())
		}
		r.Recorder.Eventf(unitset, v1.EventTypeNormal, "PodMonitor update", "update pod monitor:[%s] ok~", podMonitorName)
	}

	return nil
}

func buildExpectedPodMonitorSpec(unitset *upmiov1alpha2.UnitSet) serviceMonitorv1.PodMonitorSpec {
	return serviceMonitorv1.PodMonitorSpec{
		PodMetricsEndpoints: buildPodMetricsEndpoints(unitset.Spec.PodMonitor),
		NamespaceSelector: serviceMonitorv1.NamespaceSelector{
			MatchNames: []string{unitset.Namespace},
		},
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				upmiov1alpha2.UnitsetName: unitset.Name,
			},
		},
	}
}

func buildPodMetricsEndpoints(podMonitor upmiov1alpha2.PodMonitorInfo) []serviceMonitorv1.PodMetricsEndpoint {
	if len(podMonitor.Endpoints) == 0 {
		port := defaultPodMonitorEndpointPort
		return []serviceMonitorv1.PodMetricsEndpoint{{Port: &port}}
	}

	endpoints := make([]serviceMonitorv1.PodMetricsEndpoint, 0, len(podMonitor.Endpoints))
	for _, endpoint := range podMonitor.Endpoints {
		port := endpoint.Port
		if port == "" {
			port = defaultPodMonitorEndpointPort
		}

		endpoints = append(endpoints, serviceMonitorv1.PodMetricsEndpoint{
			Port:           &port,
			RelabelConfigs: buildRelabelConfigs(endpoint.RelabelConfigs),
		})
	}

	return endpoints
}

func buildRelabelConfigs(configs []upmiov1alpha2.PodMonitorRelabelConfig) []serviceMonitorv1.RelabelConfig {
	if len(configs) == 0 {
		return nil
	}

	relabelConfigs := make([]serviceMonitorv1.RelabelConfig, 0, len(configs))
	for _, config := range configs {
		relabelConfig := serviceMonitorv1.RelabelConfig{
			TargetLabel: config.TargetLabel,
			Action:      config.Action,
		}

		if config.Replacement != "" {
			replacement := config.Replacement
			relabelConfig.Replacement = &replacement
		}

		relabelConfigs = append(relabelConfigs, relabelConfig)
	}

	return relabelConfigs
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}

	return dst
}
