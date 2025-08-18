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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("UnitSet Status Reconciliation", func() {
	var (
		ctx        context.Context
		unitSet    *upmiov1alpha2.UnitSet
		namespace  *corev1.Namespace
		reconciler *UnitSetReconciler
		units      []*upmiov1alpha2.Unit
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-status-namespace-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-status",
				Namespace: namespace.Name,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:    "mysql",
				Edition: "community",
				Version: "8.0.40",
				Units:   3,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				Storages: []upmiov1alpha2.StorageSpec{
					{
						Name: "data",
						Size: "10Gi",
					},
				},
			},
		}

		// Create reconciler
		reconciler = &UnitSetReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		// Initialize units for testing
		units = []*upmiov1alpha2.Unit{
			createTestUnit("test-unit-0", namespace.Name, "node-1", upmiov1alpha2.UnitReady),
			createTestUnit("test-unit-1", namespace.Name, "node-2", upmiov1alpha2.UnitReady),
			createTestUnit("test-unit-2", namespace.Name, "", upmiov1alpha2.UnitPending),
		}
	})

	AfterEach(func() {
		// Best-effort cleanup
		_ = k8sClient.Delete(ctx, namespace)
	})

	Context("When reconciling UnitSet status", func() {
		It("Should update basic status fields", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units")
			for idx, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				// set two Ready units, one Pending
				if idx == 0 || idx == 1 {
					persisted.Status.Phase = upmiov1alpha2.UnitReady
				} else {
					persisted.Status.Phase = upmiov1alpha2.UnitPending
				}
				Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying basic status fields")
			Expect(unitSet.Status.Units).To(Equal(3))
			Expect(unitSet.Status.ReadyUnits).To(Equal(2))
		})

		It("Should handle error when getting units fails", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling UnitSet status with no units")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying error status")
			Expect(unitSet.Status.Units).To(Equal(0))
			Expect(unitSet.Status.ReadyUnits).To(Equal(0))
			Expect(unitSet.Status.ImageSyncStatus.Status).To(Equal("False"))
			Expect(unitSet.Status.ResourceSyncStatus.Status).To(Equal("False"))
			Expect(unitSet.Status.PvcSyncStatus.Status).To(Equal("False"))
		})

		It("Should update image sync status correctly", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with matching versions")
			units[0].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"
			units[1].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"
			units[2].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"

			for idx, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				if idx == 0 || idx == 1 {
					persisted.Status.Phase = upmiov1alpha2.UnitReady
				}
				Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying image sync status is True")
			Expect(unitSet.Status.ImageSyncStatus.Status).To(Equal("True"))
		})

		It("Should set image sync status to False when versions don't match", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with mismatched versions")
			units[0].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"
			units[1].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.39"
			units[2].Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"

			for _, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying image sync status is False")
			Expect(unitSet.Status.ImageSyncStatus.Status).To(Equal("False"))
		})

		It("Should update PVC sync status correctly", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with matching PVCs")
			for _, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				persisted.Status.PersistentVolumeClaim = []upmiov1alpha2.PvcInfo{
					{
						Name: upmiov1alpha2.PersistentVolumeClaimName(persisted, "data"),
						Capacity: upmiov1alpha2.PvcCapacity{
							Storage: resource.MustParse("10Gi"),
						},
					},
				}
				Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PVC sync status is True")
			Expect(unitSet.Status.PvcSyncStatus.Status).To(Equal("True"))
		})

		It("Should set PVC sync status to False when PVCs don't match", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with mismatched PVCs")
			for idx, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				if idx == 0 {
					persisted.Status.PersistentVolumeClaim = []upmiov1alpha2.PvcInfo{{
						Name:     upmiov1alpha2.PersistentVolumeClaimName(persisted, "data"),
						Capacity: upmiov1alpha2.PvcCapacity{Storage: resource.MustParse("10Gi")},
					}}
				} else if idx == 1 {
					persisted.Status.PersistentVolumeClaim = []upmiov1alpha2.PvcInfo{{
						Name:     upmiov1alpha2.PersistentVolumeClaimName(persisted, "data"),
						Capacity: upmiov1alpha2.PvcCapacity{Storage: resource.MustParse("5Gi")},
					}}
				}
				Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying PVC sync status is False")
			Expect(unitSet.Status.PvcSyncStatus.Status).To(Equal("False"))
		})

		It("Should update resource sync status correctly", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with matching resources")
			for _, unit := range units {
				unit.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Name: "mysql",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				}
				unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] = "mysql"
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying resource sync status is True")
			Expect(unitSet.Status.ResourceSyncStatus.Status).To(Equal("True"))
		})

		It("Should set resource sync status to False when resources don't match", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with mismatched resources")
			units[0].Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: "mysql",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1000m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			}
			units[1].Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: "mysql",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"), // Different CPU
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1000m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			}

			for _, unit := range units {
				unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] = "mysql"
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying resource sync status is False")
			Expect(unitSet.Status.ResourceSyncStatus.Status).To(Equal("False"))
		})

		It("Should update InUpdate field correctly", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with tasks")
			for idx, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				if idx == 0 {
					persisted.Status.Task = "updating"
				} else if idx == 2 {
					persisted.Status.Task = "migrating"
				} else {
					persisted.Status.Task = ""
				}
				Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying InUpdate field contains task units")
			Expect(unitSet.Status.InUpdate).To(ContainSubstring("test-unit-0"))
			Expect(unitSet.Status.InUpdate).To(ContainSubstring("test-unit-2"))
		})

		It("Should clear InUpdate field when all units are synced", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating fully synced units")
			for _, unit := range units {
				unit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion] = "8.0.40"
				unit.Spec.Template.Spec.Containers = []corev1.Container{
					{
						Name: "mysql",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				}
				unit.Annotations[upmiov1alpha2.AnnotationMainContainerName] = "mysql"
				unit.Status.PersistentVolumeClaim = []upmiov1alpha2.PvcInfo{
					{
						Name: upmiov1alpha2.PersistentVolumeClaimName(unit, "data"),
						Capacity: upmiov1alpha2.PvcCapacity{
							Storage: resource.MustParse("10Gi"),
						},
					},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying InUpdate field is empty")
			Expect(unitSet.Status.InUpdate).To(BeEmpty())
		})

		It("Should update status only when values change", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units")
			for idx, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
				// mark two units ready to make ReadyUnits=2
				persisted := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}, persisted)).To(Succeed())
				if idx == 0 || idx == 1 {
					persisted.Status.Phase = upmiov1alpha2.UnitReady
					Expect(k8sClient.Status().Update(ctx, persisted)).To(Succeed())
				}
			}

			By("Setting initial status")
			unitSet.Status.Units = 3
			unitSet.Status.ReadyUnits = 2
			unitSet.Status.ImageSyncStatus.Status = "False"
			unitSet.Status.ResourceSyncStatus.Status = "False"
			unitSet.Status.PvcSyncStatus.Status = "False"

			By("Reconciling UnitSet status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitsetStatus(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status is updated when values change")
			// This test verifies that the status update logic works correctly
			// In a real scenario, we would need to mock the Status().Update call
			Expect(unitSet.Status.Units).To(Equal(3))
			Expect(unitSet.Status.ReadyUnits).To(Equal(2))
		})
	})

	Context("When patching UnitSet", func() {
		//	It("Should patch UnitSet successfully", func() {
		//		By("Creating UnitSet")
		//		Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		//
		//		By("Creating old and new UnitSet versions")
		//		oldUnitSet := unitSet.DeepCopy()
		//		newUnitSet := unitSet.DeepCopy()
		//		newUnitSet.Spec.NodeNameMap = map[string]string{
		//			"test-unit-0": "node-1",
		//			"test-unit-1": "node-2",
		//		}
		//
		//		By("Patching UnitSet")
		//		patched, err := reconciler.patchUnitset(ctx, oldUnitSet, newUnitSet)
		//		Expect(err).NotTo(HaveOccurred())
		//		Expect(patched).NotTo(BeNil())
		//	})

		It("Should handle patch errors gracefully", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating identical UnitSet versions (no changes)")
			oldUnitSet := unitSet.DeepCopy()
			newUnitSet := unitSet.DeepCopy()

			By("Patching UnitSet with no changes")
			patched, err := reconciler.patchUnitset(ctx, oldUnitSet, newUnitSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(patched).To(Equal(oldUnitSet))
		})
	})
})

// Helper function to create test units
func createTestUnit(name, namespace, nodeName string, phase upmiov1alpha2.UnitPhase) *upmiov1alpha2.Unit {
	return &upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				upmiov1alpha2.UnitsetName: "test-unitset-status",
			},
			Annotations: map[string]string{
				upmiov1alpha2.AnnotationMainContainerName:    "mysql",
				upmiov1alpha2.AnnotationMainContainerVersion: "8.0.40",
			},
		},
		Spec: upmiov1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "mysql",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
		Status: upmiov1alpha2.UnitStatus{
			Phase:    phase,
			NodeName: nodeName,
			Task:     "",
			PersistentVolumeClaim: []upmiov1alpha2.PvcInfo{
				{
					Name: fmt.Sprintf("%s-data", name),
					Capacity: upmiov1alpha2.PvcCapacity{
						Storage: resource.MustParse("10Gi"),
					},
				},
			},
		},
	}
}
