package unitset

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	serviceMonitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	apiextensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("UnitSet PodMonitor Reconciler", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		Expect(upmiov1alpha2.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(serviceMonitorv1.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(apiextensionsV1.AddToScheme(scheme.Scheme)).To(Succeed())
	})

	newUnitSet := func() *upmiov1alpha2.UnitSet {
		return &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-podmonitor",
				Namespace: "test-ns-podmonitor",
				UID:       types.UID("test-unitset-podmonitor"),
				Labels: map[string]string{
					"app": "milvus",
				},
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:    "milvus",
				Version: "2.5.0",
				Units:   1,
				PodMonitor: upmiov1alpha2.PodMonitorInfo{
					Enable: true,
				},
			},
		}
	}

	newPodMonitorCRD := func() *apiextensionsV1.CustomResourceDefinition {
		return &apiextensionsV1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: upmiov1alpha2.MonitorPodMonitorCrdName},
		}
	}

	newRequest := func(unitset *upmiov1alpha2.UnitSet) ctrl.Request {
		return ctrl.Request{NamespacedName: types.NamespacedName{Name: unitset.Name, Namespace: unitset.Namespace}}
	}

	newReconciler := func(objects ...interface{ metav1.Object }) *UnitSetReconciler {
		clientObjects := make([]client.Object, 0, len(objects))
		for _, object := range objects {
			clientObject, ok := object.(client.Object)
			Expect(ok).To(BeTrue())
			clientObjects = append(clientObjects, clientObject)
		}

		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(clientObjects...).Build()
		return &UnitSetReconciler{Client: c, Scheme: scheme.Scheme, Recorder: record.NewFakeRecorder(10)}
	}

	It("should create pod monitor with default metrics endpoint", func() {
		unitSet := newUnitSet()
		reconciler := newReconciler(unitSet, newPodMonitorCRD())

		err := reconciler.reconcilePodMonitor(ctx, newRequest(unitSet), unitSet)
		Expect(err).NotTo(HaveOccurred())

		created := &serviceMonitorv1.PodMonitor{}
		Expect(reconciler.Get(ctx, types.NamespacedName{
			Name:      unitSet.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix,
			Namespace: unitSet.Namespace,
		}, created)).To(Succeed())

		Expect(created.Spec.PodMetricsEndpoints).To(HaveLen(1))
		Expect(created.Spec.PodMetricsEndpoints[0].Port).NotTo(BeNil())
		Expect(*created.Spec.PodMetricsEndpoints[0].Port).To(Equal(defaultPodMonitorEndpointPort))
		Expect(created.Spec.PodMetricsEndpoints[0].RelabelConfigs).To(BeNil())
	})

	It("should create pod monitor with explicit endpoint port", func() {
		unitSet := newUnitSet()
		unitSet.Spec.PodMonitor.Endpoints = []upmiov1alpha2.PodMonitorEndpoint{{Port: "milvus-metrics"}}
		reconciler := newReconciler(unitSet, newPodMonitorCRD())

		err := reconciler.reconcilePodMonitor(ctx, newRequest(unitSet), unitSet)
		Expect(err).NotTo(HaveOccurred())

		created := &serviceMonitorv1.PodMonitor{}
		Expect(reconciler.Get(ctx, types.NamespacedName{
			Name:      unitSet.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix,
			Namespace: unitSet.Namespace,
		}, created)).To(Succeed())

		Expect(created.Spec.PodMetricsEndpoints).To(HaveLen(1))
		Expect(created.Spec.PodMetricsEndpoints[0].Port).NotTo(BeNil())
		Expect(*created.Spec.PodMetricsEndpoints[0].Port).To(Equal("milvus-metrics"))
	})

	It("should update existing pod monitor when relabel configs are added", func() {
		unitSet := newUnitSet()
		unitSet.Spec.PodMonitor.Endpoints = []upmiov1alpha2.PodMonitorEndpoint{{
			RelabelConfigs: []upmiov1alpha2.PodMonitorRelabelConfig{{
				TargetLabel: "pod",
				Replacement: "$1",
				Action:      "replace",
			}},
		}}

		existingPort := defaultPodMonitorEndpointPort
		existing := &serviceMonitorv1.PodMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitSet.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix,
				Namespace: unitSet.Namespace,
			},
			Spec: serviceMonitorv1.PodMonitorSpec{
				PodMetricsEndpoints: []serviceMonitorv1.PodMetricsEndpoint{{Port: &existingPort}},
				NamespaceSelector:   serviceMonitorv1.NamespaceSelector{MatchNames: []string{unitSet.Namespace}},
				Selector:            metav1.LabelSelector{MatchLabels: map[string]string{upmiov1alpha2.UnitsetName: unitSet.Name}},
			},
		}

		reconciler := newReconciler(unitSet, newPodMonitorCRD(), existing)

		err := reconciler.reconcilePodMonitor(ctx, newRequest(unitSet), unitSet)
		Expect(err).NotTo(HaveOccurred())

		updated := &serviceMonitorv1.PodMonitor{}
		Expect(reconciler.Get(ctx, types.NamespacedName{
			Name:      unitSet.Name + upmiov1alpha2.MonitorPodMonitorNameSuffix,
			Namespace: unitSet.Namespace,
		}, updated)).To(Succeed())

		Expect(updated.Spec.PodMetricsEndpoints).To(HaveLen(1))
		Expect(updated.Spec.PodMetricsEndpoints[0].Port).NotTo(BeNil())
		Expect(*updated.Spec.PodMetricsEndpoints[0].Port).To(Equal(defaultPodMonitorEndpointPort))
		Expect(updated.Spec.PodMetricsEndpoints[0].RelabelConfigs).To(HaveLen(1))
		relabel := updated.Spec.PodMetricsEndpoints[0].RelabelConfigs[0]
		Expect(relabel.TargetLabel).To(Equal("pod"))
		Expect(relabel.Replacement).NotTo(BeNil())
		Expect(*relabel.Replacement).To(Equal("$1"))
		Expect(relabel.Action).To(Equal("replace"))
	})
})
