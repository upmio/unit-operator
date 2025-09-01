package certs

import (
	"context"

	"github.com/upmio/unit-operator/pkg/vars"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// WebhookSecretName is the name of the secret where the certificates
	// for the webhook server are stored
	WebhookSecretName = "unit-operator-webhook-cert" // #nosec

	// WebhookServiceName is the name of the service where the webhook server
	// is reachable
	WebhookServiceName = "unit-operator-webhook-service" // #nosec

	// MutatingWebhookConfigurationName is the name of the mutating webhook configuration
	MutatingWebhookConfigurationName = "unit-operator-mutating-webhook-configuration"

	// ValidatingWebhookConfigurationName is the name of the validating webhook configuration
	ValidatingWebhookConfigurationName = "unit-operator-validating-webhook-configuration"

	// DefaultWebhookCertDir The name of the directory containing the TLS certificates
	DefaultWebhookCertDir = "/run/secrets/unit-operator/webhook"

	// CaSecretName is the name of the secret which is hosting the Operator CA
	CaSecretName = "unit-operator-ca-secret" // #nosec
)

// EnsurePKI ensures that we have the required PKI infrastructure to make
// the operator and the clusters working
func EnsurePKI(
	ctx context.Context,
	kubeClient client.Client,
	mgrCertDir string,
) error {
	//if conf.WebhookCertDir != "" {
	//	// OLM is generating certificates for us, so we can avoid injecting/creating certificates.
	//	return nil
	//}

	// We need to self-manage required PKI infrastructure and install the certificates into
	// the webhooks configuration
	klog.Infof("ensuring PKI infrastructure")

	pkiConfig := PublicKeyInfrastructure{
		CaSecretName:                       CaSecretName,
		CertDir:                            mgrCertDir,
		SecretName:                         WebhookSecretName,
		ServiceName:                        WebhookServiceName,
		OperatorNamespace:                  vars.ManagerNamespace,
		MutatingWebhookConfigurationName:   MutatingWebhookConfigurationName,
		ValidatingWebhookConfigurationName: ValidatingWebhookConfigurationName,
		OperatorDeploymentLabelSelector:    "app.kubernetes.io/instance=unit-operator",
	}
	err := pkiConfig.Setup(ctx, kubeClient)
	if err != nil {
		klog.Error(err, "unable to setup PKI infrastructure")
	}
	return err
}
