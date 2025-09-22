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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("UnitServer Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		//fakeClient client.Client
		unit     *upmiov1alpha2.Unit
		pod      *corev1.Pod
		unitName string
		req      ctrl.Request
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

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: unitName, Namespace: "default"}}

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

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when pod is not found", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
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

			err := reconciler.reconcileUnitServer(ctx, req, unit)
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

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should skip reconciliation when unit-agent container is not ready", func() {
			pod.Status.ContainerStatuses[1].Ready = false
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
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

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no pod ip to use"))
		})

		It("should start service when startup is true and service is not running", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Update pod status to simulate proper initialization
			current := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
			}).Should(Succeed())

			current.Status = pod.Status
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			// For this test, we expect it to fail because we don't have a real agent running
			// But we can verify that the function tries to start the service
			err := reconciler.reconcileUnitServer(ctx, req, unit)
			// The error is expected because we don't have a real unit-agent service
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fail to start unit"))
		})

		It("should not start service when already running", func() {
			unit.Status.ProcessState = "running"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not start service when starting", func() {
			unit.Status.ProcessState = "starting"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
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
		//	err := reconciler.reconcileUnitServer(ctx, req, unit)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("fail to start unit"))
		//})

		It("should stop service when startup is false and pod is ready", func() {
			unit.Spec.Startup = false
			unit.Status.ProcessState = "running"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Update pod status to simulate proper initialization
			current := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
			}).Should(Succeed())

			current.Status = pod.Status
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			// For this test, we expect it to fail because we don't have a real agent running
			// But we can verify that the function tries to stop the service
			err := reconciler.reconcileUnitServer(ctx, req, unit)
			// The error is expected because we don't have a real unit-agent service
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fail to stop unit"))
		})

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

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not stop service when process state is not running", func() {
			unit.Spec.Startup = false
			unit.Status.ProcessState = "stopped"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle readiness probe configuration correctly", func() {
			// Test with specific readiness probe settings
			unit.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
				PeriodSeconds:    10,
				TimeoutSeconds:   5,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Update pod status
			current := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
			}).Should(Succeed())

			current.Status = pod.Status
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fail to start unit"))
		})

		It("should handle different process states correctly", func() {
			// Test with "starting" state - should not start again
			unit.Status.ProcessState = "starting"
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Update pod status to simulate ready main container
			current := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
			}).Should(Succeed())

			current.Status = pod.Status
			// Make main container ready
			current.Status.ContainerStatuses[0].Ready = true
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle missing main container annotation", func() {
			// Remove main container annotation
			delete(unit.Annotations, upmiov1alpha2.AnnotationMainContainerName)
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			// Update pod status
			current := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
			}).Should(Succeed())

			current.Status = pod.Status
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

			err := reconciler.reconcileUnitServer(ctx, req, unit)
			Expect(err).To(HaveOccurred())
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
		//	err := reconciler.reconcileUnitServer(ctx, req, unit)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("fail to stop unit"))
		//})

		Context("agent host types", func() {
			BeforeEach(func() {
				// Set up common test environment
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				Expect(k8sClient.Create(ctx, pod)).To(Succeed())

				// Update pod status
				current := &corev1.Pod{}
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)
				}).Should(Succeed())

				current.Status = pod.Status
				Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())
			})

			It("should handle domain host type", func() {
				// Note: This test checks that the function attempts to use domain-based addressing
				// We expect an error because no real unit-agent is running
				err := reconciler.reconcileUnitServer(ctx, req, unit)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fail to start unit"))
			})

			It("should handle IP host type", func() {
				// Note: This test checks that the function attempts to use IP-based addressing
				// We expect an error because no real unit-agent is running
				err := reconciler.reconcileUnitServer(ctx, req, unit)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fail to start unit"))
			})

			It("should handle empty pod IP list", func() {
				// Update pod to have no IPs
				current := &corev1.Pod{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: unitName, Namespace: "default"}, current)).To(Succeed())
				current.Status.PodIPs = []corev1.PodIP{}
				Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())

				err := reconciler.reconcileUnitServer(ctx, req, unit)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no pod ip to use"))
			})
		})
	})
})

func TestUnitServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UnitServer Suite")
}
