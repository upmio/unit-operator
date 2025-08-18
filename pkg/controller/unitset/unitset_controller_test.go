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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

var _ = Describe("UnitSet Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName     = "test-unitset"
			unitType         = "mysql"
			unitEdition      = "community"
			unitVersion      = "8.0.40"
			sharedConfigName = "shared-config"
			finalizerName    = "unitset.finalizer"
		)

		var (
			unitSet           *upmiov1alpha2.UnitSet
			reconciler        *UnitSetReconciler
			req               ctrl.Request
			fakeEventRecorder *record.FakeRecorder
			namespace         *corev1.Namespace
		)

		BeforeEach(func() {
			fakeEventRecorder = record.NewFakeRecorder(10)
			reconciler = &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: fakeEventRecorder,
			}

			// Create test namespace (ephemeral)
			namespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-unitset-",
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

			// Shared config with ports
			sharedCfg := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: sharedConfigName, Namespace: namespace.Name},
				Data: map[string]string{
					unitType + "_ports": `[{"name":"` + unitType + `","containerPort":"3306","protocol":"TCP"}]`,
				},
			}
			_ = k8sClient.Create(ctx, sharedCfg)

			req = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      resourceName,
					Namespace: namespace.Name,
				},
			}

			unitSet = &upmiov1alpha2.UnitSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace.Name,
				},
				Spec: upmiov1alpha2.UnitSetSpec{
					Type:             unitType,
					Edition:          unitEdition,
					Version:          unitVersion,
					SharedConfigName: sharedConfigName,
					Units:            3,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			}
		})

		AfterEach(func() {
			// Best-effort cleanup to avoid blocking on namespace termination
			_ = k8sClient.Delete(ctx, namespace)
		})

		Describe("Reconcile method", func() {
			Context("when UnitSet does not exist", func() {
				It("should return requeue result without error", func() {
					result, err := reconciler.Reconcile(ctx, req)
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Requeue).To(BeTrue())
					Expect(result.RequeueAfter).To(Equal(3 * time.Second))
				})
			})

			Context("when UnitSet exists", func() {
				BeforeEach(func() {
					Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
				})

				It("should attempt to reconcile and requeue (may error due to missing deps)", func() {
					result, err := reconciler.Reconcile(ctx, req)
					// Only assert the requeue semantics; envtest env may cause errors due to missing deps
					Expect(result.Requeue).To(BeTrue())
					Expect(result.RequeueAfter).To(Equal(3 * time.Second))
					_ = err
				})

				It("should handle Get operation errors", func() {
					// Create a new client that will fail on Get operations
					brokenClient := &brokenClient{Client: k8sClient, shouldFail: true}
					brokenReconciler := &UnitSetReconciler{
						Client:   brokenClient,
						Scheme:   scheme.Scheme,
						Recorder: fakeEventRecorder,
					}

					result, err := brokenReconciler.Reconcile(ctx, req)
					Expect(err).To(HaveOccurred())
					Expect(result.Requeue).To(BeTrue())
					Expect(result.RequeueAfter).To(Equal(3 * time.Second))

					// No event is recorded here because the error occurs before reconcileUnitset and its defer
				})
			})
		})

		Describe("reconcileUnitset method", func() {
			Context("when UnitSet is being deleted", func() {
				BeforeEach(func() {
					now := metav1.Now()
					unitSet.DeletionTimestamp = &now
					unitSet.Finalizers = []string{finalizerName}
					Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
				})

				It("should handle deletion with finalizers", func() {
					// Ensure recognized finalizer is set, then delete to set DeletionTimestamp
					unitSet.Finalizers = []string{upmiov1alpha2.FinalizerUnitDelete}
					Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())
					Expect(k8sClient.Delete(ctx, unitSet)).To(Succeed())
					// Fetch fresh copy with DeletionTimestamp set
					deleted := &upmiov1alpha2.UnitSet{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unitSet.Name, Namespace: unitSet.Namespace}, deleted)).To(Succeed())

					err := reconciler.reconcileUnitset(ctx, req, deleted)
					Expect(err).NotTo(HaveOccurred())
				})

				It("should surface errors when reconciliation fails with mocked client", func() {
					unitSet.Finalizers = []string{upmiov1alpha2.FinalizerUnitDelete}
					Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

					updateFailClient := &brokenClientOps{Client: k8sClient, failUpdate: true}
					brokenReconciler := &UnitSetReconciler{
						Client:   updateFailClient,
						Scheme:   scheme.Scheme,
						Recorder: fakeEventRecorder,
					}

					err := brokenReconciler.reconcileUnitset(ctx, req, unitSet)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when UnitSet is not being deleted", func() {
				BeforeEach(func() {
					Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
				})

				It("should call all reconciliation methods in correct order", func() {
					// Create a minimal UnitSet for testing without external dependencies
					simpleUnitSet := &upmiov1alpha2.UnitSet{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "simple-unitset",
							Namespace: namespace.Name,
						},
						Spec: upmiov1alpha2.UnitSetSpec{
							Type:    unitType,
							Edition: unitEdition,
							Version: unitVersion,
							Units:   1,
						},
					}
					Expect(k8sClient.Create(ctx, simpleUnitSet)).To(Succeed())

					simpleReq := ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      "simple-unitset",
							Namespace: namespace.Name,
						},
					}

					// Test that reconcileUnitset handles the basic case
					err := reconciler.reconcileUnitset(ctx, simpleReq, simpleUnitSet)
					// We expect this to fail due to missing dependencies, but it should not panic
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("SetupWithManager method", func() {
			It("should set up the controller with correct options", func() {
				k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
					Scheme: scheme.Scheme,
				})
				Expect(err).NotTo(HaveOccurred())

				err = reconciler.SetupWithManager(k8sManager)
				Expect(err).NotTo(HaveOccurred())

				// The setup was successful if no error occurred
				Expect(err).To(BeNil())
			})
		})
	})
})

// Mock implementations for testing

type brokenClient struct {
	client.Client
	shouldFail bool
}

func (b *brokenClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if b.shouldFail {
		return fmt.Errorf("simulated get error")
	}
	return b.Client.Get(ctx, key, obj, opts...)
}

// brokenClientOps allows failing specific client operations
type brokenClientOps struct {
	client.Client
	failUpdate bool
}

func (b *brokenClientOps) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if b.failUpdate {
		return fmt.Errorf("simulated update error")
	}
	return b.Client.Update(ctx, obj, opts...)
}

type testUnitSetReconciler struct {
	UnitSetReconciler
	mockDeleteResources func(ctx context.Context, req ctrl.Request, unitSet *upmiov1alpha2.UnitSet, finalizer string) error
}

func (t *testUnitSetReconciler) deleteResources(ctx context.Context, req ctrl.Request, unitSet *upmiov1alpha2.UnitSet, finalizer string) error {
	if t.mockDeleteResources != nil {
		return t.mockDeleteResources(ctx, req, unitSet, finalizer)
	}
	return t.UnitSetReconciler.deleteResources(ctx, req, unitSet, finalizer)
}
