package project

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	if apierrors.IsNotFound(err) {

		needSecret = v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   req.Name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
		}

		// initialize data map to avoid panic: assignment to entry in nil map
		needSecret.Data = make(map[string][]byte)

		needSecret.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace
		needSecret.Labels[upmiov1alpha2.LabelNamespace] = req.Name

		data, err := generateAES256Key()
		if err != nil {
			return fmt.Errorf("[reconcileSecret] generateAES256Key error: [%v]", err.Error())
		}

		needSecret.Data[defaultAESSecretKey] = data

		err = r.Create(ctx, &needSecret)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("[reconcileSecret] create secret:[%s/%s] error: [%v]", req.Name, secretName, err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileSecret] get secret:[%s/%s] error: [%v]", req.Name, secretName, err.Error())
	}

	return nil
}

func generateAES256Key() (key []byte, err error) {
	key = make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	return key, nil
}
