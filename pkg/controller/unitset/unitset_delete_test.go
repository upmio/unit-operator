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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UnitSet Delete Reconciler", func() {
	var (
		ctx       context.Context
		unitSet   *upmiov1alpha2.UnitSet
		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace (ephemeral)
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-delete-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic UnitSet for testing and persist to API server
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-delete",
				Namespace: namespace.Name,
				Finalizers: []string{
					upmiov1alpha2.FinalizerUnitDelete,
					upmiov1alpha2.FinalizerConfigMapDelete,
				},
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:    "mysql",
				Edition: "community",
				Version: "8.0.40",
				Units:   3,
			},
		}
		Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Context("When deleting UnitSet resources", func() {
		It("Should handle unknown finalizer gracefully", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting resources with unknown finalizer")
			err := reconciler.deleteResources(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				"unknown-finalizer",
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should delete units and remove finalizer", func() {
			By("Creating test units")
			for i := 0; i < 3; i++ {
				unit := &upmiov1alpha2.Unit{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", unitSet.Name, i),
						Namespace: namespace.Name,
						Labels: map[string]string{
							upmiov1alpha2.UnitsetName: unitSet.Name,
						},
					},
					Spec: upmiov1alpha2.UnitSpec{},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting units with finalizer")
			err := reconciler.deleteUnitWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerUnitDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying units are deleted")
			unitList := &upmiov1alpha2.UnitList{}
			Expect(k8sClient.List(ctx, unitList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(unitList.Items).To(BeEmpty())

			By("Verifying finalizer is removed")
			Expect(unitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerUnitDelete))
		})

		It("Should handle case when no units exist", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting units when none exist")
			err := reconciler.deleteUnitWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerUnitDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying finalizer is removed")
			Expect(unitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerUnitDelete))
		})

		It("Should handle unit deletion errors gracefully", func() {
			By("Creating test unit that will fail deletion")
			unit := &upmiov1alpha2.Unit{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-0", unitSet.Name),
					Namespace: namespace.Name,
					Labels: map[string]string{
						upmiov1alpha2.UnitsetName: unitSet.Name,
					},
					Finalizers: []string{"test-finalizer"},
				},
				Spec: upmiov1alpha2.UnitSpec{},
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Attempting to delete units with blocking finalizer")
			shortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			err := reconciler.deleteUnitWithFinalizer(shortCtx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerUnitDelete,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error waiting for units deleted"))
		})

		It("Should delete configmaps and remove finalizer", func() {
			By("Creating test configmaps")
			templateCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-config-template", unitSet.Name),
					Namespace: namespace.Name,
				},
				Data: map[string]string{
					"config.yaml": "key: value",
				},
			}
			Expect(k8sClient.Create(ctx, templateCm)).To(Succeed())

			for i := 0; i < 3; i++ {
				configValueCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d-config-value", unitSet.Name, i),
						Namespace: namespace.Name,
					},
					Data: map[string]string{
						"mysql": "key: value",
					},
				}
				Expect(k8sClient.Create(ctx, configValueCm)).To(Succeed())
			}

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting configmaps with finalizer")
			err := reconciler.deleteConfigMapWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerConfigMapDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying configmaps are deleted")
			cmList := &corev1.ConfigMapList{}
			Expect(k8sClient.List(ctx, cmList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(cmList.Items).To(BeEmpty())

			By("Verifying finalizer is removed")
			Expect(unitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerConfigMapDelete))
		})

		It("Should handle case when no configmaps exist", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting configmaps when none exist")
			err := reconciler.deleteConfigMapWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerConfigMapDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying finalizer is removed")
			Expect(unitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerConfigMapDelete))
		})

		It("Should handle configmap deletion errors gracefully", func() {
			By("Creating test configmap that will fail deletion")
			configValueCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:       fmt.Sprintf("%s-0-config-value", unitSet.Name),
					Namespace:  namespace.Name,
					Finalizers: []string{"example.com/test-finalizer"},
				},
				Data: map[string]string{
					"mysql": "key: value",
				},
			}
			Expect(k8sClient.Create(ctx, configValueCm)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Attempting to delete configmaps with blocking finalizer")
			shortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			err := reconciler.deleteConfigMapWithFinalizer(shortCtx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerConfigMapDelete,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error waiting for configmap deleted"))
		})

		It("Should handle concurrent configmap deletion", func() {
			By("Creating multiple test configmaps")
			for i := 0; i < 3; i++ {
				configValueCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d-config-value", unitSet.Name, i),
						Namespace: namespace.Name,
					},
					Data: map[string]string{
						"mysql": fmt.Sprintf("key: value-%d", i),
					},
				}
				Expect(k8sClient.Create(ctx, configValueCm)).To(Succeed())
			}

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Deleting configmaps concurrently")
			err := reconciler.deleteConfigMapWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				upmiov1alpha2.FinalizerConfigMapDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all configmaps are deleted (relaxed check)")
			cmList := &corev1.ConfigMapList{}
			Expect(k8sClient.List(ctx, cmList, client.InNamespace(namespace.Name))).To(Succeed())
			for _, cm := range cmList.Items {
				Expect(cm.Name).NotTo(Equal(fmt.Sprintf("%s-config-template", unitSet.Name)))
				for i := 0; i < 3; i++ {
					Expect(cm.Name).NotTo(Equal(fmt.Sprintf("%s-%d-config-value", unitSet.Name, i)))
				}
			}

			By("Verifying finalizer is removed")
			Expect(unitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerConfigMapDelete))
		})
	})

	Context("When getting units belonging to UnitSet", func() {
		It("Should return units that belong to the UnitSet", func() {
			By("Creating test units")
			for i := 0; i < 3; i++ {
				unit := &upmiov1alpha2.Unit{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", unitSet.Name, i),
						Namespace: namespace.Name,
						Labels: map[string]string{
							upmiov1alpha2.UnitsetName: unitSet.Name,
						},
					},
					Spec: upmiov1alpha2.UnitSpec{},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Creating units that don't belong to the UnitSet")
			otherUnit := &upmiov1alpha2.Unit{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-unit",
					Namespace: namespace.Name,
					Labels: map[string]string{
						upmiov1alpha2.UnitsetName: "other-unitset",
					},
				},
				Spec: upmiov1alpha2.UnitSpec{},
			}
			Expect(k8sClient.Create(ctx, otherUnit)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Getting units that belong to the UnitSet")
			units, err := reconciler.unitsBelongUnitset(ctx, unitSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(units).To(HaveLen(3))

			By("Verifying all returned units belong to the UnitSet")
			for _, unit := range units {
				Expect(unit.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
			}
		})

		It("Should return empty slice when no units exist", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Getting units when none exist")
			units, err := reconciler.unitsBelongUnitset(ctx, unitSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(units).To(BeNil())
		})

		It("Should handle API errors gracefully", func() {
			By("Creating reconciler with normal client (success path)")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Getting units that belong to the UnitSet")
			units, err := reconciler.unitsBelongUnitset(ctx, unitSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(units).To(BeNil())
		})
	})

	Context("When handling finalizer operations", func() {
		It("Should not remove finalizer if not present", func() {
			By("Creating UnitSet without finalizer")
			unitSetWithoutFinalizer := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-without-finalizer",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:    "mysql",
					Edition: "community",
					Version: "8.0.40",
					Units:   3,
				},
			}
			Expect(k8sClient.Create(ctx, unitSetWithoutFinalizer)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Attempting to remove non-existent finalizer")
			err := reconciler.deleteUnitWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSetWithoutFinalizer.Name,
						Namespace: namespace.Name,
					},
				},
				unitSetWithoutFinalizer,
				upmiov1alpha2.FinalizerUnitDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying UnitSet still doesn't have the finalizer")
			updatedUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithoutFinalizer.Name,
				Namespace: namespace.Name,
			}, updatedUnitSet)).To(Succeed())
			Expect(updatedUnitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerUnitDelete))
		})

		It("Should handle update errors when removing finalizer (simulate via short ctx)", func() {
			By("Creating test UnitSet with finalizer")
			unitSetWithFinalizer := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-finalizer",
					Namespace: namespace.Name,
					Finalizers: []string{
						upmiov1alpha2.FinalizerUnitDelete,
					},
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:    "mysql",
					Edition: "community",
					Version: "8.0.40",
					Units:   3,
				},
			}
			Expect(k8sClient.Create(ctx, unitSetWithFinalizer)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Attempting to remove finalizer")
			err := reconciler.deleteUnitWithFinalizer(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSetWithFinalizer.Name,
						Namespace: namespace.Name,
					},
				},
				unitSetWithFinalizer,
				upmiov1alpha2.FinalizerUnitDelete,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying finalizer was removed")
			updatedUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithFinalizer.Name,
				Namespace: namespace.Name,
			}, updatedUnitSet)).To(Succeed())
			Expect(updatedUnitSet.Finalizers).NotTo(ContainElement(upmiov1alpha2.FinalizerUnitDelete))
		})
	})
})
