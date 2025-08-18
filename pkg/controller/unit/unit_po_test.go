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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("UnitPO Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		unit       *upmiov1alpha2.Unit
		pod        *corev1.Pod
		req        ctrl.Request
		unitName   string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// unique name to avoid cross-spec interference
		suffix := time.Now().UnixNano()
		unitName = fmt.Sprintf("test-unit-%d", suffix)

		// cleanup leftovers (best-effort)
		_ = k8sClient.Delete(ctx, &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})

		reconciler = &UnitReconciler{Client: k8sClient, Scheme: scheme.Scheme, Recorder: recorder}

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: unitName, Namespace: "default"}}

		// Create test unit
		unit = &upmiov1alpha2.Unit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
				Labels: map[string]string{
					"app":        "test-app",
					"unit-label": "test-value",
				},
				Annotations: map[string]string{
					upmiov1alpha2.AnnotationMainContainerName: "main-container",
					"unit-annotation":                         "test-value",
				},
			},
			Spec: upmiov1alpha2.UnitSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{"template-label": "template-value"},
						Annotations: map[string]string{"template-annotation": "template-value"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "main-container",
								Image: "nginx:1.0.0",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
									Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
								},
								Env: []corev1.EnvVar{{Name: "ENV1", Value: "value1"}, {Name: "ENV2", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"}}}},
							},
							{
								Name:  "sidecar",
								Image: "busybox:1.0.0",
							},
						},
						NodeName: "node-1",
					},
				},
			},
		}

		// Create test pod
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
				Labels:    map[string]string{"old-label": "old-value"},
				Annotations: map[string]string{
					"old-annotation": "old-value",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "main-container",
						Image: "nginx:1.0.0",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
							Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
						},
						Env: []corev1.EnvVar{{Name: "ENV1", Value: "value1"}, {Name: "ENV2", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"}}}},
					},
					{
						Name:  "sidecar",
						Image: "busybox:1.0.0",
					},
				},
				NodeName: "node-1",
			},
		}
	})

	Context("reconcilePod", func() {
		It("should create pod when not found", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			err := reconciler.reconcilePod(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			createdPod := &corev1.Pod{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, createdPod)).To(Succeed())
			Expect(createdPod.Name).To(Equal(unitName))
			Expect(createdPod.Namespace).To(Equal("default"))
		})

		//It("should handle pod creation error", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//
		//	// Mock Create to return error
		//	originalCreate := k8sClient.Create
		//	k8sClient.Create = func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
		//		if _, ok := obj.(*corev1.Pod); ok {
		//			return fmt.Errorf("creation failed")
		//		}
		//		return originalCreate(ctx, obj, opts...)
		//	}
		//	defer func() { k8sClient.Create = originalCreate }()
		//
		//	err := reconciler.reconcilePod(ctx, req, unit)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("creation failed"))
		//})

		It("should handle existing pod without upgrade", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			err := reconciler.reconcilePod(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})

		//It("should upgrade pod when needed", func() {
		//	// Modify pod to need upgrade
		//	pod.Spec.Containers[0].Image = "nginx:2.0.0"
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock reconcileUnitServer and wait functions
		//	originalReconcileUnitServer := reconciler.reconcileUnitServer
		//	reconciler.reconcileUnitServer = func(ctx context.Context, unit *upmiov1alpha2.Unit) error {
		//		return nil
		//	}
		//	defer func() { reconciler.reconcileUnitServer = originalReconcileUnitServer }()
		//
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.reconcilePod(ctx, req, unit)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		It("should patch pod when needed", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			err := reconciler.reconcilePod(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("ifNeedPatchPod", func() {
		//It("should return patch when labels differ", func() {
		//	unit.Labels["new-label"] = "new-value"
		//
		//	patch, need, err := ifNeedPatchPod(unit, pod)
		//	Expect(err).NotTo(HaveOccurred())
		//	Expect(need).To(BeTrue())
		//	Expect(len(patch)).To(BeGreaterThan(0))
		//})
		//
		//It("should return patch when annotations differ", func() {
		//	unit.Annotations["new-annotation"] = "new-value"
		//
		//	patch, need, err := ifNeedPatchPod(unit, pod)
		//	Expect(err).NotTo(HaveOccurred())
		//	Expect(need).To(BeTrue())
		//	Expect(len(patch)).To(BeGreaterThan(0))
		//})

		It("should return no patch when no differences", func() {
			pod.Labels = unit.Labels
			pod.Annotations = unit.Annotations
			patch, need, err := ifNeedPatchPod(unit, pod)
			Expect(err).NotTo(HaveOccurred())
			Expect(need).To(BeFalse())
			Expect(string(patch)).To(SatisfyAny(Equal(""), Equal("{}")))
		})

		It("should handle empty pod without error", func() {
			invalidPod := &corev1.Pod{}
			patch, need, err := ifNeedPatchPod(unit, invalidPod)
			Expect(err).NotTo(HaveOccurred())
			Expect(need).To(BeTrue())
			Expect(len(patch)).To(BeNumerically(">", 0))
		})
	})

	Context("generatePatchPod", func() {
		It("should update labels from unit", func() {
			patchedPod := generatePatchPod(unit, pod)

			Expect(patchedPod.Labels).To(HaveKey("app"))
			Expect(patchedPod.Labels["app"]).To(Equal("test-app"))
			Expect(patchedPod.Labels).To(HaveKey("unit-label"))
			Expect(patchedPod.Labels["unit-label"]).To(Equal("test-value"))
		})

		It("should update annotations from unit", func() {
			patchedPod := generatePatchPod(unit, pod)

			Expect(patchedPod.Annotations).To(HaveKey("unit-annotation"))
			Expect(patchedPod.Annotations["unit-annotation"]).To(Equal("test-value"))
		})

		It("should preserve existing node name when unit has empty node name", func() {
			unit.Spec.Template.Spec.NodeName = ""
			pod.Spec.NodeName = "existing-node"

			patchedPod := generatePatchPod(unit, pod)
			Expect(patchedPod.Spec.NodeName).To(Equal("existing-node"))
		})

		It("should update node name from unit when specified", func() {
			unit.Spec.Template.Spec.NodeName = "new-node"
			pod.Spec.NodeName = "existing-node"

			patchedPod := generatePatchPod(unit, pod)
			Expect(patchedPod.Spec.NodeName).To(Equal("new-node"))
		})

		It("should update non-main container images", func() {
			unit.Spec.Template.Spec.Containers[1].Image = "busybox:2.0.0"

			patchedPod := generatePatchPod(unit, pod)
			Expect(patchedPod.Spec.Containers[1].Image).To(Equal("busybox:2.0.0"))
		})

		It("should not update main container image", func() {
			unit.Spec.Template.Spec.Containers[0].Image = "nginx:2.0.0"

			patchedPod := generatePatchPod(unit, pod)
			Expect(patchedPod.Spec.Containers[0].Image).To(Equal("nginx:1.0.0"))
		})
	})

	Context("convert2Pod", func() {
		It("should create pod from unit template", func() {
			createdPod, err := convert2Pod(unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdPod.Name).To(Equal(unit.Name))
			Expect(createdPod.Namespace).To(Equal(unit.Namespace))
			Expect(createdPod.Labels).To(HaveKey("template-label"))
			Expect(createdPod.Annotations).To(HaveKey("template-annotation"))
			Expect(createdPod.Spec.Containers).To(HaveLen(2))
			Expect(createdPod.OwnerReferences).To(HaveLen(1))
			Expect(createdPod.OwnerReferences[0].Name).To(Equal(unit.Name))
		})
	})

	Context("getPodsLabelSet", func() {
		It("should merge template and unit labels", func() {
			labels := getPodsLabelSet(unit)

			Expect(labels).To(HaveKey("template-label"))
			Expect(labels["template-label"]).To(Equal("template-value"))
			Expect(labels).To(HaveKey("app"))
			Expect(labels["app"]).To(Equal("test-app"))
			Expect(labels).To(HaveKey("unit-label"))
			Expect(labels["unit-label"]).To(Equal("test-value"))
		})

		It("should prioritize template labels over unit labels", func() {
			unit.Labels["template-label"] = "unit-value"

			labels := getPodsLabelSet(unit)
			Expect(labels["template-label"]).To(Equal("template-value"))
		})
	})

	Context("getPodsAnnotationSet", func() {
		It("should merge template and unit annotations", func() {
			annotations := getPodsAnnotationSet(unit)

			Expect(annotations).To(HaveKey("template-annotation"))
			Expect(annotations["template-annotation"]).To(Equal("template-value"))
			Expect(annotations).To(HaveKey("unit-annotation"))
			Expect(annotations["unit-annotation"]).To(Equal("test-value"))
		})

		It("should prioritize template annotations over unit annotations", func() {
			unit.Annotations["template-annotation"] = "unit-value"

			annotations := getPodsAnnotationSet(unit)
			Expect(annotations["template-annotation"]).To(Equal("template-value"))
		})
	})

	Context("ifNeedUpgradePod", func() {
		It("should detect image change", func() {
			pod.Spec.Containers[0].Image = "nginx:2.0.0"

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("image changed"))
		})

		It("should detect CPU change", func() {
			pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("200m")

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("cpu changed"))
		})

		It("should detect memory change", func() {
			pod.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse("256Mi")

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("memory changed"))
		})

		It("should detect env value change", func() {
			pod.Spec.Containers[0].Env[0].Value = "changed-value"

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("env changed"))
		})

		It("should detect env valueFrom change", func() {
			pod.Spec.Containers[0].Env[1].ValueFrom = nil

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("env changed"))
		})

		It("should detect node affinity failure", func() {
			pod.Status.Phase = corev1.PodFailed
			pod.Status.Reason = "NodeAffinity"
			pod.Spec.NodeName = "node-1"

			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeTrue())
			Expect(reason).To(Equal("Pod Predicate NodeAffinity failed"))
		})

		It("should not need upgrade when everything matches", func() {
			reason, needUpgrade := ifNeedUpgradePod(unit, pod)
			Expect(needUpgrade).To(BeFalse())
			Expect(reason).To(BeEmpty())
		})
	})

	Context("LoopCompareEnv", func() {
		It("should return true when env vars match", func() {
			unitEnvs := []corev1.EnvVar{{Name: "ENV1", Value: "value1"}}
			podEnvs := []corev1.EnvVar{{Name: "ENV1", Value: "value1"}}

			result := LoopCompareEnv(unitEnvs, podEnvs)
			Expect(result).To(BeTrue())
		})

		It("should return false when env var values differ", func() {
			unitEnvs := []corev1.EnvVar{{Name: "ENV1", Value: "value1"}}
			podEnvs := []corev1.EnvVar{{Name: "ENV1", Value: "value2"}}

			result := LoopCompareEnv(unitEnvs, podEnvs)
			Expect(result).To(BeFalse())
		})

		It("should return false when env var valueFrom differs", func() {
			unitEnvs := []corev1.EnvVar{{Name: "ENV1", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}}}
			podEnvs := []corev1.EnvVar{{Name: "ENV1", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}}}

			result := LoopCompareEnv(unitEnvs, podEnvs)
			Expect(result).To(BeFalse())
		})

		It("should return false when unit env not found in pod", func() {
			unitEnvs := []corev1.EnvVar{{Name: "ENV1", Value: "value1"}}
			podEnvs := []corev1.EnvVar{{Name: "ENV2", Value: "value2"}}

			result := LoopCompareEnv(unitEnvs, podEnvs)
			Expect(result).To(BeFalse())
		})

		It("should handle nil env vars", func() {
			result := LoopCompareEnv(nil, nil)
			Expect(result).To(BeFalse())
		})
	})

	Context("waitUntilPodScheduled", func() {
		It("should return scheduled pod", func() {
			pod.Spec.NodeName = "node-1"
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			result, err := reconciler.waitUntilPodScheduled(ctx, unitName, "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Name).To(Equal(unitName))
		})

		//It("should handle pod not found", func() {
		//	// Mock the wait function to simulate pod not found
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate timeout
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	_, err := reconciler.waitUntilPodScheduled(ctx, "test-unit", "default")
		//	Expect(err).To(HaveOccurred())
		//})
	})
})

func TestUnitPO(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "UnitPO Suite") }
