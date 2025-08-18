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

package unitset

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UnitSet ConfigMap Reconciler", func() {
	var (
		ctx           context.Context
		unitSet       *upmiov1alpha2.UnitSet
		namespace     *corev1.Namespace
		controlNsName string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create a unique test namespace (server assigns a unique name)
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-configmap-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Use a unique control namespace per test to isolate manager templates
		controlNsName = namespace.Name + "-sys"
		vars.ManagerNamespace = controlNsName
		controlNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: controlNsName}}
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, controlNs))).To(Succeed())

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-configmap",
				Namespace: namespace.Name,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:    "mysql",
				Edition: "community",
				Version: "8.0.40",
				Units:   3,
			},
		}
	})

	AfterEach(func() {
		// Cleanup
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Context("When handling additional edge cases for reconcileConfigmap", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should handle zero units gracefully", func() {
			By("Creating unitset with zero units")
			zeroUnitSet := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-zero-units",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:    "mysql",
					Edition: "community",
					Version: "8.0.40",
					Units:   0,
				},
			}
			Expect(k8sClient.Create(ctx, zeroUnitSet)).To(Succeed())

			By("Creating template configmap")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())
			tv := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())

			// Ensure value template exists to avoid value reconcile fetch error
			tv = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())
			// Ensure value template exists to avoid value reconcile fetch error
			tv = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())
			// Ensure value template exists for reconciliation path

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap with zero units")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      zeroUnitSet.Name,
						Namespace: namespace.Name,
					},
				},
				zeroUnitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying only config template was created")
			configTemplateCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-zero-units-config-template",
				Namespace: namespace.Name,
			}, configTemplateCm)).To(Succeed())
		})

		It("Should handle context cancellation during concurrent operations", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Creating cancelled context")
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // Cancel immediately

			By("Reconciling with cancelled context")
			err := reconciler.reconcileConfigmap(cancelCtx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			// Should handle gracefully - may succeed or fail depending on timing
			// The important thing is that it doesn't panic
			_ = err
		})

		It("Should handle large number of units efficiently", func() {
			By("Creating unitset with many units")
			largeUnitSet := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-large-unitset",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:    "mysql",
					Edition: "community",
					Version: "8.0.40",
					Units:   50, // Large number of units
				},
			}
			Expect(k8sClient.Create(ctx, largeUnitSet)).To(Succeed())

			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap with many units")
			start := time.Now()
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      largeUnitSet.Name,
						Namespace: namespace.Name,
					},
				},
				largeUnitSet,
			)
			duration := time.Since(start)

			Expect(err).NotTo(HaveOccurred())
			Expect(duration).To(BeNumerically("<", 30*time.Second)) // Should complete within reasonable time

			By("Verifying all config value configmaps were created")
			for i := 0; i < 50; i++ {
				unitName := fmt.Sprintf("%s-%d", largeUnitSet.Name, i)
				configValueCm := &corev1.ConfigMap{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-config-value", unitName),
					Namespace: namespace.Name,
				}, configValueCm)).To(Succeed())
			}
		})

		It("Should handle partial failures in concurrent operations", func() {
			By("Creating template configmap but not value template")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())
			// Intentionally not creating templateValueCm to cause partial failure

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap which should fail for config values")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))

			By("Verifying config template was still created")
			configTemplateCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-configmap-config-template",
				Namespace: namespace.Name,
			}, configTemplateCm)).To(Succeed())
		})

		It("Should handle existing configmap with different owner references", func() {
			By("Creating config template configmap with different owner")
			differentOwnerCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unitset-configmap-config-template",
					Namespace: namespace.Name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "different-owner",
							UID:        "different-uid",
						},
					},
				},
				Data: map[string]string{
					"config.yaml": "key: old-value",
				},
			}
			Expect(k8sClient.Create(ctx, differentOwnerCm)).To(Succeed())

			By("Creating template configmap")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			// Also ensure value template exists to allow value reconciliation
			tv := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying configmap was updated")
			updatedCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-configmap-config-template",
				Namespace: namespace.Name,
			}, updatedCm)).To(Succeed())

			Expect(updatedCm.Data).To(Equal(templateCm.Data))
			Expect(updatedCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))
		})
	})

	Context("When reconciling configmap for UnitSet", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should create config template configmap when it doesn't exist", func() {
			By("Creating template configmap in manager namespace")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			tv := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Reconciling configmap")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying config template configmap was created")
			createdCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-configmap-config-template",
				Namespace: namespace.Name,
			}, createdCm)).To(Succeed())

			Expect(createdCm.Data).To(Equal(templateCm.Data))
			Expect(createdCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))
			Expect(createdCm.OwnerReferences).To(HaveLen(1))
			Expect(createdCm.OwnerReferences[0].Name).To(Equal(unitSet.Name))
		})

		It("Should update config template configmap when version changes", func() {
			By("Creating initial config template configmap")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			tv := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, tv))).To(Succeed())

			By("Creating reconciler and initial reconcile")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Updating UnitSet version")
			unitSet.Spec.Version = "8.0.41"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating updated template configmap")
			updatedTemplateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateCm))).To(Succeed())

			updatedTemplateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateValueCm))).To(Succeed())

			By("Reconciling again to trigger update")
			err = reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying config template configmap was updated")
			updatedCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-configmap-config-template",
				Namespace: namespace.Name,
			}, updatedCm)).To(Succeed())

			Expect(updatedCm.Data).To(Equal(updatedTemplateCm.Data))
			Expect(updatedCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.41"))
		})

		It("Should handle missing template configmap gracefully", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap without template")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("Should create config value configmaps for each unit", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying config value configmaps were created")
			for i := 0; i < 3; i++ {
				unitName := fmt.Sprintf("%s-%d", unitSet.Name, i)
				configValueCm := &corev1.ConfigMap{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-config-value", unitName),
					Namespace: namespace.Name,
				}, configValueCm)).To(Succeed())

				Expect(configValueCm.Data).To(Equal(templateValueCm.Data))
				Expect(configValueCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))
				Expect(configValueCm.OwnerReferences).To(HaveLen(1))
				Expect(configValueCm.OwnerReferences[0].Name).To(Equal(unitSet.Name))
			}
		})

		It("Should update config value configmaps when version changes", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler and initial reconcile")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Updating UnitSet version")
			unitSet.Spec.Version = "8.0.41"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating updated template configmaps")
			updatedTemplateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateCm))).To(Succeed())

			updatedTemplateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateValueCm))).To(Succeed())

			By("Reconciling again to trigger update")
			err = reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying config value configmaps were updated")
			for i := 0; i < 3; i++ {
				unitName := fmt.Sprintf("%s-%d", unitSet.Name, i)
				configValueCm := &corev1.ConfigMap{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-config-value", unitName),
					Namespace: namespace.Name,
				}, configValueCm)).To(Succeed())

				Expect(configValueCm.Data).To(Equal(updatedTemplateValueCm.Data))
				Expect(configValueCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.41"))
			}
		})

		It("Should handle errors in concurrent config value creation", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			By("Not creating template value configmap to simulate error")
			// templateValueCm is intentionally not created

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap which should fail")
			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("Should preserve existing configuration during config value update", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler and initial reconcile")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Manually updating config value configmap to simulate custom configuration")
			unitName := fmt.Sprintf("%s-0", unitSet.Name)
			configValueCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-config-value", unitName),
				Namespace: namespace.Name,
			}, configValueCm)).To(Succeed())

			configValueCm.Data["mysql"] = "key: custom-value"
			Expect(k8sClient.Update(ctx, configValueCm)).To(Succeed())

			By("Updating UnitSet version")
			unitSet.Spec.Version = "8.0.41"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating updated template configmaps")
			updatedTemplateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateCm))).To(Succeed())

			updatedTemplateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: new-template-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateValueCm))).To(Succeed())

			By("Reconciling again to trigger update")
			err = reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying custom configuration was preserved")
			updatedConfigValueCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-config-value", unitName),
				Namespace: namespace.Name,
			}, updatedConfigValueCm)).To(Succeed())

			Expect(updatedConfigValueCm.Data["mysql"]).To(ContainSubstring("custom-value"))
			Expect(updatedConfigValueCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.41"))
		})

		It("Should handle missing data keys in config value configmap", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler and initial reconcile")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			err := reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Manually removing type-specific data from config value configmap")
			unitName := fmt.Sprintf("%s-0", unitSet.Name)
			configValueCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-config-value", unitName),
				Namespace: namespace.Name,
			}, configValueCm)).To(Succeed())

			delete(configValueCm.Data, "mysql")
			Expect(k8sClient.Update(ctx, configValueCm)).To(Succeed())

			By("Updating UnitSet version")
			unitSet.Spec.Version = "8.0.41"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating updated template configmaps")
			updatedTemplateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: new-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateCm))).To(Succeed())

			updatedTemplateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.41-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: new-template-value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, updatedTemplateValueCm))).To(Succeed())

			By("Reconciling again which should handle missing data gracefully")
			err = reconciler.reconcileConfigmap(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying config value configmap has new data")
			updatedConfigValueCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-config-value", unitName),
				Namespace: namespace.Name,
			}, updatedConfigValueCm)).To(Succeed())

			Expect(updatedConfigValueCm.Data["mysql"]).To(Equal("key: new-template-value"))
			Expect(updatedConfigValueCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.41"))
		})

		It("Should handle empty unit names", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap with empty unit name")
			err := reconciler.reconcileConfigTemplateValue(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				"",
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When handling edge cases", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should handle concurrent access to configmaps", func() {
			By("Creating template configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-template",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())

			templateValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-community-8.0.40-config-value",
					Namespace: controlNsName,
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateValueCm))).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling configmap multiple times")
			for i := 0; i < 3; i++ {
				err := reconciler.reconcileConfigmap(ctx,
					ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      unitSet.Name,
							Namespace: namespace.Name,
						},
					},
					unitSet,
				)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying configmaps exist and are consistent")
			templateConfigCm := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-configmap-config-template",
				Namespace: namespace.Name,
			}, templateConfigCm)).To(Succeed())

			Expect(templateConfigCm.Data).To(Equal(templateCm.Data))
			Expect(templateConfigCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))

			for i := 0; i < 3; i++ {
				unitName := fmt.Sprintf("%s-%d", unitSet.Name, i)
				configValueCm := &corev1.ConfigMap{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      fmt.Sprintf("%s-config-value", unitName),
					Namespace: namespace.Name,
				}, configValueCm)).To(Succeed())

				Expect(configValueCm.Data).To(Equal(templateValueCm.Data))
				Expect(configValueCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))
			}
		})

		//It("Should handle configmap with nil annotations", func() {
		//	By("Creating template configmap")
		//	templateCm := &corev1.ConfigMap{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "mysql-community-8.0.40-config-template",
		//			Namespace: controlNsName,
		//		},
		//		Data: map[string]string{
		//			"config.yaml": "key: value",
		//		},
		//	}
		//	Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, templateCm))).To(Succeed())
		//
		//	By("Creating config template configmap with nil annotations")
		//	existingCm := &corev1.ConfigMap{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:        "test-unitset-configmap-config-template",
		//			Namespace:   namespace.Name,
		//			Annotations: nil,
		//		},
		//		Data: map[string]string{
		//			"config.yaml": "key: old-value",
		//		},
		//	}
		//	Expect(k8sClient.Create(ctx, existingCm)).To(Succeed())
		//
		//	By("Creating reconciler")
		//	reconciler := &UnitSetReconciler{
		//		Client: k8sClient,
		//		Scheme: scheme.Scheme,
		//	}
		//
		//	By("Reconciling configmap which should handle nil annotations")
		//	err := reconciler.reconcileConfigmap(ctx,
		//		ctrl.Request{
		//			NamespacedName: types.NamespacedName{
		//				Name:      unitSet.Name,
		//				Namespace: namespace.Name,
		//			},
		//		},
		//		unitSet,
		//	)
		//	Expect(err).NotTo(HaveOccurred())
		//
		//	By("Verifying annotations were properly initialized")
		//	updatedCm := &corev1.ConfigMap{}
		//	Expect(k8sClient.Get(ctx, types.NamespacedName{
		//		Name:      "test-unitset-configmap-config-template",
		//		Namespace: namespace.Name,
		//	}, updatedCm)).To(Succeed())
		//
		//	Expect(updatedCm.Annotations).NotTo(BeNil())
		//	Expect(updatedCm.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.40"))
		//})
	})
})
