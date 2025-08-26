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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("UnitConfig Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		unit       *upmiov1alpha2.Unit
		pod        *corev1.Pod
		unitName   string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// unique name
		suffix := time.Now().UnixNano()
		unitName = fmt.Sprintf("test-unit-%d", suffix)

		// cleanup leftovers
		_ = k8sClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})

		reconciler = &UnitReconciler{Client: k8sClient, Scheme: scheme.Scheme, Recorder: recorder}

		// Create test unit
		unit = &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}, Spec: upmiov1alpha2.UnitSpec{Startup: true, ConfigTemplateName: "test-config-template", ConfigValueName: "test-config-value", Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main-container", Image: "nginx:latest"}, {Name: "unit-agent", Image: "unit-agent:latest"}}}}}}

		// Create test pod
		pod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main-container", Image: "nginx:latest"}, {Name: "unit-agent", Image: "unit-agent:latest"}}}, Status: corev1.PodStatus{PodIPs: []corev1.PodIP{{IP: "192.168.1.100"}}, Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "main-container", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, Ready: false}, {Name: "unit-agent", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}, Ready: true}}, Conditions: []corev1.PodCondition{{Type: corev1.PodInitialized, Status: corev1.ConditionTrue}}}}
	})

	Context("When unit is in maintenance mode", func() {
		It("should skip reconcile and return nil", func() {
			unit.Annotations = map[string]string{upmiov1alpha2.AnnotationMaintenance: "true"}
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When pod is not found", func() {
		It("should return error when pod cannot be found", func() {
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Context("When pod is not initialized", func() {
		BeforeEach(func() {
			pod.Status.PodIPs = []corev1.PodIP{}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		})

		It("should return nil when pod is not initialized", func() {
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When main container is ready", func() {
		BeforeEach(func() {
			pod.Status.ContainerStatuses[0].Ready = true
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		})

		It("should return nil when main container is ready", func() {
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When unit-agent container is not ready", func() {
		BeforeEach(func() {
			pod.Status.ContainerStatuses[1].Ready = false
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		})

		It("should return nil when unit-agent container is not ready", func() {
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When pod has no IP", func() {
		BeforeEach(func() {
			// create first, then update status to simulate initialized + agent ready but no IPs
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			current := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unitName, Namespace: "default"}, current)).To(Succeed())
			current.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodInitialized, Status: corev1.ConditionTrue}}
			if len(current.Status.ContainerStatuses) < 2 {
				current.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "main-container", Ready: false, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, {Name: "unit-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}
			} else {
				current.Status.ContainerStatuses[1].Ready = true
			}
			current.Status.PodIPs = []corev1.PodIP{}
			Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())
		})

		It("should return error when pod has no IP", func() {
			err := reconciler.reconcileUnitConfig(ctx, unit)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no pod ip"))
		})
	})
})

func TestUnitConfig(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "UnitConfig Suite") }
