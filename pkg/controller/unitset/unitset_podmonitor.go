package unitset

import (
	"context"
	"fmt"

	serviceMonitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	apiextensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcilePodMonitor(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	if !unitset.Spec.PodMonitor.Enable {
		return nil
	}

	exceptedCrd := apiextensionsV1.CustomResourceDefinition{}
	err := r.Get(ctx, client.ObjectKey{Name: upmiov1alpha2.MonitorPodMonitorCrdName}, &exceptedCrd)
	if err != nil && errors.IsNotFound(err) {
		// no crd found, creation of service monitor not supported
		r.Recorder.Eventf(unitset, "WARNNING", "PodMonitor",
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

	PodMetricsEndpointsPort := "exporter"

	podMonitorName := unitset.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix
	exceptedPodMonitor := &serviceMonitorv1.PodMonitor{}
	exceptedPodMonitorNamespacedName := client.ObjectKey{Name: podMonitorName, Namespace: unitset.Namespace}
	err = r.Get(ctx, exceptedPodMonitorNamespacedName, exceptedPodMonitor)
	if err != nil && errors.IsNotFound(err) {
		pm := &serviceMonitorv1.PodMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:            podMonitorName,
				Namespace:       unitset.Namespace,
				Labels:          unitset.Labels,
				OwnerReferences: []metav1.OwnerReference{*ref},
			},
			Spec: serviceMonitorv1.PodMonitorSpec{
				PodMetricsEndpoints: []serviceMonitorv1.PodMetricsEndpoint{
					{
						Port: &PodMetricsEndpointsPort,
					},
				},
				NamespaceSelector: serviceMonitorv1.NamespaceSelector{
					MatchNames: []string{
						unitset.Namespace,
					},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: make(map[string]string),
				},
			},
		}

		pm.Spec.Selector.MatchLabels[upmiov1alpha2.UnitsetName] = unitset.Name

		err = r.Create(ctx, pm)
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("create pod monitor:[%s] error:[%s]", podMonitorName, err.Error())
		}

		r.Recorder.Eventf(unitset, "SuccessCreated", "PodMonitor create", "create pod monitor:[%s] ok~", podMonitorName)
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("get pod monitor:[%s] error:[%s]", podMonitorName, err.Error())
	}

	return nil
}
