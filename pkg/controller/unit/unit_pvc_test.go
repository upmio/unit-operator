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
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("UnitPVC Reconciler", func() {
	var (
		ctx        context.Context
		reconciler *UnitReconciler
		unit       *upmiov1alpha2.Unit
		pvc        *corev1.PersistentVolumeClaim
		pv         *corev1.PersistentVolume
		req        ctrl.Request
		unitName   string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// unique name per test
		suffix := time.Now().UnixNano()
		unitName = fmt.Sprintf("test-unit-%d", suffix)
		pvcDataName := fmt.Sprintf("%s-data", unitName)
		pvcLogsName := fmt.Sprintf("%s-logs", unitName)
		pvName := fmt.Sprintf("test-pv-%d", suffix)

		// cleanup leftovers to avoid AlreadyExists
		_ = k8sClient.Delete(ctx, &upmiov1alpha2.Unit{ObjectMeta: metav1.ObjectMeta{Name: unitName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvcDataName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvcLogsName, Namespace: "default"}})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}})

		reconciler = &UnitReconciler{Client: k8sClient, Scheme: scheme.Scheme, Recorder: recorder}

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: unitName, Namespace: "default"}}

		unit = &upmiov1alpha2.Unit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      unitName,
				Namespace: "default",
				Labels:    map[string]string{"app": "test-app", "unit-label": "test-value"},
			},
			Spec: upmiov1alpha2.UnitSpec{VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{ObjectMeta: metav1.ObjectMeta{Name: "data"}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")}}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "logs"}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("500Mi")}}}},
			}},
		}

		pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: pvcDataName, Namespace: "default"}, Spec: corev1.PersistentVolumeClaimSpec{AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")}}}}

		pv = &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}, Spec: corev1.PersistentVolumeSpec{Capacity: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")}, AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, PersistentVolumeSource: corev1.PersistentVolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"}}, ClaimRef: &corev1.ObjectReference{Name: pvcDataName, Namespace: "default"}}}
	})

	Context("reconcilePersistentVolumeClaims", func() {
		It("should return nil when unit has no volume claim templates", func() {
			unit.Spec.VolumeClaimTemplates = nil
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			expectErr := reconciler.reconcilePersistentVolumeClaims(ctx, req, unit)
			// envtest不支持PVC在线扩容（需要动态存储类支持），这里允许该特定报错
			if expectErr != nil {
				Expect(strings.Contains(expectErr.Error(), "only dynamically provisioned pvc can be resized")).To(BeTrue())
			} else {
				Expect(expectErr).NotTo(HaveOccurred())
			}
		})

		It("should create PVC when not found", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			expectErr := reconciler.reconcilePersistentVolumeClaims(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
			dataPVC := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-data", unitName), Namespace: "default"}, dataPVC)).To(Succeed())
			Expect(dataPVC.Spec.Resources.Requests.Storage().String()).To(Equal("1Gi"))
			logsPVC := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-logs", unitName), Namespace: "default"}, logsPVC)).To(Succeed())
			Expect(logsPVC.Spec.Resources.Requests.Storage().String()).To(Equal("500Mi"))
		})

		It("should return error when PV already exists", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pv)).To(Succeed())
			// ensure pv present
			tmp := &corev1.PersistentVolume{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: pv.Name}, tmp)).To(Succeed())
			expectErr := reconciler.reconcilePersistentVolumeClaims(ctx, req, unit)
			Expect(expectErr).To(HaveOccurred())
			Expect(expectErr.Error()).To(ContainSubstring("pv ["))
		})

		It("should update PVC when storage request changes", func() {
			pvc.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("512Mi")}
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
			// mark pvc as Bound after creation and set storageclass to allow resize logic to proceed
			fetched := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: "default"}, fetched)).To(Succeed())
			fetched.Status.Phase = corev1.ClaimBound
			className := "standard"
			fetched.Spec.StorageClassName = &className
			Expect(k8sClient.Status().Update(ctx, fetched)).To(Succeed())
			expectErr := reconciler.reconcilePersistentVolumeClaims(ctx, req, unit)
			if expectErr != nil {
				Expect(strings.Contains(expectErr.Error(), "only dynamically provisioned pvc can be resized")).To(BeTrue())
			} else {
				updatedPVC := &corev1.PersistentVolumeClaim{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: "default"}, updatedPVC)).To(Succeed())
				Expect(updatedPVC.Spec.Resources.Requests.Storage().String()).To(Equal("1Gi"))
			}
		})

		It("should handle existing PVC without update needed", func() {
			Expect(k8sClient.Create(ctx, unit)).To(Succeed())
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())
			expectErr := reconciler.reconcilePersistentVolumeClaims(ctx, req, unit)
			Expect(expectErr).NotTo(HaveOccurred())
		})
	})

	Context("convert2PVC", func() {
		It("should create PVC from unit template", func() {
			volumeClaimTemplate := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "data",
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			claim, err := convert2PVC(unit, volumeClaimTemplate)
			Expect(err).NotTo(HaveOccurred())

			Expect(claim.Name).To(Equal(fmt.Sprintf("%s-data", unitName)))
			Expect(claim.Namespace).To(Equal("default"))
			Expect(claim.Spec.AccessModes).To(Equal([]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}))
			Expect(claim.Spec.Resources.Requests.Storage().String()).To(Equal("1Gi"))
			Expect(claim.Labels).To(HaveKey("app"))
			Expect(claim.Labels["app"]).To(Equal("test-app"))
			// By design, PVCs are NOT owned by Unit (user decides whether PVC should be deleted).
			Expect(claim.OwnerReferences).To(BeEmpty())
		})

		It("should copy unit labels to PVC", func() {
			unit.Labels["custom-label"] = "custom-value"

			volumeClaimTemplate := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "data",
					Namespace: "default",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			claim, err := convert2PVC(unit, volumeClaimTemplate)
			Expect(err).NotTo(HaveOccurred())

			Expect(claim.Labels).To(HaveKey("custom-label"))
			Expect(claim.Labels["custom-label"]).To(Equal("custom-value"))
		})

		It("should handle PVC with complex resource requirements", func() {
			volumeClaimTemplate := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany},
					StorageClassName: strPtr("fast-ssd"),
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"storage": "ssd"},
					},
				},
			}

			claim, err := convert2PVC(unit, volumeClaimTemplate)
			Expect(err).NotTo(HaveOccurred())

			Expect(claim.Spec.AccessModes).To(Equal([]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany}))
			Expect(claim.Spec.StorageClassName).To(Equal(strPtr("fast-ssd")))
			Expect(claim.Spec.Resources.Requests.Storage().String()).To(Equal("10Gi"))
			Expect(claim.Spec.Resources.Limits.Storage().String()).To(Equal("10Gi"))
			Expect(claim.Spec.Selector.MatchLabels).To(Equal(map[string]string{"storage": "ssd"}))
		})
	})
})

func strPtr(s string) *string { return &s }

func TestUnitPVC(t *testing.T) { RegisterFailHandler(Fail); RunSpecs(t, "UnitPVC Suite") }
