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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func setPodStatus(ctx context.Context, name, ns string, status corev1.PodStatus) {
	p := &corev1.Pod{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, p)).To(Succeed())
	p.Status = status
	Expect(k8sClient.Status().Update(ctx, p)).To(Succeed())
}

var _ = Describe("Unit Status Reconciliation", func() {
	var (
		ctx        context.Context
		unit       *upmiov1alpha2.Unit
		namespace  *corev1.Namespace
		testNsName = "test-unit-status-namespace"
		reconciler *UnitReconciler
		pod        *corev1.Pod
		node       *corev1.Node
		pvc        *corev1.PersistentVolumeClaim
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace with unique name per spec
		testNsName = fmt.Sprintf("test-unit-status-namespace-%d", time.Now().UnixNano())
		namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName}}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic Unit for testing
		unit = &upmiov1alpha2.Unit{
			ObjectMeta: metav1.ObjectMeta{Name: "test-unit-status", Namespace: namespace.Name},
			Spec: upmiov1alpha2.UnitSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name: "mysql", Image: "mysql:8.0.40",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("512Mi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1000m"), corev1.ResourceMemory: resource.MustParse("1Gi")}},
				}}}},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "data"}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}}}}},
			},
		}

		// Create reconciler
		reconciler = &UnitReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

		// Create test node with unique name
		node = &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("test-node-%d", time.Now().UnixNano())}, Spec: corev1.NodeSpec{PodCIDR: "10.244.0.0/24"}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}}

		// Create test PVC
		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "test-unit-status-data", Namespace: namespace.Name}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}}, VolumeName: "pv-1"}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound, Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")}}}
	})

	AfterEach(func() { _ = k8sClient.Delete(ctx, namespace) })

	Context("When reconciling unit status with no resources", func() {
		It("Should keep status unchanged when resources are absent", func() {
			By("Creating unit")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			By("Reconciling unit status with no resources")
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}}
			expectErr := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
		})
	})

	Context("When reconciling unit status with running pod", func() {
		It("Should update status correctly for ready pod", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())
			pod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unit.Name, Namespace: namespace.Name}, Spec: corev1.PodSpec{NodeName: node.Name, Containers: []corev1.Container{{Name: "mysql", Image: "mysql:8.0.40"}, {Name: "unit-agent", Image: "unit-agent:latest"}}}}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			setPodStatus(ctx, unit.Name, namespace.Name, corev1.PodStatus{Phase: corev1.PodRunning, HostIP: "192.168.1.1", PodIPs: []corev1.PodIP{{IP: "10.244.0.1"}}, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}, ContainerStatuses: []corev1.ContainerStatus{{Name: "unit-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}})
			By("Reconciling unit status")
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}}
			expectErr := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
			By("Verifying status is updated correctly")
			Expect(unit.Status.Phase).To(Equal(upmiov1alpha2.UnitReady))
			Expect(unit.Status.NodeName).To(Equal(node.Name))
		})

		It("Should handle running but not ready pod", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())
			pod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unit.Name, Namespace: namespace.Name}, Spec: corev1.PodSpec{NodeName: node.Name, Containers: []corev1.Container{{Name: "mysql", Image: "mysql:8.0.40"}, {Name: "unit-agent", Image: "unit-agent:latest"}}}}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			setPodStatus(ctx, unit.Name, namespace.Name, corev1.PodStatus{Phase: corev1.PodRunning, HostIP: "192.168.1.1", PodIPs: []corev1.PodIP{{IP: "10.244.0.1"}}, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}, ContainerStatuses: []corev1.ContainerStatus{{Name: "unit-agent", Ready: false, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}})
			By("Reconciling unit status")
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}}
			expectErr := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
			By("Verifying status shows running but not ready")
			Expect(unit.Status.Phase).To(Equal(upmiov1alpha2.UnitRunning))
		})

		It("Should handle pod in other phases", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: node.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase:  corev1.PodPending,
					HostIP: "192.168.1.1",
					PodIPs: []corev1.PodIP{{IP: "10.244.0.1"}},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status reflects pod phase")
			Expect(unit.Status.Phase).To(Equal(upmiov1alpha2.UnitPhase(corev1.PodPending)))
			Expect(unit.Status.NodeReady).To(Equal("True"))
			Expect(unit.Status.NodeName).To(Equal(node.Name))
		})
	})

	Context("When handling node status", func() {
		It("Should set NodeReady to True for ready node", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			readyNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("ready-node-%d", time.Now().UnixNano()),
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, readyNode)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: readyNode.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying node status is correct")
			Expect(unit.Status.NodeReady).To(Equal("True"))
			Expect(unit.Status.NodeName).To(Equal(readyNode.Name))
		})

		It("Should set NodeReady to False for not ready node", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			notReadyNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("not-ready-node-%d", time.Now().UnixNano()),
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, notReadyNode)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: notReadyNode.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying node status is correct")
			Expect(unit.Status.NodeReady).To(Equal("False"))
			Expect(unit.Status.NodeName).To(Equal(notReadyNode.Name))
		})

		It("Should handle node without ready condition", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			noConditionNode := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("no-condition-node-%d", time.Now().UnixNano()),
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeMemoryPressure,
							Status: corev1.ConditionFalse,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, noConditionNode)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: noConditionNode.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying node status defaults to False")
			Expect(unit.Status.NodeReady).To(Equal("False"))
			Expect(unit.Status.NodeName).To(Equal(noConditionNode.Name))
		})
	})

	Context("When handling PVC status", func() {
		It("Should update PVC info correctly", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
			pod = &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unit.Name, Namespace: namespace.Name}, Spec: corev1.PodSpec{NodeName: node.Name, Containers: []corev1.Container{{Name: "mysql", Image: "mysql:8.0.40"}}}}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			setPodStatus(ctx, unit.Name, namespace.Name, corev1.PodStatus{Phase: corev1.PodRunning})
			By("Reconciling unit status")
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}}
			expectErr := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
			By("Verifying PVC info is updated")
			Expect(unit.Status.PersistentVolumeClaim).To(HaveLen(1))
			// compare capacity by value
			Expect(unit.Status.PersistentVolumeClaim[0].Capacity.Storage.Cmp(*pvc.Status.Capacity.Storage())).To(Equal(0))
		})

		It("Should handle multiple PVCs", func() {
			By("Creating unit with multiple volume templates")
			unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "data"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "logs"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("5Gi"),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			By("Creating multiple PVCs")
			dataPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unit-status-data",
					Namespace: namespace.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
					VolumeName: "pv-data",
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Phase: corev1.ClaimBound,
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10Gi"),
					},
				},
			}
			logsPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unit-status-logs",
					Namespace: namespace.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
					VolumeName: "pv-logs",
				},
				Status: corev1.PersistentVolumeClaimStatus{
					Phase: corev1.ClaimBound,
					Capacity: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("5Gi"),
					},
				},
			}
			Expect(k8sClient.Create(ctx, dataPVC)).To(Succeed())
			Expect(k8sClient.Create(ctx, logsPVC)).To(Succeed())

			Expect(k8sClient.Create(ctx, node)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: node.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying multiple PVCs are handled")
			Expect(unit.Status.PersistentVolumeClaim).To(HaveLen(2))
			pvcNames := []string{
				unit.Status.PersistentVolumeClaim[0].Name,
				unit.Status.PersistentVolumeClaim[1].Name,
			}
			Expect(pvcNames).To(ContainElement(dataPVC.Name))
			Expect(pvcNames).To(ContainElement(logsPVC.Name))
		})

		It("Should handle unit with no volume templates", func() {
			By("Creating unit with no volume templates")
			unit.Spec.VolumeClaimTemplates = nil
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: node.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no PVC info when no templates")
			Expect(unit.Status.PersistentVolumeClaim).To(BeNil())
		})
	})

	Context("When testing unitManagedResources", func() {
		It("Should not error when pod is absent", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			req := reconcile.Request{NamespacedName: types.NamespacedName{Name: unit.Name, Namespace: namespace.Name}}
			retrievedPod, pvcs, retrievedNode, err := reconciler.unitManagedResources(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPod).To(BeNil())
			Expect(pvcs).To(BeNil())
			Expect(retrievedNode).To(BeNil())
		})

		It("Should not error when node retrieval fails (IsNotFound treated as nil)", func() {
			By("Creating unit and pod")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: fmt.Sprintf("non-existent-node-%d", time.Now().UnixNano()),
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Calling unitManagedResources with non-existent node")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			retrievedPod, pvcs, node, err := reconciler.unitManagedResources(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPod).NotTo(BeNil())
			Expect(pvcs).To(BeNil())
			Expect(node).To(BeNil())
		})

		It("Should not error when PVC retrieval fails (missing PVCs ignored)", func() {
			By("Creating unit and pod")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: node.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Calling unitManagedResources with non-existent PVC")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			retrievedPod, pvcs, retrievedNode, err := reconciler.unitManagedResources(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPod).NotTo(BeNil())
			Expect(pvcs).To(BeNil())
			Expect(retrievedNode).NotTo(BeNil())
		})

		It("Should return all resources successfully", func() {
			By("Creating all resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, node)).To(Succeed())
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: node.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Calling unitManagedResources")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			retrievedPod, pvcs, retrievedNode, err := reconciler.unitManagedResources(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPod).NotTo(BeNil())
			Expect(pvcs).To(HaveLen(1))
			Expect(retrievedNode).NotTo(BeNil())
			Expect(pvcs[0].Name).To(Equal(pvc.Name))
		})

		It("Should handle pod without node name without error", func() {
			By("Creating unit and pod without node")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			By("Calling unitManagedResources with no node name")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			retrievedPod, pvcs, retrievedNode, err := reconciler.unitManagedResources(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedPod).NotTo(BeNil())
			Expect(pvcs).To(BeNil())
			Expect(retrievedNode).To(BeNil())
		})
	})

	Context("When testing status update optimization", func() {
		It("Should skip update when status hasn't changed", func() {
			By("Creating resources")
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			// create a ready node
			readyNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ready-node-%d", time.Now().UnixNano())}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}}
			Expect(k8sClient.Create(ctx, readyNode)).To(Succeed())

			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
				Spec: corev1.PodSpec{
					NodeName: readyNode.Name,
					Containers: []corev1.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.40",
						},
						{
							Name:  "unit-agent",
							Image: "unit-agent:latest",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())
			// set pod ready status after creation
			setPodStatus(ctx, unit.Name, namespace.Name, corev1.PodStatus{
				Phase:             corev1.PodRunning,
				HostIP:            "192.168.1.1",
				PodIPs:            []corev1.PodIP{{IP: "10.244.0.1"}},
				Conditions:        []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
				ContainerStatuses: []corev1.ContainerStatus{{Name: "unit-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
			})

			By("Setting initial status")
			unit.Status.Phase = upmiov1alpha2.UnitReady
			unit.Status.NodeReady = "True"
			unit.Status.NodeName = readyNode.Name
			unit.Status.HostIP = "192.168.1.1"
			unit.Status.PodIPs = []corev1.PodIP{{IP: "10.244.0.1"}}

			By("Reconciling unit status")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unit.Name,
					Namespace: namespace.Name,
				},
			}

			// Create a mock status writer to check if Update is called
			origUnit := unit.DeepCopy()
			err := reconciler.reconcileUnitStatus(ctx, req, unit)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying status is the same (no update needed)")
			Expect(unit.Status.Phase).To(Equal(origUnit.Status.Phase))
			Expect(unit.Status.NodeReady).To(Equal(origUnit.Status.NodeReady))
			Expect(unit.Status.NodeName).To(Equal(origUnit.Status.NodeName))
			Expect(unit.Status.HostIP).To(Equal(origUnit.Status.HostIP))
			Expect(unit.Status.PodIPs).To(Equal(origUnit.Status.PodIPs))
		})
	})
})
