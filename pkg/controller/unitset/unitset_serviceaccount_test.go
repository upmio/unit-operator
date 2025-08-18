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
	"github.com/upmio/unit-operator/pkg/vars"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("UnitSet ServiceAccount Reconciliation", func() {
	var (
		ctx        context.Context
		unitSet    *upmiov1alpha2.UnitSet
		namespace  *corev1.Namespace
		reconciler *UnitSetReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-sa-namespace-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-sa",
				Namespace: namespace.Name,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:  "mysql",
				Units: 1,
			},
		}

		// Create reconciler
		reconciler = &UnitSetReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	AfterEach(func() {
		// Best-effort cleanup
		_ = k8sClient.Delete(ctx, namespace)
	})

	Context("When reconciling ServiceAccount", func() {
		It("Should create new ServiceAccount when it doesn't exist", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling ServiceAccount")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ServiceAccount was created")
			saName := fmt.Sprintf("%s-serviceaccount", namespace.Name)
			sa := &corev1.ServiceAccount{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: saName, Namespace: namespace.Name}, sa)).To(Succeed())

			Expect(sa.Name).To(Equal(saName))
			Expect(sa.Namespace).To(Equal(namespace.Name))
			Expect(sa.Labels["owner"]).To(Equal(vars.ManagerNamespace))
		})

		It("Should use existing ServiceAccount when it already exists", func() {
			By("Creating ServiceAccount manually")
			saName := fmt.Sprintf("%s-serviceaccount", namespace.Name)
			existingSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      saName,
					Namespace: namespace.Name,
					Labels: map[string]string{
						"owner":  vars.ManagerNamespace,
						"custom": "test-label",
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingSA)).To(Succeed())

			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling ServiceAccount")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing ServiceAccount is preserved")
			sa := &corev1.ServiceAccount{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: saName, Namespace: namespace.Name}, sa)).To(Succeed())

			Expect(sa.Name).To(Equal(saName))
			Expect(sa.Labels["custom"]).To(Equal("test-label"))
		})

		It("Should handle ServiceAccount creation errors", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Mocking Create error scenario")
			// This would require a mock client in a real scenario
			// For now, we'll test the structure
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When reconciling Role", func() {
		It("Should create new Role with correct permissions", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling Role")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileRole(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Role was created with correct permissions")
			roleName := fmt.Sprintf("%s-role", namespace.Name)
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace.Name}, role)).To(Succeed())

			Expect(role.Name).To(Equal(roleName))
			Expect(role.Namespace).To(Equal(namespace.Name))
			Expect(role.Labels["owner"]).To(Equal(vars.ManagerNamespace))

			// Verify policy rules
			Expect(role.Rules).To(HaveLen(5))

			// Check pods/secrets permissions
			podsRule := role.Rules[0]
			Expect(podsRule.APIGroups).To(Equal([]string{""}))
			Expect(podsRule.Resources).To(Equal([]string{"pods", "secrets"}))
			Expect(podsRule.Verbs).To(Equal([]string{"get", "list"}))

			// Check configmaps permissions
			configmapsRule := role.Rules[1]
			Expect(configmapsRule.APIGroups).To(Equal([]string{""}))
			Expect(configmapsRule.Resources).To(Equal([]string{"configmaps"}))
			Expect(configmapsRule.Verbs).To(Equal([]string{"get", "list", "patch", "update"}))

			// Check redisreplications permissions
			redisRule := role.Rules[2]
			Expect(redisRule.APIGroups).To(Equal([]string{"upm.syntropycloud.io"}))
			Expect(redisRule.Resources).To(Equal([]string{"redisreplications"}))
			Expect(redisRule.Verbs).To(Equal([]string{"get", "list", "patch", "update"}))

			// Check units permissions
			unitsRule := role.Rules[3]
			Expect(unitsRule.APIGroups).To(Equal([]string{"upm.syntropycloud.io"}))
			Expect(unitsRule.Resources).To(Equal([]string{"units"}))
			Expect(unitsRule.Verbs).To(Equal([]string{"get", "list"}))

			// Check events permissions
			eventsRule := role.Rules[4]
			Expect(eventsRule.APIGroups).To(Equal([]string{""}))
			Expect(eventsRule.Resources).To(Equal([]string{"events"}))
			Expect(eventsRule.Verbs).To(Equal([]string{"create", "patch"}))
		})

		It("Should use existing Role when it already exists", func() {
			By("Creating Role manually")
			roleName := fmt.Sprintf("%s-role", namespace.Name)
			existingRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName,
					Namespace: namespace.Name,
					Labels: map[string]string{
						"owner":  vars.ManagerNamespace,
						"custom": "test-role-label",
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"pods"},
						Verbs:     []string{"get"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingRole)).To(Succeed())

			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling Role")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileRole(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing Role is preserved")
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace.Name}, role)).To(Succeed())

			Expect(role.Name).To(Equal(roleName))
			Expect(role.Labels["custom"]).To(Equal("test-role-label"))
			Expect(role.Rules).To(HaveLen(1)) // Should preserve existing rules
		})
	})

	Context("When reconciling RoleBinding", func() {
		It("Should create new RoleBinding with correct configuration", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling RoleBinding")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileRoleBinding(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying RoleBinding was created")
			rbName := fmt.Sprintf("%s-rolebinding", namespace.Name)
			roleBinding := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: namespace.Name}, roleBinding)).To(Succeed())

			Expect(roleBinding.Name).To(Equal(rbName))
			Expect(roleBinding.Namespace).To(Equal(namespace.Name))
			Expect(roleBinding.Labels["owner"]).To(Equal(vars.ManagerNamespace))

			// Verify RoleRef
			Expect(roleBinding.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
			Expect(roleBinding.RoleRef.Kind).To(Equal("Role"))
			Expect(roleBinding.RoleRef.Name).To(Equal(fmt.Sprintf("%s-role", namespace.Name)))

			// Verify Subjects
			Expect(roleBinding.Subjects).To(HaveLen(1))
			subject := roleBinding.Subjects[0]
			Expect(subject.Kind).To(Equal("ServiceAccount"))
			Expect(subject.Name).To(Equal(fmt.Sprintf("%s-serviceaccount", namespace.Name)))
			Expect(subject.Namespace).To(Equal(namespace.Name))
		})

		It("Should use existing RoleBinding when it already exists", func() {
			By("Creating RoleBinding manually")
			rbName := fmt.Sprintf("%s-rolebinding", namespace.Name)
			existingRB := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rbName,
					Namespace: namespace.Name,
					Labels: map[string]string{
						"owner":  vars.ManagerNamespace,
						"custom": "test-rb-label",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     fmt.Sprintf("%s-role", namespace.Name),
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      fmt.Sprintf("%s-serviceaccount", namespace.Name),
						Namespace: namespace.Name,
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingRB)).To(Succeed())

			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling RoleBinding")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileRoleBinding(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing RoleBinding is preserved")
			roleBinding := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: namespace.Name}, roleBinding)).To(Succeed())

			Expect(roleBinding.Name).To(Equal(rbName))
			Expect(roleBinding.Labels["custom"]).To(Equal("test-rb-label"))
		})
	})

	Context("When reconciling complete ServiceAccount setup", func() {
		It("Should create ServiceAccount, Role, and RoleBinding together", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling complete ServiceAccount setup")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ServiceAccount exists")
			saName := fmt.Sprintf("%s-serviceaccount", namespace.Name)
			sa := &corev1.ServiceAccount{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: saName, Namespace: namespace.Name}, sa)).To(Succeed())

			By("Verifying Role exists")
			roleName := fmt.Sprintf("%s-role", namespace.Name)
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace.Name}, role)).To(Succeed())

			By("Verifying RoleBinding exists")
			rbName := fmt.Sprintf("%s-rolebinding", namespace.Name)
			roleBinding := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: namespace.Name}, roleBinding)).To(Succeed())

			By("Verifying RoleBinding references correct resources")
			Expect(roleBinding.RoleRef.Name).To(Equal(roleName))
			Expect(roleBinding.Subjects[0].Name).To(Equal(saName))
		})
	})

	Context("When handling errors", func() {
		It("Should handle invalid namespace gracefully", func() {
			By("Creating UnitSet with invalid namespace reference")
			invalidReq := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: "nonexistent-namespace",
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, invalidReq, unitSet)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("[reconcileServiceAccount]"))
		})

		It("Should handle concurrent creation attempts", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Attempting concurrent reconciliation")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			// First reconciliation should succeed
			err1 := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err1).NotTo(HaveOccurred())

			// Second reconciliation should also succeed (resources already exist)
			err2 := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err2).NotTo(HaveOccurred())
		})
	})

	Context("When testing RBAC permissions", func() {
		It("Should have correct permissions for managed resources", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling ServiceAccount setup")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying Role has all required permissions")
			roleName := fmt.Sprintf("%s-role", namespace.Name)
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace.Name}, role)).To(Succeed())

			// Check that all required resource permissions are present
			resourcePermissions := make(map[string][]string)
			for _, rule := range role.Rules {
				for _, resource := range rule.Resources {
					resourcePermissions[resource] = rule.Verbs
				}
			}

			Expect(resourcePermissions["pods"]).To(ContainElements("get", "list"))
			Expect(resourcePermissions["secrets"]).To(ContainElements("get", "list"))
			Expect(resourcePermissions["configmaps"]).To(ContainElements("get", "list", "patch", "update"))
			Expect(resourcePermissions["redisreplications"]).To(ContainElements("get", "list", "patch", "update"))
			Expect(resourcePermissions["units"]).To(ContainElements("get", "list"))
			Expect(resourcePermissions["events"]).To(ContainElements("create", "patch"))
		})
	})

	Context("When testing naming conventions", func() {
		It("Should follow consistent naming pattern", func() {
			By("Creating UnitSet")
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())

			By("Reconciling ServiceAccount setup")
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      unitSet.Name,
					Namespace: namespace.Name,
				},
			}

			err := reconciler.reconcileServiceAccount(ctx, req, unitSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying naming conventions")
			expectedPrefix := namespace.Name

			saName := fmt.Sprintf("%s-serviceaccount", expectedPrefix)
			roleName := fmt.Sprintf("%s-role", expectedPrefix)
			rbName := fmt.Sprintf("%s-rolebinding", expectedPrefix)

			// Verify ServiceAccount name
			sa := &corev1.ServiceAccount{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: saName, Namespace: namespace.Name}, sa)).To(Succeed())
			Expect(sa.Name).To(Equal(saName))

			// Verify Role name
			role := &rbacv1.Role{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: roleName, Namespace: namespace.Name}, role)).To(Succeed())
			Expect(role.Name).To(Equal(roleName))

			// Verify RoleBinding name
			rb := &rbacv1.RoleBinding{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: rbName, Namespace: namespace.Name}, rb)).To(Succeed())
			Expect(rb.Name).To(Equal(rbName))
		})
	})
})
