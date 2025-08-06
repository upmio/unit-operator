package certs

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createFakeOperatorDeploymentByName(ctx context.Context,
	kubeClient client.Client,
	deploymentName string,
	labels map[string]string,
) error {
	operatorDep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: operatorNamespaceName,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{},
	}

	return kubeClient.Create(ctx, &operatorDep)
}

func deleteFakeOperatorDeployment(ctx context.Context,
	kubeClient client.Client,
	deploymentName string,
) error {
	operatorDep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: operatorNamespaceName,
		},
		Spec: appsv1.DeploymentSpec{},
	}

	return kubeClient.Delete(ctx, &operatorDep)
}

var _ = Describe("Difference of values of maps", func() {
	It("will always set the app.kubernetes.io/name to unit-operator", func(ctx SpecContext) {
		operatorLabelSelector := "app.kubernetes.io/name=unit-operator"
		operatorLabels := map[string]string{
			"app.kubernetes.io/name": "unit-operator",
		}
		kubeClient := generateFakeClient()
		err := createFakeOperatorDeploymentByName(ctx, kubeClient, operatorDeploymentName, operatorLabels)
		Expect(err).ToNot(HaveOccurred())
		labelMap, err := labels.ConvertSelectorToLabelsMap(operatorLabelSelector)
		Expect(err).ToNot(HaveOccurred())

		deployment, err := findOperatorDeploymentByFilter(ctx,
			kubeClient,
			operatorNamespaceName,
			client.MatchingLabelsSelector{Selector: labelMap.AsSelector()})
		Expect(err).ToNot(HaveOccurred())
		Expect(deployment).ToNot(BeNil())

		err = deleteFakeOperatorDeployment(ctx, kubeClient, operatorDeploymentName)
		Expect(err).ToNot(HaveOccurred())

		operatorLabels = map[string]string{
			"app.kubernetes.io/name": "some-app",
		}
		err = createFakeOperatorDeploymentByName(ctx, kubeClient, "some-app", operatorLabels)
		Expect(err).ToNot(HaveOccurred())
		deployment, err = findOperatorDeploymentByFilter(ctx,
			kubeClient,
			operatorNamespaceName,
			client.MatchingLabelsSelector{Selector: labelMap.AsSelector()})
		Expect(err).To(HaveOccurred())
		Expect(deployment).To(BeNil())

		operatorLabels = map[string]string{
			"app.kubernetes.io/name": "unit-operator",
		}
		err = createFakeOperatorDeploymentByName(ctx, kubeClient, operatorNamespaceName, operatorLabels)
		Expect(err).ToNot(HaveOccurred())
		deployment, err = findOperatorDeploymentByFilter(ctx,
			kubeClient,
			operatorNamespaceName,
			client.MatchingLabelsSelector{Selector: labelMap.AsSelector()})
		Expect(err).ToNot(HaveOccurred())
		Expect(deployment).ToNot(BeNil())
	})
})
