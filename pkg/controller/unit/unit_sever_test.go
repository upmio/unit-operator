/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("UnitServer Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		//fakeClient client.Client
		unit     *upmiov1alpha2.Unit
		pod      *corev1.Pod
		unitName string
		//req        ctrl.Request
	)

	BeforeEach(func() {
		ctx = context.Background()

		// unique name
		suffix := time.Now().UnixNano()
		unitName = fmt.Sprintf("test-unit-%d", suffix)

		// cleanup leftovers to avoid AlreadyExists between specs
		_ = k8sClient.Delete(ctx, &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: unitName + "-data", Namespace: "default"}})

		// Create fake client
		//fakeClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

		// Create reconciler with fake client
		reconciler = &UnitReconciler{
			Client:   k8sClient,
			Scheme:   scheme.Scheme,
			Recorder: recorder,
			//Recorder: &serverTestRecorder{},
		}

		// Create test request
		//req = ctrl.Request{
		//	NamespacedName: types.NamespacedName{
		//		Name:      "test-unit",
		//		Namespace: "default",
		//	},
		//}

		// Create test unit
		unit = &upmiov1alpha2.Unit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
				Annotations: map[string]string{
					upmiov1alpha2.AnnotationMainContainerName: "main-container",
				},
				Labels: map[string]string{
					"unitset": "test-unitset",
				},
			},
			Spec: upmiov1alpha2.UnitSpec{
				Startup: true,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "main-container",
								Image: "nginx:latest",
							},
							{
								Name:  "unit-agent",
								Image: "unit-agent:latest",
							},
						},
					},
				},
			},
			Status: upmiov1alpha2.UnitStatus{
				ProcessState: "stopped",
			},
		}

		// Create test pod
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "main-container",
						Image: "nginx:latest",
					},
					{
						Name:  "unit-agent",
						Image: "unit-agent:latest",
					},
				},
			},
			Status: corev1.PodStatus{
				Phase:  corev1.PodRunning,
				PodIPs: []corev1.PodIP{{IP: "10.0.0.1"}},
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:  "main-container",
						Ready: true,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
					{
						Name:  "unit-agent",
						Ready: true,
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodInitialized,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   corev1.PodScheduled,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}
	})

	Context("reconcileUnitServer", func() {
		It("should skip reconciliation when unit is in maintenance mode", func() {
			unit.Annotations[upmiov1alpha2.AnnotationMaintenance] = "true"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when pod is not found", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should skip reconciliation when pod is not initialized", func() {
			pod.Status.Conditions = []corev1.PodCondition{
				{
					Type:   corev1.PodInitialized,
					Status: corev1.ConditionFalse,
				},
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should skip reconciliation when pod is not scheduled", func() {
			pod.Status.Conditions = []corev1.PodCondition{
				{
					Type:   corev1.PodScheduled,
					Status: corev1.ConditionFalse,
				},
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should skip reconciliation when unit-agent container is not ready", func() {
			pod.Status.ContainerStatuses[1].Ready = false
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when pod has no IP", func() {
			// create then update status to ensure initialized/scheduled and agent ready, but no IPs
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			current := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)).To(Succeed())
			current.Status.Conditions = []corev1.PodCondition{
				{Type: corev1.PodInitialized, Status: corev1.ConditionTrue},
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			}
			current.Status.ContainerStatuses = []corev1.ContainerStatus{
				{Name: "main-container", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{Name: "unit-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			}
			current.Status.PodIPs = []corev1.PodIP{}
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no pod ip to use"))
		})

		//It("should start service when startup is true and service is not running", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the ServiceLifecycleManagement function
		//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
		//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
		//		return "service started", nil
		//	}
		//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
		//
		//	err := reconciler.reconcileUnitServer(ctx, unit)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		It("should not start service when already running", func() {
			unit.Status.ProcessState = "running"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not start service when starting", func() {
			unit.Status.ProcessState = "starting"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		//It("should return error when service start fails", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the ServiceLifecycleManagement function to return error
		//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
		//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
		//		return "start failed", fmt.Errorf("service start failed")
		//	}
		//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
		//
		//	err := reconciler.reconcileUnitServer(ctx, unit)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("fail to start unit"))
		//})

		//It("should stop service when startup is false and pod is ready", func() {
		//	unit.Spec.Startup = false
		//	unit.Status.ProcessState = "running"
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the ServiceLifecycleManagement function
		//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
		//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
		//		return "service stopped", nil
		//	}
		//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
		//
		//	err := reconciler.reconcileUnitServer(ctx, unit)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		It("should not stop service when pod is not ready", func() {
			unit.Spec.Startup = false
			unit.Status.ProcessState = "stopped"
			pod.Status.Conditions = []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not stop service when process state is not running", func() {
			unit.Spec.Startup = false
			unit.Status.ProcessState = "stopped"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		//It("should return error when service stop fails", func() {
		//	unit.Spec.Startup = false
		//	unit.Status.ProcessState = "running"
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the ServiceLifecycleManagement function to return error
		//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
		//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
		//		return "stop failed", fmt.Errorf("service stop failed")
		//	}
		//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
		//
		//	err := reconciler.reconcileUnitServer(ctx, unit)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("fail to stop unit"))
		//})

		Context("agent host types", func() {
			//It("should use domain host type", func() {
			//	vars.UnitAgentHostType = "domain"
			//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			//
			//	// Mock the ServiceLifecycleManagement function
			//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
			//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
			//		Expect(host).To(Equal("test-unit"))
			//		return "service started", nil
			//	}
			//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
			//
			//	err := reconciler.reconcileUnitServer(ctx, unit)
			//	Expect(err).NotTo(HaveOccurred())
			//})

			//It("should use IP host type", func() {
			//	vars.UnitAgentHostType = "ip"
			//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			//
			//	// Mock the ServiceLifecycleManagement function
			//	originalServiceLifecycleManagement := internalAgent.ServiceLifecycleManagement
			//	internalAgent.ServiceLifecycleManagement = func(agentHostType, unitsetHeadlessSvc, host, namespace, port, actionType string) (string, error) {
			//		Expect(host).To(Equal("10.0.0.1"))
			//		return "service started", nil
			//	}
			//	defer func() { internalAgent.ServiceLifecycleManagement = originalServiceLifecycleManagement }()
			//
			//	err := reconciler.reconcileUnitServer(ctx, unit)
			//	Expect(err).NotTo(HaveOccurred())
			//})
		})
	})
})

func TestUnitServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UnitServer Suite")
}

// serverTestRecorder is a mock implementation of EventRecorder
type serverTestRecorder struct {
	events []client.Object
}

func (r *serverTestRecorder) Event(object client.Object, eventtype, reason, message string) {
	r.events = append(r.events, object)
}

func (r *serverTestRecorder) Eventf(object client.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	r.events = append(r.events, object)
}

func (r *serverTestRecorder) AnnotatedEventf(object client.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	r.events = append(r.events, object)
}
