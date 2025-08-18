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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("UnitDelete Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		unit       *upmiov1alpha2.Unit
		unitName   string
		//pod  *corev1.Pod
		//pvc  *corev1.PersistentVolumeClaim
		//pv   *corev1.PersistentVolume
		req ctrl.Request
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create fake client
		//k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

		// Create reconciler with fake client
		reconciler = &UnitReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}

		// unique name per test and cleanup leftovers
		suffix := time.Now().UnixNano()
		unitName = fmt.Sprintf("test-unit-%d", suffix)
		_ = k8sClient.Delete(ctx, &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: unitName + "-data", Namespace: "default"}})

		// Create test request
		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: unitName, Namespace: "default"}}

		// Create test unit
		unit = &upmiov1alpha2.Unit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
				Finalizers: []string{
					upmiov1alpha2.FinalizerPodDelete,
					upmiov1alpha2.FinalizerPvcDelete,
				},
			},
			Spec: upmiov1alpha2.UnitSpec{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{},
							},
						},
					},
				},
			},
		}

		// Create test pod
		//pod = &corev1.Pod{
		//	ObjectMeta: metav1.ObjectMeta{
		//		Name:      "test-unit",
		//		Namespace: "default",
		//	},
		//	Spec: corev1.PodSpec{
		//		Containers: []corev1.Container{
		//			{
		//				Name:  "main",
		//				Image: "nginx:latest",
		//			},
		//		},
		//	},
		//}
		//
		//// Create test PVC
		//pvc = &corev1.PersistentVolumeClaim{
		//	ObjectMeta: metav1.ObjectMeta{
		//		Name:      "test-unit-data",
		//		Namespace: "default",
		//	},
		//	Spec: corev1.PersistentVolumeClaimSpec{
		//		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		//		Resources: corev1.VolumeResourceRequirements{
		//			Requests: corev1.ResourceList{},
		//		},
		//	},
		//}
		//
		//// Create test PV
		//pv = &corev1.PersistentVolume{
		//	ObjectMeta: metav1.ObjectMeta{
		//		Name: "test-pv",
		//	},
		//	Spec: corev1.PersistentVolumeSpec{
		//		ClaimRef: &corev1.ObjectReference{
		//			Name:      "test-unit-data",
		//			Namespace: "default",
		//		},
		//	},
		//}
	})

	Context("deleteResources", func() {
		It("should call deletePodWithFinalizer when finalizer is pod-delete", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			err := reconciler.deleteResources(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should call deletePVCWithFinalizer when finalizer is pvc-delete", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			err := reconciler.deleteResources(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return nil when finalizer is unknown", func() {
			err := reconciler.deleteResources(ctx, req, unit, "unknown-finalizer")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("deletePodWithFinalizer", func() {
		It("should remove finalizer when pod is not found", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed
			updatedUnit := &upmiov1alpha2.Unit{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, updatedUnit)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(updatedUnit, upmiov1alpha2.FinalizerPodDelete)).To(BeFalse())
		})

		//It("should delete pod normally when force delete is not set", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the wait function to return immediately
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		//It("should force delete pod when force delete annotation is set", func() {
		//	unit.Annotations = map[string]string{
		//		upmiov1alpha2.AnnotationForceDelete: "true",
		//	}
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the wait function to return immediately
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		//It("should return error when pod deletion fails", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the Delete function to return an error
		//	originalDelete := k8sClient.Delete
		//	k8sClient.Delete = func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
		//		return fmt.Errorf("delete failed")
		//	}
		//	defer func() { k8sClient.Delete = originalDelete }()
		//
		//	err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error deleting pod"))
		//})

		//It("should return error when waiting for pod deletion times out", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pod)).To(Succeed())
		//
		//	// Mock the wait function to return timeout
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return wait.ErrWaitTimeout
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error waiting for pod deleted"))
		//})
	})

	Context("deletePVCWithFinalizer", func() {
		It("should handle when unit has no volume claim templates", func() {
			unit.Spec.VolumeClaimTemplates = nil
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
			Expect(err).NotTo(HaveOccurred())

			// Verify finalizer was removed
			updatedUnit := &upmiov1alpha2.Unit{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, updatedUnit)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(updatedUnit, upmiov1alpha2.FinalizerPvcDelete)).To(BeFalse())
		})

		//It("should delete PVC normally when force delete is not set", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//
		//	// Mock the wait function to return immediately
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		//It("should force delete PVC and PV when force delete annotation is set", func() {
		//	unit.Annotations = map[string]string{
		//		upmiov1alpha2.AnnotationForceDelete: "true",
		//	}
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pv)).To(Succeed())
		//
		//	// Mock the wait function to return immediately
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).NotTo(HaveOccurred())
		//})

		//It("should return error when PVC deletion fails", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//
		//	// Mock the Delete function to return an error
		//	originalDelete := k8sClient.Delete
		//	k8sClient.Delete = func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
		//		if _, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		//			return fmt.Errorf("pvc delete failed")
		//		}
		//		return originalDelete(ctx, obj, opts...)
		//	}
		//	defer func() { k8sClient.Delete = originalDelete }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error deleting pvc"))
		//})

		//It("should return error when PV deletion fails in force delete mode", func() {
		//	unit.Annotations = map[string]string{
		//		upmiov1alpha2.AnnotationForceDelete: "true",
		//	}
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pv)).To(Succeed())
		//
		//	// Mock the Delete function to return an error for PV
		//	originalDelete := k8sClient.Delete
		//	k8sClient.Delete = func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
		//		if _, ok := obj.(*corev1.PersistentVolume); ok {
		//			return fmt.Errorf("pv delete failed")
		//		}
		//		return originalDelete(ctx, obj, opts...)
		//	}
		//	defer func() { k8sClient.Delete = originalDelete }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error force deleting pv"))
		//})

		//It("should return error when waiting for PVC deletion times out", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//
		//	// Mock the wait function to return timeout
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return wait.ErrWaitTimeout
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error waiting for pvc deleted"))
		//})

		//It("should handle concurrent PVC deletion", func() {
		//	// Add multiple volume claim templates
		//	unit.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		//		{
		//			ObjectMeta: metav1.ObjectMeta{
		//				Name:      "data",
		//			},
		//		},
		//		{
		//			ObjectMeta: metav1.ObjectMeta{
		//				Name:      "logs",
		//			},
		//		},
		//	}
		//
		//	// Create multiple PVCs
		//	pvc2 := &corev1.PersistentVolumeClaim{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name:      "test-unit-logs",
		//			Namespace: "default",
		//		},
		//		Spec: corev1.PersistentVolumeClaimSpec{
		//			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		//			Resources: corev1.VolumeResourceRequirements{
		//				Requests: corev1.ResourceList{},
		//			},
		//		},
		//	}
		//
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
		//	Expect(k8sClient.Create(ctx, pvc2)).To(Succeed())
		//
		//	// Mock the wait function to return immediately
		//	originalPollUntilContextTimeout := wait.PollUntilContextTimeout
		//	wait.PollUntilContextTimeout = func(ctx context.Context, interval, timeout time.Duration, immediate bool, condition wait.ConditionWithContextFunc) error {
		//		return nil // Simulate successful deletion
		//	}
		//	defer func() { wait.PollUntilContextTimeout = originalPollUntilContextTimeout }()
		//
		//	err := reconciler.deletePVCWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPvcDelete)
		//	Expect(err).NotTo(HaveOccurred())
		//})
	})

	Context("Finalizer removal", func() {
		It("should remove finalizer when update succeeds", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())

			// Remove finalizer manually to test the update logic
			controllerutil.RemoveFinalizer(unit, upmiov1alpha2.FinalizerPodDelete)
			Expect(k8sClient.Update(ctx, unit)).To(Succeed())

			// Verify finalizer was removed
			updatedUnit := &upmiov1alpha2.Unit{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, updatedUnit)).To(Succeed())
			Expect(controllerutil.ContainsFinalizer(updatedUnit, upmiov1alpha2.FinalizerPodDelete)).To(BeFalse())
		})

		//It("should return error when finalizer removal fails", func() {
		//	Expect(k8sClient.Create(ctx, unit)).To(Succeed())
		//
		//	// Mock the Update function to return an error
		//	originalUpdate := k8sClient.Update
		//	k8sClient.Update = func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
		//		return fmt.Errorf("update failed")
		//	}
		//	defer func() { k8sClient.Update = originalUpdate }()
		//
		//	err := reconciler.deletePodWithFinalizer(ctx, req, unit, upmiov1alpha2.FinalizerPodDelete)
		//	Expect(err).To(HaveOccurred())
		//	Expect(err.Error()).To(ContainSubstring("error removing finalizer"))
		//})
	})
})

func TestUnitDelete(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UnitDelete Suite")
}
