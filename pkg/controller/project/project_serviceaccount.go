package project

import (
	"context"
	"fmt"

	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	rbacV1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ProjectReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	saName := fmt.Sprintf("%s-serviceaccount", req.Name)
	sa := v1.ServiceAccount{}

	err := r.Get(ctx, client.ObjectKey{Name: saName, Namespace: req.Name}, &sa)
	if apierrors.IsNotFound(err) {
		sa = v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:        saName,
				Namespace:   req.Name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
		}

		if project.Labels != nil {
			sa.Labels = project.Labels
		}
		sa.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace

		err = r.Create(ctx, &sa)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("[reconcileServiceAccount] create serviceaccount error: [%v]", err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] get serviceaccount error: [%v]", err.Error())
	}

	//role
	err = r.reconcileRole(ctx, req, project)
	if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] reconcile role error: [%v]", err.Error())
	}

	//rolebinding
	err = r.reconcileRoleBinding(ctx, req, project)
	if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] reconcile rolebinding error: [%v]", err.Error())
	}

	return nil
}

func (r *ProjectReconciler) reconcileRole(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	roleName := fmt.Sprintf("%s-role", req.Name)
	role := rbacV1.Role{}

	err := r.Get(ctx, client.ObjectKey{Name: roleName, Namespace: req.Name}, &role)
	if apierrors.IsNotFound(err) {
		role = rbacV1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:        roleName,
				Namespace:   req.Name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			Rules: []rbacV1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"get", "list", "create", "patch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "list", "patch", "update"},
				},
				{
					APIGroups: []string{"upm.syntropycloud.io"},
					Resources: []string{"redisreplications"},
					Verbs:     []string{"get", "list", "patch", "update"},
				},
				{
					APIGroups: []string{"upm.syntropycloud.io"},
					Resources: []string{"units"},
					Verbs:     []string{"get", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
			},
		}

		if project.Labels != nil {
			role.Labels = project.Labels
		}
		role.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace

		err = r.Create(ctx, &role)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *ProjectReconciler) reconcileRoleBinding(ctx context.Context, req ctrl.Request, project *upmiov1alpha2.Project) error {
	rbName := fmt.Sprintf("%s-rolebinding", req.Name)
	rb := rbacV1.RoleBinding{}

	err := r.Get(ctx, client.ObjectKey{Name: rbName, Namespace: req.Name}, &rb)
	if apierrors.IsNotFound(err) {
		rb = rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        rbName,
				Namespace:   req.Name,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			RoleRef: rbacV1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     fmt.Sprintf("%s-role", req.Name),
			},
			Subjects: []rbacV1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      fmt.Sprintf("%s-serviceaccount", req.Name),
					Namespace: req.Namespace,
				},
			},
		}

		if project.Labels != nil {
			rb.Labels = project.Labels
		}
		rb.Labels[upmiov1alpha2.LabelProjectOwner] = vars.ManagerNamespace
		
		err = r.Create(ctx, &rb)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}

	} else if err != nil {
		return err
	}

	return nil
}
