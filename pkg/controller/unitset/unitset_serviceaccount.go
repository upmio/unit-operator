package unitset

import (
	"context"
	"fmt"
	upmiov1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
	"github.com/upmio/unit-operator/pkg/vars"
	v1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *UnitSetReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	saName := fmt.Sprintf("%s-serviceaccount", req.Namespace)
	sa := v1.ServiceAccount{}

	err := r.Get(ctx, client.ObjectKey{Name: saName, Namespace: req.Namespace}, &sa)
	if apierrors.IsNotFound(err) {
		sa = v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: req.Namespace,
				Labels:    make(map[string]string),
			},
		}

		sa.Labels["owner"] = vars.ManagerNamespace

		err = r.Create(ctx, &sa)
		if err != nil {
			return fmt.Errorf("[reconcileServiceAccount] create serviceaccount error: [%v]", err.Error())
		}

	} else if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] get serviceaccount error: [%v]", err.Error())
	}

	//role
	err = r.reconcileRole(ctx, req, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] reconcile role error: [%v]", err.Error())
	}

	//rolebinding
	err = r.reconcileRoleBinding(ctx, req, unitset)
	if err != nil {
		return fmt.Errorf("[reconcileServiceAccount] reconcile rolebinding error: [%v]", err.Error())
	}

	return nil
}

func (r *UnitSetReconciler) reconcileRole(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	roleName := fmt.Sprintf("%s-role", req.Namespace)
	role := rbacV1.Role{}

	err := r.Get(ctx, client.ObjectKey{Name: roleName, Namespace: req.Namespace}, &role)
	if apierrors.IsNotFound(err) {
		role = rbacV1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleName,
				Namespace: req.Namespace,
				Labels:    make(map[string]string),
			},
			Rules: []rbacV1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "secrets"},
					Verbs:     []string{"get", "list"},
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

		role.Labels["owner"] = vars.ManagerNamespace

		err = r.Create(ctx, &role)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *UnitSetReconciler) reconcileRoleBinding(ctx context.Context, req ctrl.Request, unitset *upmiov1alpha2.UnitSet) error {
	rbName := fmt.Sprintf("%s-rolebinding", req.Namespace)
	rb := rbacV1.RoleBinding{}

	err := r.Get(ctx, client.ObjectKey{Name: rbName, Namespace: req.Namespace}, &rb)
	if apierrors.IsNotFound(err) {
		rb = rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rbName,
				Namespace: req.Namespace,
				Labels:    make(map[string]string),
			},
			RoleRef: rbacV1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     fmt.Sprintf("%s-role", req.Namespace),
			},
			Subjects: []rbacV1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      fmt.Sprintf("%s-serviceaccount", req.Namespace),
					Namespace: req.Namespace,
				},
			},
		}

		rb.Labels["owner"] = vars.ManagerNamespace

		err = r.Create(ctx, &rb)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
