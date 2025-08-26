package unitset

import (
	"context"
	"fmt"

	certmanagerV1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("UnitSet Certificates Reconciler", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "test-ns-cert"
	})

	It("should no-op when units is zero", func() {
		us := &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "us-zero",
				Namespace: namespace,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Units: 0,
				CertificateProfile: upmiov1alpha2.CertificateProfile{
					Organizations: []string{"acme"},
					RootSecret:    "root-ca-secret",
				},
			},
		}

		Expect(upmiov1alpha2.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(certmanagerV1.AddToScheme(scheme.Scheme)).To(Succeed())

		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(us).Build()
		r := &UnitSetReconciler{Client: c, Scheme: scheme.Scheme}

		err := r.reconcileUnitCertificates(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: us.Name, Namespace: us.Namespace}}, us)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should no-op when certificate profile is incomplete", func() {
		us := &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "us-no-profile",
				Namespace: namespace,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Units: 2,
				CertificateProfile: upmiov1alpha2.CertificateProfile{
					Organizations: nil,
					RootSecret:    "",
				},
			},
		}

		Expect(upmiov1alpha2.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(certmanagerV1.AddToScheme(scheme.Scheme)).To(Succeed())

		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(us).Build()
		r := &UnitSetReconciler{Client: c, Scheme: scheme.Scheme}

		err := r.reconcileUnitCertificates(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: us.Name, Namespace: us.Namespace}}, us)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create issuer and certificate per unit when missing", func() {
		us := &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "us-create",
				Namespace: namespace,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Units: 2,
				CertificateProfile: upmiov1alpha2.CertificateProfile{
					Organizations: []string{"acme"},
					RootSecret:    "root-ca-secret",
				},
			},
		}

		Expect(upmiov1alpha2.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(certmanagerV1.AddToScheme(scheme.Scheme)).To(Succeed())

		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(us).Build()
		r := &UnitSetReconciler{Client: c, Scheme: scheme.Scheme}

		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: us.Name, Namespace: us.Namespace}}
		err := r.reconcileUnitCertificates(ctx, req, us)
		Expect(err).NotTo(HaveOccurred())

		// Verify resources created in the same namespace as UnitSet
		for i := 0; i < us.Spec.Units; i++ {
			unitName := fmt.Sprintf("%s-%d", us.Name, i)
			issuerName := fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerIssuerSuffix)
			certName := fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerCertificateSuffix)

			issuer := &certmanagerV1.Issuer{}
			Expect(c.Get(ctx, types.NamespacedName{Name: issuerName, Namespace: namespace}, issuer)).To(Succeed())

			cert := &certmanagerV1.Certificate{}
			Expect(c.Get(ctx, types.NamespacedName{Name: certName, Namespace: namespace}, cert)).To(Succeed())

			// Basic fields
			Expect(cert.Spec.IssuerRef.Kind).To(Equal("Issuer"))
			Expect(cert.Spec.IssuerRef.Group).To(Equal("cert-manager.io"))
			Expect(cert.Spec.DNSNames).To(ContainElement(unitName))
			Expect(cert.Spec.SecretName).To(Equal(fmt.Sprintf("%s-%s", certName, upmiov1alpha2.CertmanagerSecretNameSuffix)))
		}
	})

	It("should be idempotent when issuer/certificate already exist", func() {
		us := &upmiov1alpha2.UnitSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "us-exist",
				Namespace: namespace,
			},
			Spec: upmiov1alpha2.UnitSetSpec{
				Units: 1,
				CertificateProfile: upmiov1alpha2.CertificateProfile{
					Organizations: []string{"acme"},
					RootSecret:    "root-ca-secret",
				},
			},
		}

		Expect(upmiov1alpha2.AddToScheme(scheme.Scheme)).To(Succeed())
		Expect(certmanagerV1.AddToScheme(scheme.Scheme)).To(Succeed())

		// Pre-create Issuer and Certificate
		unitName := fmt.Sprintf("%s-0", us.Name)
		issuer := &certmanagerV1.Issuer{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerIssuerSuffix), Namespace: namespace}}
		cert := &certmanagerV1.Certificate{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerCertificateSuffix), Namespace: namespace}}

		c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(us, issuer, cert).Build()
		r := &UnitSetReconciler{Client: c, Scheme: scheme.Scheme}

		err := r.reconcileUnitCertificates(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: us.Name, Namespace: us.Namespace}}, us)
		Expect(err).NotTo(HaveOccurred())
	})
})
