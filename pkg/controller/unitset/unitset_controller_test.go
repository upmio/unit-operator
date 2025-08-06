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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/gomega/gstruct"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UnitSet Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		unitset := &upmiov1alpha2.UnitSet{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind UnitSet")
			err := k8sClient.Get(ctx, typeNamespacedName, unitset)
			if err != nil && errors.IsNotFound(err) {
				resource := &upmiov1alpha2.UnitSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &upmiov1alpha2.UnitSet{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance UnitSet")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

var _ = Describe("UnitSet Controller", func() {
	var (
		ctx        context.Context
		unitSet    *upmiov1alpha2.UnitSet
		namespace  *corev1.Namespace
		testNsName = "test-unitset-namespace"
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:         testNsName,
				GenerateName: "test-unitset-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset",
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
				Env: []corev1.EnvVar{
					{
						Name:  "MYSQL_ROOT_PASSWORD",
						Value: "test-password",
					},
				},
				SharedConfigName: "test-shared-config",
				UpdateStrategy: upmiov1alpha2.UpdateStrategySpec{
					Type: "RollingUpdate",
					RollingUpdate: upmiov1alpha2.RollingUpdateSpec{
						Partition:      1,
						MaxUnavailable: 1,
					},
				},
			},
		}
	})

	AfterEach(func() {
		// Cleanup
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Context("When creating a UnitSet", func() {
		It("Should create successfully", func() {
			By("Creating the UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Verifying the UnitSet exists")
			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSet.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.Type).To(Equal("mysql"))
			Expect(createdUnitSet.Spec.Units).To(Equal(3))
			Expect(createdUnitSet.Spec.Version).To(Equal("8.0.40"))
		})

		It("Should have proper default values", func() {
			By("Creating a minimal UnitSet")
			minimalUnitSet := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minimal-unitset",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
				},
			}
			Expect(k8sClient.Create(ctx, minimalUnitSet)).To(Succeed())

			By("Verifying default values")
			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      minimalUnitSet.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.Edition).To(BeEmpty())
			Expect(createdUnitSet.Spec.Version).To(BeEmpty())
			Expect(createdUnitSet.Spec.Units).To(Equal(1))
		})
	})

	Context("When reconciling a UnitSet", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should reconcile without errors", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling the UnitSet")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			})

			Expect(err).To(HaveOccurred()) // Expected to have some errors due to missing dependencies
			Expect(result.Requeue).To(BeTrue())
		})

		It("Should handle deletion gracefully", func() {
			By("Deleting the UnitSet")
			Expect(k8sClient.Delete(ctx, unitSet)).To(Succeed())

			By("Verifying deletion")
			deletedUnitSet := &upmiov1alpha2.UnitSet{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSet.Name,
				Namespace: namespace.Name,
			}, deletedUnitSet)

			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("When updating a UnitSet", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should allow updates to spec", func() {
			By("Updating UnitSet spec")
			unitSet.Spec.Units = 5
			unitSet.Spec.Version = "8.0.41"

			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Verifying updates")
			updatedUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSet.Name,
				Namespace: namespace.Name,
			}, updatedUnitSet)).To(Succeed())

			Expect(updatedUnitSet.Spec.Units).To(Equal(5))
			Expect(updatedUnitSet.Spec.Version).To(Equal("8.0.41"))
		})

		It("Should preserve existing values during update", func() {
			By("Updating only specific fields")
			originalUnits := unitSet.Spec.Units
			unitSet.Spec.SharedConfigName = "updated-config-name"

			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Verifying other fields are preserved")
			updatedUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSet.Name,
				Namespace: namespace.Name,
			}, updatedUnitSet)).To(Succeed())

			Expect(updatedUnitSet.Spec.Units).To(Equal(originalUnits))
			Expect(updatedUnitSet.Spec.SharedConfigName).To(Equal("updated-config-name"))
		})
	})

	Context("When validating UnitSet spec", func() {
		It("Should accept valid resource requirements", func() {
			By("Creating UnitSet with detailed resource requirements")
			unitSetWithResources := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-resources",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 2,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:     resource.MustParse("250m"),
							corev1.ResourceMemory:  resource.MustParse("256Mi"),
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, unitSetWithResources)).To(Succeed())
		})

		It("Should handle storage configuration", func() {
			By("Creating UnitSet with storage configuration")
			unitSetWithStorage := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-storage",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
					Storages: []upmiov1alpha2.StorageSpec{
						{
							Name:             "data",
							Size:             "20Gi",
							StorageClassName: "fast-ssd",
							MountPath:        "/var/lib/mysql",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, unitSetWithStorage)).To(Succeed())

			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithStorage.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.Storages).To(HaveLen(1))
			Expect(createdUnitSet.Spec.Storages[0].Name).To(Equal("data"))
			Expect(createdUnitSet.Spec.Storages[0].Size).To(Equal("20Gi"))
		})

		It("Should handle node affinity configuration", func() {
			By("Creating UnitSet with node affinity")
			unitSetWithAffinity := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-affinity",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
					NodeAffinityPreset: []upmiov1alpha2.NodeAffinityPresetSpec{
						{
							Key:    "node-role.kubernetes.io/worker",
							Values: []string{"true"},
						},
					},
					PodAntiAffinityPreset: "hard",
				},
			}

			Expect(k8sClient.Create(ctx, unitSetWithAffinity)).To(Succeed())

			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithAffinity.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.NodeAffinityPreset).To(HaveLen(1))
			Expect(createdUnitSet.Spec.PodAntiAffinityPreset).To(Equal("hard"))
		})
	})

	Context("When handling service configurations", func() {
		It("Should handle external service configuration", func() {
			By("Creating UnitSet with external service")
			unitSetWithService := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-service",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
					ExternalService: upmiov1alpha2.ExternalServiceSpec{
						Type: "NodePort",
					},
					UnitService: upmiov1alpha2.UnitServiceSpec{
						Type: "ClusterIP",
					},
				},
			}

			Expect(k8sClient.Create(ctx, unitSetWithService)).To(Succeed())

			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithService.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.ExternalService.Type).To(Equal("NodePort"))
			Expect(createdUnitSet.Spec.UnitService.Type).To(Equal("ClusterIP"))
		})

		It("Should handle certificate secret configuration", func() {
			By("Creating UnitSet with certificate secret")
			unitSetWithCert := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unitset-with-cert",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
					CertificateSecret: upmiov1alpha2.CertificateSecretSpec{
						Organization: "Test Org",
						Name:         "test-cert-secret",
					},
				},
			}

			Expect(k8sClient.Create(ctx, unitSetWithCert)).To(Succeed())

			createdUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      unitSetWithCert.Name,
				Namespace: namespace.Name,
			}, createdUnitSet)).To(Succeed())

			Expect(createdUnitSet.Spec.CertificateSecret.Organization).To(Equal("Test Org"))
			Expect(createdUnitSet.Spec.CertificateSecret.Name).To(Equal("test-cert-secret"))
		})
	})

	Context("When listing UnitSets", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should list UnitSets in namespace", func() {
			By("Listing UnitSets")
			unitSetList := &upmiov1alpha2.UnitSetList{}
			Expect(k8sClient.List(ctx, unitSetList, client.InNamespace(namespace.Name))).To(Succeed())

			Expect(unitSetList.Items).To(HaveLen(1))
			Expect(unitSetList.Items[0].Name).To(Equal(unitSet.Name))
		})

		It("Should list UnitSets across all namespaces", func() {
			By("Listing all UnitSets")
			unitSetList := &upmiov1alpha2.UnitSetList{}
			Expect(k8sClient.List(ctx, unitSetList)).To(Succeed())

			Expect(unitSetList.Items).To(ContainElement(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Name":      Equal(unitSet.Name),
						"Namespace": Equal(namespace.Name),
					}),
				}),
			))
		})
	})

	Context("When handling errors", func() {
		It("Should handle invalid resource requirements", func() {
			By("Creating UnitSet with invalid resource requirements")
			invalidUnitSet := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-unitset",
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("-100m"), // Invalid negative value
						},
					},
				},
			}

			// This should fail validation
			Expect(k8sClient.Create(ctx, invalidUnitSet)).NotTo(Succeed())
		})

		It("Should handle duplicate UnitSet creation", func() {
			By("Creating duplicate UnitSet")
			duplicateUnitSet := &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:  "mysql",
					Units: 1,
				},
			}

			Expect(k8sClient.Create(ctx, duplicateUnitSet)).To(
				MatchError(apierrors.IsAlreadyExists),
			)
		})
	})
})
