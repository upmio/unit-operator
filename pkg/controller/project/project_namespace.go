package project

import (
	"context"
	"fmt"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ProjectReconciler) reconcileNamespace(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	namespaceName := req.Name
	ns := v1.Namespace{}

	err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, &ns)
	if apierrors.IsNotFound(err) {
		ns = v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   namespaceName,
				Labels: make(map[string]string),
			},
		}

		if project.Labels != nil {
			ns.Labels = project.Labels
		}
		ns.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace

		err = r.Create(ctx, &ns)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("[reconcileNamespace] create Namespace:[%s] error: [%v]",
				namespaceName, err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileNamespace] get Namespace:[%s] error: [%v]",
			namespaceName, err.Error())
	}

	return nil
}
