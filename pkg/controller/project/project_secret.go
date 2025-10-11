package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmMetaApi "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ProjectReconciler) reconcileCA(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	if !project.Spec.CA.Enabled {
		return nil
	}

	ns := project.Name
	caSpec := project.Spec.CA

	// --- Step 1: ensure Issuer  ---
	issuerName := fmt.Sprintf("%s-%s", project.Name, upmiov1alpha2.CertmanagerIssuerSuffix)
	issuer := &cmapi.Issuer{}
	err := r.Get(ctx, client.ObjectKey{Name: issuerName, Namespace: ns}, issuer)
	if apierrors.IsNotFound(err) {
		newIssuer := &cmapi.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      issuerName,
				Namespace: ns,
			},
			Spec: cmapi.IssuerSpec{
				IssuerConfig: cmapi.IssuerConfig{
					SelfSigned: nil,
				},
			},
		}
		if err := r.Create(ctx, newIssuer); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// --- Step 2: ensure Certificate  ---
	certName := fmt.Sprintf("%s-%s", project.Name, upmiov1alpha2.CertmanagerCertificateSuffix)
	cert := &cmapi.Certificate{}
	err = r.Get(ctx, client.ObjectKey{Name: certName, Namespace: ns}, cert)
	if apierrors.IsNotFound(err) {
		duration, _ := time.ParseDuration(defaultIfEmpty(caSpec.Duration, "87600h"))      // default: 10 years
		renewBefore, _ := time.ParseDuration(defaultIfEmpty(caSpec.RenewBefore, "2160h")) // default: 90 days
		newCert := &cmapi.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certName,
				Namespace: ns,
			},
			Spec: cmapi.CertificateSpec{
				SecretName:  caSpec.SecretName,
				CommonName:  caSpec.CommonName,
				IsCA:        true,
				Duration:    &metav1.Duration{Duration: duration},
				RenewBefore: &metav1.Duration{Duration: renewBefore},
				PrivateKey: &cmapi.CertificatePrivateKey{
					Algorithm: cmapi.PrivateKeyAlgorithm(caSpec.PrivateKey.Algorithm),
					Size:      caSpec.PrivateKey.Size,
				},
				IssuerRef: cmMetaApi.ObjectReference{
					Name: issuerName,
					Kind: "Issuer",
				},
			},
		}
		if err := r.Create(ctx, newCert); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

const (
	defaultAESSecretKey = "AES_SECRET_KEY"
)

func (r *ProjectReconciler) reconcileSecret(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	secretName := "aes-secret-key"
	envPathSecretName := os.Getenv("AES_SECRET_KEY")
	if envPathSecretName != "" {
		secretName = envPathSecretName
	}

	needSecret := v1.Secret{}

	err := r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: req.Name}, &needSecret)
	if err != nil && apierrors.IsNotFound(err) {

		needSecret = v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   req.Name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Finalizers: []string{
					upmiov1alpha2.FinalizerProtect,
				},
			},
		}

		// initialize data map to avoid panic: assignment to entry in nil map
		needSecret.Data = make(map[string][]byte)

		if project.Labels != nil {
			needSecret.Labels = project.Labels
		}
		needSecret.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace
		needSecret.Labels[upmiov1alpha2.LabelNamespace] = req.Name

		if _, ok := project.Annotations[upmiov1alpha2.AnnotationAesSecretKey]; ok {
			needSecret.Data[defaultAESSecretKey] = []byte(project.Annotations[upmiov1alpha2.AnnotationAesSecretKey])
		} else {
			data, err := generateAES256Key()
			if err != nil {
				return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
			}

			needSecret.Data[defaultAESSecretKey] = data

			if project.Annotations == nil {
				project.Annotations = make(map[string]string)
			}

			project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
			updateProjectErr := r.Update(ctx, project)
			if updateProjectErr != nil {
				return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
			}
		}

		err = r.Create(ctx, &needSecret)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("[reconcileSecret] create secret:[%s/%s] error: [%v]", req.Name, secretName, err.Error())
		}

	} else if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("[reconcileSecret] get secret:[%s/%s] error: [%v]", req.Name, secretName, err.Error())
	} else {
		// Secret exists: validate AES key (must be 32-char hex) and self-heal if needed
		current, ok := needSecret.Data[defaultAESSecretKey]
		// no aes secret key
		if !ok {
			// project with annotation
			pCurrent, pOk := project.Annotations[upmiov1alpha2.AnnotationAesSecretKey]
			if pOk {
				if isValidHex32([]byte(pCurrent)) {
					if needSecret.Data == nil {
						needSecret.Data = make(map[string][]byte)
					}

					needSecret.Data[defaultAESSecretKey] = []byte(project.Annotations[upmiov1alpha2.AnnotationAesSecretKey])
					if updErr := r.Update(ctx, &needSecret); updErr != nil {
						return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
					}
				} else {
					data, err := generateAES256Key()
					if err != nil {
						return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
					}

					if needSecret.Data == nil {
						needSecret.Data = make(map[string][]byte)
					}

					needSecret.Data[defaultAESSecretKey] = data
					if updErr := r.Update(ctx, &needSecret); updErr != nil {
						return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
					}

					if project.Annotations == nil {
						project.Annotations = make(map[string]string)
					}

					project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
					updateProjectErr := r.Update(ctx, project)
					if updateProjectErr != nil {
						return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
					}
				}
			} else if !pOk {
				data, err := generateAES256Key()
				if err != nil {
					return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
				}

				if needSecret.Data == nil {
					needSecret.Data = make(map[string][]byte)
				}

				needSecret.Data[defaultAESSecretKey] = data
				if updErr := r.Update(ctx, &needSecret); updErr != nil {
					return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
				}

				if project.Annotations == nil {
					project.Annotations = make(map[string]string)
				}

				project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
				updateProjectErr := r.Update(ctx, project)
				if updateProjectErr != nil {
					return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
				}
			}

		} else if ok && !isValidHex32(current) {
			pCurrent, pOk := project.Annotations[upmiov1alpha2.AnnotationAesSecretKey]
			if pOk {
				if isValidHex32([]byte(pCurrent)) {
					if needSecret.Data == nil {
						needSecret.Data = make(map[string][]byte)
					}

					needSecret.Data[defaultAESSecretKey] = []byte(project.Annotations[upmiov1alpha2.AnnotationAesSecretKey])
					if updErr := r.Update(ctx, &needSecret); updErr != nil {
						return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
					}
				} else {
					data, err := generateAES256Key()
					if err != nil {
						return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
					}

					if needSecret.Data == nil {
						needSecret.Data = make(map[string][]byte)
					}

					needSecret.Data[defaultAESSecretKey] = data
					if updErr := r.Update(ctx, &needSecret); updErr != nil {
						return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
					}

					if project.Annotations == nil {
						project.Annotations = make(map[string]string)
					}

					project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
					updateProjectErr := r.Update(ctx, project)
					if updateProjectErr != nil {
						return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
					}
				}
			} else if !pOk {
				data, err := generateAES256Key()
				if err != nil {
					return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
				}

				if needSecret.Data == nil {
					needSecret.Data = make(map[string][]byte)
				}

				needSecret.Data[defaultAESSecretKey] = data
				if updErr := r.Update(ctx, &needSecret); updErr != nil {
					return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
				}

				if project.Annotations == nil {
					project.Annotations = make(map[string]string)
				}

				project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
				updateProjectErr := r.Update(ctx, project)
				if updateProjectErr != nil {
					return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
				}
			}
		} else {
			if !isValidHex32(current) {
				data, err := generateAES256Key()
				if err != nil {
					return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
				}

				if needSecret.Data == nil {
					needSecret.Data = make(map[string][]byte)
				}

				needSecret.Data[defaultAESSecretKey] = data
				if updErr := r.Update(ctx, &needSecret); updErr != nil {
					return fmt.Errorf("[reconcileSecret] update secret:[%s/%s] error: [%v]", req.Name, secretName, updErr)
				}

				if project.Annotations == nil {
					project.Annotations = make(map[string]string)
				}

				project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(data)
				updateProjectErr := r.Update(ctx, project)
				if updateProjectErr != nil {
					return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
				}
			} else {
				if project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] == "" {
					project.Annotations[upmiov1alpha2.AnnotationAesSecretKey] = string(needSecret.Data[defaultAESSecretKey])
					updateProjectErr := r.Update(ctx, project)
					if updateProjectErr != nil {
						return fmt.Errorf("[reconcileSecret] update project:[%s] error: [%s]", req.Name, updateProjectErr.Error())
					}
				}
			}
		}
	}

	return nil
}

func generateAES256Key() (key []byte, err error) {
	// Generate 16 random bytes and hex-encode to 32 characters
	raw := make([]byte, 16)
	if _, err = io.ReadFull(rand.Reader, raw); err != nil {
		return nil, err
	}
	hexStr := hex.EncodeToString(raw)
	return []byte(hexStr), nil
}

func isValidHex32(b []byte) bool {
	if len(b) != 32 {
		return false
	}
	if _, err := hex.DecodeString(string(b)); err != nil {
		return false
	}
	return true
}
