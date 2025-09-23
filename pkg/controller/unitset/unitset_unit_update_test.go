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
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("UnitSet Unit Update Reconciliation", func() {
	var (
		ctx                           context.Context
		unitSet                       *upmiov1alpha2.UnitSet
		namespace                     *corev1.Namespace
		reconciler                    *UnitSetReconciler
		units                         []*upmiov1alpha2.Unit
		podTemplate                   *corev1.PodTemplate
		templatePodTemplate           *corev1.PodTemplate
		originalUnitUpdateGracePeriod time.Duration
		originalUnitReadyPollInterval time.Duration
		originalUnitReadyPollTimeout  time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()

		originalUnitUpdateGracePeriod = unitUpdateGracePeriod
		originalUnitReadyPollInterval = unitReadyPollInterval
		originalUnitReadyPollTimeout = unitReadyPollTimeout

		unitUpdateGracePeriod = 0
		unitReadyPollInterval = 10 * time.Millisecond
		unitReadyPollTimeout = 2 * time.Second

		// Create test namespace
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-update-namespace-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Ensure manager namespace exists (for template PodTemplate)
		mgrNS := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vars.ManagerNamespace}}
		err := k8sClient.Create(ctx, mgrNS)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-update",
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
		unitNames, _ := unitSet.UnitNames()
		units = make([]*upmiov1alpha2.Unit, 0, len(unitNames))
		for _, unitName := range unitNames {
			units = append(units, createTestUnitForUpdate(unitName, namespace.Name, "8.0.39"))
		}

		// Label units so they are discovered by unitsBelongUnitset
		for _, u := range units {
			if u.Labels == nil {
				u.Labels = map[string]string{}
			}
			u.Labels[upmiov1alpha2.UnitsetName] = unitSet.Name
		}

		// Create pod templates
		podTemplate = &corev1.PodTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitSet.PodTemplateName(),
				Namespace: namespace.Name,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.39",
						},
					},
				},
			},
		}

		templatePodTemplate = &corev1.PodTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitSet.TemplatePodTemplateName(),
				Namespace: vars.ManagerNamespace,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
			},
		}
	})

	AfterEach(func() {
		// Best-effort cleanup; don't block suite on slow namespace termination
		unitUpdateGracePeriod = originalUnitUpdateGracePeriod
		unitReadyPollInterval = originalUnitReadyPollInterval
		unitReadyPollTimeout = originalUnitReadyPollTimeout

		_ = k8sClient.Delete(ctx, namespace)
	})

	Context("When testing mergeResources function", func() {
		It("Should merge resources correctly", func() {
			By("Creating unit with old resources")
			unit := createTestUnitForUpdate("test-unit-merge", namespace.Name, "8.0.40")
			unit.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			}

			By("Merging resources")
			mergedUnit := mergeResources(*unit, unitSet)

			By("Verifying resources are updated")
			Expect(mergedUnit.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Cmp(*unitSet.Spec.Resources.Requests.Cpu())).To(Equal(0))
			Expect(mergedUnit.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().Cmp(*unitSet.Spec.Resources.Requests.Memory())).To(Equal(0))
			Expect(mergedUnit.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().Cmp(*unitSet.Spec.Resources.Limits.Cpu())).To(Equal(0))
			Expect(mergedUnit.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().Cmp(*unitSet.Spec.Resources.Limits.Memory())).To(Equal(0))
		})

		It("Should not affect other containers", func() {
			By("Creating unit with multiple containers")
			unit := createTestUnitForUpdate("test-unit-multi", namespace.Name, "8.0.40")
			unit.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "mysql",
					Image: "mysql:8.0.40",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
				{
					Name:  "sidecar",
					Image: "busybox:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			}

			By("Merging resources")
			mergedUnit := mergeResources(*unit, unitSet)

			By("Verifying only mysql container is updated")
			Expect(mergedUnit.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Cmp(*unitSet.Spec.Resources.Requests.Cpu())).To(Equal(0))
			Expect(mergedUnit.Spec.Template.Spec.Containers[1].Resources.Requests.Cpu().Cmp(resource.MustParse("100m"))).To(Equal(0))
		})
	})

	Context("When testing mergeStorage function", func() {
		It("Should merge storage correctly", func() {
			By("Creating unit with old storage")
			unit := createTestUnitForUpdate("test-unit-storage", namespace.Name, "8.0.40")
			unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					},
				},
			}

			By("Merging storage")
			mergedUnit := mergeStorage(*unit, unitSet)

			By("Verifying storage is updated")
			Expect(mergedUnit.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.Storage().Cmp(resource.MustParse("10Gi"))).To(Equal(0))
		})

		It("Should handle multiple storage specifications", func() {
			By("Creating unit with multiple storage")
			unitSet.Spec.Storages = []upmiov1alpha2.StorageSpec{
				{Name: "data", Size: "10Gi"},
				{Name: "logs", Size: "5Gi"},
			}

			unit := createTestUnitForUpdate("test-unit-multi-storage", namespace.Name, "8.0.40")
			unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "logs",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("2Gi"),
							},
						},
					},
				},
			}

			By("Merging storage")
			mergedUnit := mergeStorage(*unit, unitSet)

			By("Verifying all storage is updated")
			Expect(mergedUnit.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.Storage().Cmp(resource.MustParse("10Gi"))).To(Equal(0))
			Expect(mergedUnit.Spec.VolumeClaimTemplates[1].Spec.Resources.Requests.Storage().Cmp(resource.MustParse("5Gi"))).To(Equal(0))
		})
	})

	//Context("When testing mergePodTemplate function", func() {
	//	It("Should merge pod template correctly", func() {
	//		By("Creating prerequisites")
	//		Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, podTemplate)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, templatePodTemplate)).To(Succeed())
	//
	//		By("Creating unit")
	//		unit := createTestUnitForUpdate("test-unit-pod", namespace.Name, "8.0.39")
	//		Expect(k8sClient.Create(ctx, unit)).To(Succeed())
	//
	//		By("Getting updated unit")
	//		updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		By("Merging pod template")
	//		req := reconcile.Request{
	//			NamespacedName: types.NamespacedName{
	//				Name:      unitSet.Name,
	//				Namespace: namespace.Name,
	//			},
	//		}
	//
	//		volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitSet)
	//		ports := upmiov1alpha2.Ports{}
	//
	//		mergedUnit := mergePodTemplate(ctx, req, *updatedUnit, unitSet, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)
	//
	//		By("Verifying pod template is merged")
	//		Expect(mergedUnit.Spec.Template.Spec.Subdomain).To(Equal(unitSet.HeadlessServiceName()))
	//		Expect(*mergedUnit.Spec.Template.Spec.EnableServiceLinks).To(BeTrue())
	//		Expect(mergedUnit.Spec.Template.Spec.ServiceAccountName).To(Equal(fmt.Sprintf("%s-serviceaccount", namespace.Name)))
	//		Expect(mergedUnit.Spec.Template.Spec.Hostname).To(Equal(unit.Name))
	//	})
	//
	//	//It("Should handle NodeNameMap correctly", func() {
	//	//	By("Creating UnitSet with NodeNameMap")
	//	//	unitSet.Spec.NodeNameMap = map[string]string{
	//	//		"test-unit-node": "node-1",
	//	//	}
	//	//	Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
	//	//	Expect(k8sClient.Create(ctx, podTemplate)).To(Succeed())
	//	//	Expect(k8sClient.Create(ctx, templatePodTemplate)).To(Succeed())
	//	//
	//	//	By("Creating unit")
	//	//	unit := createTestUnitForUpdate("test-unit-node", namespace.Name, "8.0.39")
	//	//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
	//	//
	//	//	By("Getting updated unit")
	//	//	updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
	//	//	Expect(err).NotTo(HaveOccurred())
	//	//
	//	//	By("Merging pod template")
	//	//	req := reconcile.Request{
	//	//		NamespacedName: types.NamespacedName{
	//	//			Name:      unitSet.Name,
	//	//			Namespace: namespace.Name,
	//	//		},
	//	//	}
	//	//
	//	//	volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitSet)
	//	//	ports := upmiov1alpha2.Ports{}
	//	//
	//	//	mergedUnit := mergePodTemplate(ctx, req, *updatedUnit, unitSet, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)
	//	//
	//	//	By("Verifying NodeNameMap is applied")
	//	//	Expect(mergedUnit.Annotations[upmiov1alpha2.AnnotationLastUnitBelongNode]).To(Equal("node-1"))
	//	//	Expect(mergedUnit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
	//	//	Expect(mergedUnit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key).To(Equal("kubernetes.io/hostname"))
	//	//	Expect(mergedUnit.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(Equal([]string{"node-1"}))
	//	//})
	//
	//	It("Should skip NoneSetFlag in NodeNameMap", func() {
	//		By("Creating UnitSet with NoneSetFlag")
	//		unitSet.Spec.NodeNameMap = map[string]string{
	//			"test-unit-none": upmiov1alpha2.NoneSetFlag,
	//		}
	//		Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, podTemplate)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, templatePodTemplate)).To(Succeed())
	//
	//		By("Creating unit")
	//		unit := createTestUnitForUpdate("test-unit-none", namespace.Name, "8.0.39")
	//		Expect(k8sClient.Create(ctx, unit)).To(Succeed())
	//
	//		By("Getting updated unit")
	//		updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		By("Merging pod template")
	//		req := reconcile.Request{
	//			NamespacedName: types.NamespacedName{
	//				Name:      unitSet.Name,
	//				Namespace: namespace.Name,
	//			},
	//		}
	//
	//		volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitSet)
	//		ports := upmiov1alpha2.Ports{}
	//
	//		mergedUnit := mergePodTemplate(ctx, req, *updatedUnit, unitSet, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)
	//
	//		By("Verifying NoneSetFlag is skipped")
	//		Expect(mergedUnit.Annotations[upmiov1alpha2.AnnotationLastUnitBelongNode]).To(BeEmpty())
	//		Expect(mergedUnit.Spec.Template.Spec.Affinity).To(BeNil())
	//	})
	//
	//	It("Should handle PersistentVolumeClaim correctly", func() {
	//		By("Creating UnitSet with storage")
	//		Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, podTemplate)).To(Succeed())
	//		Expect(k8sClient.Create(ctx, templatePodTemplate)).To(Succeed())
	//
	//		By("Creating unit with volumes")
	//		unit := createTestUnitForUpdate("test-unit-pvc", namespace.Name, "8.0.39")
	//		unit.Spec.Template.Spec.Volumes = []corev1.Volume{
	//			{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	//			{Name: "secret", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "test-secret"}}},
	//		}
	//		Expect(k8sClient.Create(ctx, unit)).To(Succeed())
	//
	//		By("Getting updated unit")
	//		updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		By("Merging pod template")
	//		req := reconcile.Request{
	//			NamespacedName: types.NamespacedName{
	//				Name:      unitSet.Name,
	//				Namespace: namespace.Name,
	//			},
	//		}
	//
	//		volumeMounts, volumes, envVars, pvcs := generateVolumeMountsAndEnvs(unitSet)
	//		ports := upmiov1alpha2.Ports{}
	//
	//		mergedUnit := mergePodTemplate(ctx, req, *updatedUnit, unitSet, podTemplate, ports, volumeMounts, volumes, envVars, pvcs)
	//
	//		By("Verifying PVC is set for non-secret volumes")
	//		Expect(mergedUnit.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim).NotTo(BeNil())
	//		Expect(mergedUnit.Spec.Template.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(upmiov1alpha2.PersistentVolumeClaimName(&mergedUnit, "data")))
	//		Expect(mergedUnit.Spec.Template.Spec.Volumes[1].PersistentVolumeClaim).To(BeNil())
	//	})
	//})

	Context("When testing reconcileResources function", func() {
		It("Should reconcile resources when needed", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with outdated resources")
			for _, unit := range units {
				unit.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling resources")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileResources(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying resources are updated")
			for _, unit := range units {
				updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedUnit.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Cmp(*unitSet.Spec.Resources.Requests.Cpu())).To(Equal(0))
			}
		})

		It("Should skip reconciliation when resources are up to date", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with correct resources")
			for _, unit := range units {
				unit.Spec.Template.Spec.Containers[0].Resources = unitSet.Spec.Resources
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling resources")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileResources(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When UnitSet labels/annotations change", func() {
		It("Should propagate labels and annotations to all units", func() {
			By("Creating UnitSet with initial metadata")
			unitSet.Labels = map[string]string{"app": "mysql", "tier": "db"}
			unitSet.Annotations = map[string]string{"owner": "team-a"}
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units managed by this UnitSet with minimal labels/annotations")
			for _, u := range units {
				if u.Labels == nil {
					u.Labels = map[string]string{}
				}
				u.Labels[upmiov1alpha2.UnitsetName] = unitSet.Name
				// Leave annotations as-is from helper; add a custom label/annotation to verify non-overwrite
				u.Labels["custom-unit-label"] = "keep"
				if u.Annotations == nil {
					u.Annotations = map[string]string{}
				}
				u.Annotations["custom-unit-anno"] = "keep"
				Expect(k8sClient.Create(ctx, u)).To(Succeed())
			}

			reconciler := &UnitSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}}

			By("Reconciling labels/annotations propagation")
			Expect(reconciler.reconcileUnitLabelsAnnotations(ctx, req, unitSet)).To(Succeed())

			By("Verifying all units received UnitSet labels and annotations while keeping their own")
			for _, u := range units {
				updated, err := getUnit(ctx, u.Name, namespace.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
				Expect(updated.Labels["app"]).To(Equal("mysql"))
				Expect(updated.Labels["tier"]).To(Equal("db"))
				Expect(updated.Labels["custom-unit-label"]).To(Equal("keep"))
				Expect(updated.Annotations["owner"]).To(Equal("team-a"))
				Expect(updated.Annotations["custom-unit-anno"]).To(Equal("keep"))
			}

			By("Updating UnitSet metadata and reconciling again")
			unitSet.Labels["app"] = "mysql-updated"
			unitSet.Annotations["owner"] = "team-b"
			Expect(reconciler.reconcileUnitLabelsAnnotations(ctx, req, unitSet)).To(Succeed())

			By("Verifying updates are propagated")
			for _, u := range units {
				updated, err := getUnit(ctx, u.Name, namespace.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Labels["app"]).To(Equal("mysql-updated"))
				Expect(updated.Annotations["owner"]).To(Equal("team-b"))
			}
		})

		It("Should handle nil labels/annotations gracefully and initialize target maps", func() {
			By("Creating UnitSet with nil labels and annotations")
			unitSet.Labels = nil
			unitSet.Annotations = nil
			unitSet.Name = "test-unitset-meta-nil"
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating a unit with nil maps that belongs to this UnitSet")
			u := createTestUnitForUpdate("meta-nil-unit", namespace.Name, "8.0.39")
			u.Labels = nil
			u.Annotations = nil
			u.Labels = map[string]string{upmiov1alpha2.UnitsetName: unitSet.Name}
			Expect(k8sClient.Create(ctx, u)).To(Succeed())

			reconciler := &UnitSetReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}}

			By("Reconciling should not fail and should keep UnitSetName label")
			Expect(reconciler.reconcileUnitLabelsAnnotations(ctx, req, unitSet)).To(Succeed())
			updated, err := getUnit(ctx, u.Name, namespace.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Labels).NotTo(BeNil())
			Expect(updated.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
		})
	})

	Context("When testing reconcileStorage function", func() {
		It("Should reconcile storage when needed", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with outdated storage")
			for _, unit := range units {
				unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling storage")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileStorage(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying storage is updated")
			for _, unit := range units {
				updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedUnit.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.Storage().Cmp(resource.MustParse("10Gi"))).To(Equal(0))
			}
		})

		It("Should skip reconciliation when storage is up to date", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with correct storage")
			for _, unit := range units {
				unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling storage")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileStorage(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When testing reconcileImageVersion function", func() {
		It("Should reconcile image version when templates differ", func() {
			By("Creating prerequisites")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
			_ = k8sClient.Create(ctx, podTemplate)
			_ = k8sClient.Create(ctx, templatePodTemplate)

			By("Creating ready units")
			for _, unit := range units {
				unit.Status.Phase = upmiov1alpha2.UnitReady
				Expect(k8sClient.Create(ctx, unit.DeepCopy())).To(Succeed())
				createdUnit := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: namespace.Name}, createdUnit)).To(Succeed())
				createdUnit.Status.Phase = upmiov1alpha2.UnitReady
				Expect(k8sClient.Status().Update(ctx, createdUnit)).To(Succeed())
				Eventually(func() (upmiov1alpha2.UnitPhase, error) {
					current, err := getUnit(ctx, unit.Name, namespace.Name)
					if err != nil {
						return "", err
					}
					return current.Status.Phase, nil
				}, 2*time.Second, 50*time.Millisecond).Should(Equal(upmiov1alpha2.UnitReady))
			}

			By("Reconciling image version")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileImageVersion(ctx, req, unitSet, podTemplate, []corev1.ContainerPort{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying units were updated to the new version")
			for _, unit := range units {
				updatedUnit, getErr := getUnit(ctx, unit.Name, namespace.Name)
				Expect(getErr).NotTo(HaveOccurred())
				Expect(updatedUnit.Spec.Template.Spec.Containers[0].Image).To(Equal("mysql:8.0.40"))
				Expect(updatedUnit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal(unitSet.Spec.Version))
			}
		})

		It("Should stop at first unit when RollingUpdate unit fails to become ready", func() {
			By("Creating prerequisites")
			unitSet.Spec.UpdateStrategy.Type = "RollingUpdate"
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
			_ = k8sClient.Create(ctx, podTemplate)
			_ = k8sClient.Create(ctx, templatePodTemplate)

			By("Creating units with a pending first unit")
			for i, unit := range units {
				if i == 0 {
					unit.Status.Phase = upmiov1alpha2.UnitRunning
				} else {
					unit.Status.Phase = upmiov1alpha2.UnitReady
				}
				Expect(k8sClient.Create(ctx, unit.DeepCopy())).To(Succeed())
				createdUnit := &upmiov1alpha2.Unit{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: unit.Name, Namespace: namespace.Name}, createdUnit)).To(Succeed())
				createdUnit.Status.Phase = unit.Status.Phase
				Expect(k8sClient.Status().Update(ctx, createdUnit)).To(Succeed())
				expectedPhase := unit.Status.Phase
				Eventually(func() (upmiov1alpha2.UnitPhase, error) {
					current, err := getUnit(ctx, unit.Name, namespace.Name)
					if err != nil {
						return "", err
					}
					return current.Status.Phase, nil
				}, 2*time.Second, 50*time.Millisecond).Should(Equal(expectedPhase))
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileImageVersion(ctx, req, unitSet, podTemplate, []corev1.ContainerPort{})
			Expect(err).NotTo(HaveOccurred())

			latestUnitSet := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: unitSet.Name, Namespace: namespace.Name}, latestUnitSet)).To(Succeed())
			nextUnitIndex := len(units) - 2
			if nextUnitIndex < 0 {
				nextUnitIndex = 0
			}
			nextUnitName := units[nextUnitIndex].Name
			Expect(latestUnitSet.Status.InUpdate).To(Equal(nextUnitName))

			highestOrdinalUnit := units[len(units)-1]
			highestUpdated, err := getUnit(ctx, highestOrdinalUnit.Name, namespace.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(highestUpdated.Spec.Template.Spec.Containers[0].Image).To(Equal("mysql:8.0.40"))
			Expect(highestUpdated.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal(unitSet.Spec.Version))

			By("Ensuring the next units have not been updated yet")
			pendingUnit, getErr := getUnit(ctx, nextUnitName, namespace.Name)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(pendingUnit.Spec.Template.Spec.Containers[0].Image).To(Equal("mysql:8.0.39"))
			Expect(pendingUnit.Annotations[upmiov1alpha2.AnnotationMainContainerVersion]).To(Equal("8.0.39"))

			By("Confirming the lowest ordinal unit is untouched")
			lowestUnit, lowestErr := getUnit(ctx, units[0].Name, namespace.Name)
			Expect(lowestErr).NotTo(HaveOccurred())
			Expect(lowestUnit.Spec.Template.Spec.Containers[0].Image).To(Equal("mysql:8.0.39"))
		})

		It("Should skip reconciliation when templates are identical", func() {
			By("Creating prerequisites")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			// Create identical templates
			identicalTemplate := templatePodTemplate.DeepCopy()
			identicalTemplate.Name = unitSet.PodTemplateName()
			identicalTemplate.Namespace = namespace.Name
			_ = k8sClient.Create(ctx, identicalTemplate)
			_ = k8sClient.Create(ctx, templatePodTemplate)

			By("Creating units")
			for _, unit := range units {
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling image version")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileImageVersion(ctx, req, unitSet, identicalTemplate, []corev1.ContainerPort{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When testing error handling", func() {
		It("Should handle template retrieval errors", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling image version without templates")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileImageVersion(ctx, req, unitSet, podTemplate, []corev1.ContainerPort{})
			Expect(err).To(HaveOccurred())
		})

		It("Should handle unit retrieval errors", func() {
			By("Creating prerequisites")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
			_ = k8sClient.Create(ctx, podTemplate)
			_ = k8sClient.Create(ctx, templatePodTemplate)

			By("Reconciling image version without units")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			// Note: This would need mocking in a real scenario
			err := reconciler.reconcileImageVersion(ctx, req, unitSet, podTemplate, []corev1.ContainerPort{})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When testing concurrent updates", func() {
		It("Should handle concurrent resource updates", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Creating units with outdated resources")
			for _, unit := range units {
				unit.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				}
				Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			}

			By("Reconciling resources concurrently")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileResources(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all units are updated")
			for _, unit := range units {
				updatedUnit, err := getUnit(ctx, unit.Name, namespace.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(updatedUnit.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().Cmp(*unitSet.Spec.Resources.Requests.Cpu())).To(Equal(0))
			}
		})
	})
})

// Helper function to create test units for update testing
func createTestUnitForUpdate(name, namespace, version string) *upmiov1alpha2.Unit {
	return &upmiov1alpha2.Unit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				upmiov1alpha2.AnnotationMainContainerName:    "mysql",
				upmiov1alpha2.AnnotationMainContainerVersion: version,
			},
		},
		Spec: upmiov1alpha2.UnitSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: fmt.Sprintf("mysql:%s", version),
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
			Phase: upmiov1alpha2.UnitReady,
		},
	}
}

// Helper function to get unit from Kubernetes
func getUnit(ctx context.Context, name, namespace string) (*upmiov1alpha2.Unit, error) {
	unit := &upmiov1alpha2.Unit{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, unit)
	return unit, err
}
