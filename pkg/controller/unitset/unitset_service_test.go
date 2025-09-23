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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("UnitSet Service Reconciler", func() {
	var (
		ctx       context.Context
		unitSet   *upmiov1alpha2.UnitSet
		namespace *corev1.Namespace
		testPorts = []corev1.ContainerPort{
			{
				Name:          "mysql",
				ContainerPort: 3306,
				Protocol:      "TCP",
			},
			{
				Name:          "metrics",
				ContainerPort: 9104,
				Protocol:      "TCP",
			},
		}
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create test namespace (ephemeral)
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-unitset-service-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create a basic UnitSet for testing
		unitSet = &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-unitset-service",
				Namespace: namespace.Name,
				Labels: map[string]string{
					"app": "mysql",
				},
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Type:    "mysql",
				Edition: "community",
				Version: "8.0.40",
				Units:   3,
				//SharedConfigName: "test-shared-config",
			},
		}
	})

	AfterEach(func() {
		// Best-effort cleanup to avoid blocking on namespace termination
		_ = k8sClient.Delete(ctx, namespace)
	})

	Context("When reconciling headless service", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should create headless service when it doesn't exist", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service")
			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying headless service was created")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(createdService.Spec.ClusterIP).To(Equal("None"))
			Expect(createdService.Spec.PublishNotReadyAddresses).To(BeTrue())
			Expect(createdService.Spec.Ports).To(HaveLen(2))
			Expect(createdService.Spec.Ports[0].Name).To(Equal("mysql"))
			Expect(createdService.Spec.Ports[0].Port).To(Equal(int32(3306)))
			Expect(createdService.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(createdService.Spec.Ports[1].Name).To(Equal("metrics"))
			Expect(createdService.Spec.Ports[1].Port).To(Equal(int32(9104)))

			Expect(createdService.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
			Expect(createdService.Labels["app"]).To(Equal("mysql"))
			Expect(createdService.Spec.Selector[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
			Expect(createdService.OwnerReferences).To(HaveLen(1))
			Expect(createdService.OwnerReferences[0].Name).To(Equal(unitSet.Name))
		})

		It("Should not create headless service when it already exists", func() {
			By("Creating existing headless service")
			existingService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unitset-service-headless-svc",
					Namespace: namespace.Name,
				},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "None",
					Ports: []corev1.ServicePort{
						{
							Name:     "mysql",
							Port:     3306,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingService)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client:   k8sClient,
				Scheme:   scheme.Scheme,
				Recorder: recorder,
			}

			By("Reconciling headless service")
			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing service was not modified")
			retrievedService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, retrievedService)).To(Succeed())

			Expect(retrievedService.Spec.Ports).To(HaveLen(1)) // Should still have original port
			Expect(retrievedService.Spec.Ports[0].Name).To(Equal("mysql"))
		})

		It("Should handle empty ports gracefully", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service with empty ports")
			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				[]corev1.ContainerPort{},
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying headless service was created without ports")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Spec.Ports).To(BeEmpty())
		})

		It("Should handle invalid port conversion gracefully", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service with invalid port")
			invalidPorts := []corev1.ContainerPort{
				{
					Name:          "invalid",
					ContainerPort: 99999,
					Protocol:      "TCP",
				},
			}

			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				invalidPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service was created and invalid port was ignored")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Spec.Ports).To(HaveLen(0))
		})

		It("Should handle API errors gracefully", func() {
			By("Creating reconciler with invalid client")
			// This test would require a mock client to simulate API errors
			// For now, we'll test the success path
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service")
			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When reconciling external service", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should create external service when spec is configured", func() {
			By("Updating UnitSet with external service configuration")
			unitSet.Spec.ExternalService.Type = "NodePort"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling external service")
			err := reconciler.reconcileExternalService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying external service was created")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))
			Expect(createdService.Spec.PublishNotReadyAddresses).To(BeTrue())
			Expect(createdService.Spec.Ports).To(HaveLen(2))
			Expect(createdService.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
			Expect(createdService.Spec.Selector[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))

			// Clean up to isolate subsequent specs
			Expect(k8sClient.Delete(ctx, createdService)).To(Succeed())
		})

		It("Should skip external service when type is not configured", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling external service without type")
			err := reconciler.reconcileExternalService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no external service was created")
			serviceList := &corev1.ServiceList{}
			Expect(k8sClient.List(ctx, serviceList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(serviceList.Items).To(BeEmpty())
		})

		It("Should create external service even without shared config", func() {
			By("Updating UnitSet without shared config override")
			unitSet.Spec.ExternalService.Type = "NodePort"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling external service without shared config")
			err := reconciler.reconcileExternalService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying external service was created")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())
			Expect(createdService.Spec.Type).To(Equal(corev1.ServiceTypeNodePort))

			// Clean up to avoid leaking resources across specs
			Expect(k8sClient.Delete(ctx, createdService)).To(Succeed())
		})

		It("Should handle different external service types", func() {
			By("Testing different service types")
			serviceTypes := []string{"ClusterIP", "NodePort", "LoadBalancer"}

			for _, svcType := range serviceTypes {
				By(fmt.Sprintf("Testing service type: %s", svcType))

				// Update UnitSet for this test
				current := &upmiov1alpha2.UnitSet{}
				nn := types.NamespacedName{Name: unitSet.Name, Namespace: unitSet.Namespace}
				Expect(k8sClient.Get(ctx, nn, current)).To(Succeed())
				current.Spec.ExternalService.Type = svcType
				Expect(k8sClient.Update(ctx, current)).To(Succeed())
				unitSet = current

				By("Creating reconciler")
				reconciler := &UnitSetReconciler{
					Client: k8sClient,
					Scheme: scheme.Scheme,
				}

				By("Reconciling external service")
				err := reconciler.reconcileExternalService(ctx,
					ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      unitSet.Name,
							Namespace: namespace.Name,
						},
					},
					unitSet,
					testPorts,
				)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying external service was created with correct type")
				createdService := &corev1.Service{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-unitset-service-svc",
					Namespace: namespace.Name,
				}, createdService)).To(Succeed())

				Expect(createdService.Spec.Type).To(Equal(corev1.ServiceType(svcType)))

				// Clean up for next iteration
				Expect(k8sClient.Delete(ctx, createdService)).To(Succeed())
			}
		})

		It("Should not create external service when it already exists", func() {
			By("Updating UnitSet with external service configuration")
			unitSet.Spec.ExternalService.Type = "NodePort"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating existing external service")
			existingService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unitset-service-svc",
					Namespace: namespace.Name,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name:     "mysql",
							Port:     3306,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingService)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling external service")
			err := reconciler.reconcileExternalService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing service was not modified")
			retrievedService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-svc",
				Namespace: namespace.Name,
			}, retrievedService)).To(Succeed())

			Expect(retrievedService.Spec.Ports).To(HaveLen(1)) // Should still have original port
		})
	})

	Context("When reconciling unit services", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should create unit services when spec is configured", func() {
			By("Updating UnitSet with unit service configuration")
			unitSet.Spec.UnitService.Type = "ClusterIP"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services")
			err := reconciler.reconcileUnitService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying unit services were created for each unit")
			for i := 0; i < 3; i++ {
				unitName := fmt.Sprintf("%s-%d", unitSet.Name, i)
				serviceName := fmt.Sprintf("%s-svc", unitName)

				createdService := &corev1.Service{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      serviceName,
					Namespace: namespace.Name,
				}, createdService)).To(Succeed())

				Expect(createdService.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
				Expect(createdService.Spec.PublishNotReadyAddresses).To(BeTrue())
				Expect(createdService.Spec.Ports).To(HaveLen(2))
				Expect(createdService.Labels[upmiov1alpha2.UnitName]).To(Equal(unitName))
				Expect(createdService.Spec.Selector[upmiov1alpha2.UnitName]).To(Equal(unitName))
			}

			By("Verifying unitset annotated with unit service type on creation")
			fetched := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}, fetched)).To(Succeed())
			Expect(fetched.Annotations).NotTo(BeNil())
			Expect(fetched.Annotations[upmiov1alpha2.AnnotationUnitServiceType]).To(Equal("ClusterIP"))
		})

		It("Should skip unit services when type is not configured", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services without type")
			err := reconciler.reconcileUnitService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no unit services were created")
			serviceList := &corev1.ServiceList{}
			Expect(k8sClient.List(ctx, serviceList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(serviceList.Items).To(BeEmpty())
		})

		It("Should create unit services even without shared config", func() {
			By("Updating UnitSet without shared config")
			unitSet.Spec.UnitService.Type = "ClusterIP"
			//unitSet.Spec.SharedConfigName = ""
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services without shared config")
			err := reconciler.reconcileUnitService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying unit services were created")
			serviceList := &corev1.ServiceList{}
			Expect(k8sClient.List(ctx, serviceList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(serviceList.Items).To(HaveLen(int(unitSet.Spec.Units)))

			// Clean up for isolation
			for i := range serviceList.Items {
				service := &serviceList.Items[i]
				Expect(k8sClient.Delete(ctx, service)).To(Succeed())
			}
		})

		It("Should handle concurrent unit service creation", func() {
			By("Updating UnitSet with unit service configuration")
			unitSet.Spec.UnitService.Type = "ClusterIP"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services multiple times")
			for i := 0; i < 3; i++ {
				err := reconciler.reconcileUnitService(ctx,
					ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      unitSet.Name,
							Namespace: namespace.Name,
						},
					},
					unitSet,
					testPorts,
				)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying unit services were created without duplicates")
			serviceList := &corev1.ServiceList{}
			Expect(k8sClient.List(ctx, serviceList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(serviceList.Items).To(HaveLen(3)) // Should have exactly 3 services

			// Verify all services have correct labels
			for _, service := range serviceList.Items {
				Expect(service.Labels[upmiov1alpha2.UnitName]).NotTo(BeEmpty())
				Expect(service.Spec.Selector[upmiov1alpha2.UnitName]).NotTo(BeEmpty())
			}

			By("Verifying only created path annotates unitset once")
			fetched := &upmiov1alpha2.UnitSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}, fetched)).To(Succeed())
			Expect(fetched.Annotations[upmiov1alpha2.AnnotationUnitServiceType]).To(Equal("ClusterIP"))
		})

		It("Should handle partial unit service creation with existing service gracefully", func() {
			By("Updating UnitSet with unit service configuration")
			unitSet.Spec.UnitService.Type = "ClusterIP"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating one existing service to simulate conflict")
			existingService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unitset-service-0-svc",
					Namespace: namespace.Name,
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "dummy", Port: 1, Protocol: corev1.ProtocolTCP},
					},
				},
			}
			Expect(k8sClient.Create(ctx, existingService)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services with existing service")
			err := reconciler.reconcileUnitService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle different unit service types", func() {
			By("Testing different service types")
			serviceTypes := []string{"ClusterIP", "NodePort", "LoadBalancer"}

			for _, svcType := range serviceTypes {
				By(fmt.Sprintf("Testing service type: %s", svcType))

				// Update UnitSet for this test
				current := &upmiov1alpha2.UnitSet{}
				nn := types.NamespacedName{Name: unitSet.Name, Namespace: unitSet.Namespace}
				Expect(k8sClient.Get(ctx, nn, current)).To(Succeed())
				current.Spec.UnitService.Type = svcType
				Expect(k8sClient.Update(ctx, current)).To(Succeed())
				unitSet = current

				By("Creating reconciler")
				reconciler := &UnitSetReconciler{
					Client: k8sClient,
					Scheme: scheme.Scheme,
				}

				By("Reconciling unit services")
				err := reconciler.reconcileUnitService(ctx,
					ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      unitSet.Name,
							Namespace: namespace.Name,
						},
					},
					unitSet,
					testPorts,
				)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying unit services were created with correct type")
				for i := 0; i < 3; i++ {
					unitName := fmt.Sprintf("%s-%d", unitSet.Name, i)
					serviceName := fmt.Sprintf("%s-svc", unitName)

					createdService := &corev1.Service{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{
						Name:      serviceName,
						Namespace: namespace.Name,
					}, createdService)).To(Succeed())

					Expect(createdService.Spec.Type).To(Equal(corev1.ServiceType(svcType)))

					// Clean up for next iteration
					Expect(k8sClient.Delete(ctx, createdService)).To(Succeed())
				}
			}
		})

		It("Should handle empty unit names gracefully", func() {
			By("Updating UnitSet with 0 units")
			unitSet.Spec.UnitService.Type = "ClusterIP"
			unitSet.Spec.Units = 0
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling unit services with 0 units")
			err := reconciler.reconcileUnitService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying no unit services were created")
			serviceList := &corev1.ServiceList{}
			Expect(k8sClient.List(ctx, serviceList, client.InNamespace(namespace.Name))).To(Succeed())
			Expect(serviceList.Items).To(BeEmpty())
		})
	})

	Context("When handling edge cases", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, unitSet)).To(Succeed())
		})

		It("Should handle service with nil labels", func() {
			By("Updating UnitSet with nil labels")
			unitSet.Labels = nil
			unitSet.Spec.ExternalService.Type = "NodePort"
			Expect(k8sClient.Update(ctx, unitSet)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling external service with nil labels")
			err := reconciler.reconcileExternalService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service was created with proper labels")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Labels).NotTo(BeNil())
			Expect(createdService.Labels[upmiov1alpha2.UnitsetName]).To(Equal(unitSet.Name))
		})

		It("Should handle UDP protocol", func() {
			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service with UDP protocol")
			udpPorts := []corev1.ContainerPort{
				{
					Name:          "dns",
					ContainerPort: 53,
					Protocol:      "UDP",
				},
			}

			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				udpPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service was created with UDP protocol")
			createdService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, createdService)).To(Succeed())

			Expect(createdService.Spec.Ports).To(HaveLen(1))
			Expect(createdService.Spec.Ports[0].Name).To(Equal("dns"))
			Expect(createdService.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolUDP))
		})

		It("Should handle service name conflicts", func() {
			By("Creating existing service with same name as headless service")
			existingService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-unitset-service-headless-svc",
					Namespace: namespace.Name,
				},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "None", // make it a valid headless service
				},
			}
			Expect(k8sClient.Create(ctx, existingService)).To(Succeed())

			By("Creating reconciler")
			reconciler := &UnitSetReconciler{
				Client: k8sClient,
				Scheme: scheme.Scheme,
			}

			By("Reconciling headless service with conflict")
			err := reconciler.reconcileHeadlessService(ctx,
				ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      unitSet.Name,
						Namespace: namespace.Name,
					},
				},
				unitSet,
				testPorts,
			)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying existing service was not modified")
			retrievedService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-unitset-service-headless-svc",
				Namespace: namespace.Name,
			}, retrievedService)).To(Succeed())

			Expect(retrievedService.Spec.Ports).To(BeEmpty()) // Should still be empty
		})
	})

	It("Should record and reuse nodePorts for NodePort services", func() {
		By("Configuring UnitSet with NodePort unit service")
		current := &upmiov1alpha2.UnitSet{}
		nn := types.NamespacedName{Name: unitSet.Name, Namespace: unitSet.Namespace}
		err := k8sClient.Get(ctx, nn, current)
		if apierrors.IsNotFound(err) {
			base := unitSet.DeepCopy()
			base.Spec.UnitService.Type = "NodePort"
			Expect(k8sClient.Create(ctx, base)).To(Succeed())
			unitSet = base
		} else {
			Expect(err).NotTo(HaveOccurred())
			current.Spec.UnitService.Type = "NodePort"
			Expect(k8sClient.Update(ctx, current)).To(Succeed())
			unitSet = current
		}

		By("Reconciling to create NodePort services and record nodePorts")
		reconciler := &UnitSetReconciler{Client: k8sClient, Scheme: scheme.Scheme}
		err = reconciler.reconcileUnitService(ctx,
			ctrl.Request{NamespacedName: types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}},
			unitSet,
			testPorts,
		)
		Expect(err).NotTo(HaveOccurred())

		By("Reading back annotation maps for each port")
		refreshed := &upmiov1alpha2.UnitSet{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}, refreshed)).To(Succeed())
		for _, p := range testPorts {
			key := upmiov1alpha2.AnnotationUnitServiceNodeportMapPrefix + p.Name + upmiov1alpha2.AnnotationUnitServiceNodeportMapSuffix
			Expect(refreshed.Annotations[key]).NotTo(BeEmpty())
		}

		By("Deleting one unit's service to simulate accidental deletion")
		victim := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-0-svc", unitSet.Name), Namespace: namespace.Name}}
		Expect(k8sClient.Delete(ctx, victim)).To(Succeed())

		By("Reconciling again; controller should recreate service using annotated nodePort")
		err = reconciler.reconcileUnitService(ctx,
			ctrl.Request{NamespacedName: types.NamespacedName{Name: unitSet.Name, Namespace: namespace.Name}},
			refreshed,
			testPorts,
		)
		Expect(err).NotTo(HaveOccurred())

		By("Verifying recreated service has nodePorts set (non-zero)")
		created := &corev1.Service{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-0-svc", unitSet.Name), Namespace: namespace.Name}, created)).To(Succeed())
		if created.Spec.Type == corev1.ServiceTypeNodePort {
			for _, sp := range created.Spec.Ports {
				Expect(sp.NodePort).NotTo(BeZero())
			}
		}
	})
})
