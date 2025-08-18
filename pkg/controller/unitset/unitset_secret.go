package unitset

import (
	"context"
	"fmt"
	"sync"
	"time"

	certmanagerV1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagerApiV1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcileUnitCertificates(
	ctx context.Context,
	req ctrl.Request,
	unitset *upmiov1alpha2.UnitSet,
) error {
	units, _ := unitset.UnitNames()
	if len(units) == 0 {
		return nil
	}

	if len(unitset.Spec.CertificateProfile.Organizations) == 0 || unitset.Spec.CertificateProfile.RootSecret == "" {
		return nil
	}

	errs := []error{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, unit := range units {
		wg.Add(1)
		go func(unitName string) {
			defer wg.Done()

			blockOwnerDeletion := true
			ref := metav1.NewControllerRef(unitset, controllerKind)
			ref.BlockOwnerDeletion = &blockOwnerDeletion

			// cert-certificate, cert-issuer
			issuerName := fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerIssuerSuffix)
			issuer := &certmanagerV1.Issuer{}
			issuerNamespacedName := client.ObjectKey{Name: issuerName, Namespace: req.Namespace}
			getIssuerErr := r.Get(ctx, issuerNamespacedName, issuer)
			if getIssuerErr != nil && errors.IsNotFound(getIssuerErr) {
				caIssuer := certmanagerV1.CAIssuer{
					SecretName:            unitset.Spec.CertificateProfile.RootSecret,
					CRLDistributionPoints: nil,
					OCSPServers:           nil,
				}

				issuer := &certmanagerV1.Issuer{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       req.Namespace,
						Name:            issuerName,
						OwnerReferences: []metav1.OwnerReference{*ref},
					},
					Spec: certmanagerV1.IssuerSpec{
						IssuerConfig: certmanagerV1.IssuerConfig{
							CA: &caIssuer,
						},
					},
				}

				createIssuerErr := r.Create(ctx, issuer)
				if createIssuerErr != nil && !errors.IsAlreadyExists(createIssuerErr) {
					mu.Lock()
					errs = append(errs, fmt.Errorf("[reconcileUnitCertificates] create cert-manager issuer [%s] error: %s", issuerName, createIssuerErr.Error()))
					mu.Unlock()
				}
			}

			if getIssuerErr != nil && !errors.IsNotFound(getIssuerErr) {
				mu.Lock()
				errs = append(errs, fmt.Errorf("[reconcileUnitCertificates] get cert-manager issuer [%s] error: %s", issuerName, getIssuerErr.Error()))
				mu.Unlock()
			}

			certificateName := fmt.Sprintf("%s-%s", unitName, upmiov1alpha2.CertmanagerCertificateSuffix)
			secretName := fmt.Sprintf("%s-%s", certificateName, upmiov1alpha2.CertmanagerSecretNameSuffix)

			certificate := &certmanagerV1.Certificate{}
			certificateNamespacedName := client.ObjectKey{Name: certificateName, Namespace: req.Namespace}
			getCertErr := r.Get(ctx, certificateNamespacedName, certificate)
			if getCertErr != nil && errors.IsNotFound(getCertErr) {
				privateKey := certmanagerV1.CertificatePrivateKey{
					Algorithm: certmanagerV1.RSAKeyAlgorithm,
					Encoding:  certmanagerV1.PKCS8,
					Size:      2048,
				}

				x509Subject := certmanagerV1.X509Subject{
					Organizations: unitset.Spec.CertificateProfile.Organizations,
				}

				duration := metav1.Duration{Duration: time.Hour * 87600}
				renewBefore := metav1.Duration{Duration: time.Hour * 2160}

				certificate := &certmanagerV1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:            certificateName,
						Namespace:       req.Namespace,
						OwnerReferences: []metav1.OwnerReference{*ref},
					},
					Spec: certmanagerV1.CertificateSpec{
						IsCA: false,
						DNSNames: []string{
							unitName,
						},
						Subject:    &x509Subject,
						PrivateKey: &privateKey,
						IssuerRef: certmanagerApiV1.ObjectReference{
							Group: "cert-manager.io",
							Kind:  "Issuer",
							Name:  issuerName,
						},
						SecretName:  secretName,
						Duration:    &duration,
						RenewBefore: &renewBefore,
					},
				}

				createCertErr := r.Create(ctx, certificate)
				if createCertErr != nil && !errors.IsAlreadyExists(createCertErr) {
					mu.Lock()
					errs = append(errs, fmt.Errorf("[reconcileUnitCertificates] create cert-manager certificate [%s] error: %s", certificateName, createCertErr.Error()))
					mu.Unlock()
				}
			}

			if getCertErr != nil && !errors.IsNotFound(getCertErr) {
				mu.Lock()
				errs = append(errs, fmt.Errorf("[reconcileUnitCertificates] get cert-manager certificate [%s] error: %s", certificateName, getCertErr.Error()))
				mu.Unlock()
			}

		}(unit)
	}

	wg.Wait()

	if agg := utilerrors.NewAggregate(errs); agg != nil {
		return fmt.Errorf("[reconcileUnitCertificates] error: [%s]", agg.Error())
	}

	return nil
}
