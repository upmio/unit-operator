/*
 * UPM for Enterprise
 *
 * Copyright (c) 2009-2025 SYNTROPY Pte. Ltd.
 * All rights reserved.
 *
 * This software is the confidential and proprietary information of
 * SYNTROPY Pte. Ltd. ("Confidential Information"). You shall not
 * disclose such Confidential Information and shall use it only in
 * accordance with the terms of the license agreement you entered
 * into with SYNTROPY.
 */

package grpccall

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	appName   = "grpc-call"
	agentName = "unit-agent"
)

// ReconcileGrpcCall reconciles GrpcCall resources.
type ReconcileGrpcCall struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	logger   logr.Logger
}

// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=grpccalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=grpccalls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=grpccalls/finalizers,verbs=update
// +kubebuilder:rbac:groups=upm.syntropycloud.io,resources=units,verbs=get;list;update;patch

func (r *ReconcileGrpcCall) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.logger.WithValues("request.namespace", req.Namespace, "request.name", req.Name)
	//reqLogger.Info("start reconciling grpc call")
	klog.Infof("start reconciling grpc call instance [%s]", req.String())
	startTime := time.Now()

	defer func() {
		klog.Infof("finished reconciliation grpc call instance [%s], duration [%v]", req.String(), time.Since(startTime))

		//reqLogger.Info("finish reconciling grpc call",
		//	"cost.time", time.Since(startTime))
	}()

	// Fetch the GrpcCall instance
	instance := &upmv1alpha1.GrpcCall{}
	if err := r.client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			//reqLogger.Info("can't found grpc call instance")
			klog.Errorf("grpc call instance [%s] not found, probably deleted.", req.String())

			return reconcile.Result{}, nil
		}

		//reqLogger.Info("get grpc call instance failed")
		klog.Errorf("failed to fetch grcp call instance [%s]: [%v]", req.String(), err.Error())

		return reconcile.Result{}, err
	}

	oldStatus := instance.Status.DeepCopy()

	if oldStatus.CompletionTime != nil &&
		instance.Spec.TTLSecondsAfterFinished != nil &&
		time.Since(oldStatus.CompletionTime.Time).Seconds() >= float64(*instance.Spec.TTLSecondsAfterFinished) {
		klog.Infof(
			"grpc call instance [%s] is marked for automatic deletion: completed at [%s], TTL = %d seconds",
			req.String(),
			instance.Status.CompletionTime.Time.Format(time.RFC3339),
			*instance.Spec.TTLSecondsAfterFinished,
		)
		err := r.client.Delete(ctx, instance)

		return reconcile.Result{}, err
	}

	if oldStatus.StartTime != nil {
		klog.Infof("grpc call instance [%s] is already started", req.String())

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Hour,
		}, nil
	} else {
		now := metav1.Now()
		instance.Status.StartTime = &now
	}

	// Pipeline-style error handling
	err := func() error {
		host, port, err := gatherUnitAgentEndpoint(ctx, r.client, instance, reqLogger)
		if err != nil {
			return fmt.Errorf("failed to gather unit agent endpoint: %v", err)
		}

		c, err := newGrpcClient(host, port)
		if err != nil {
			return fmt.Errorf("failed to initialize grpc client: %v", err)
		}
		defer c.Close()

		if err = r.handleGrpcCall(ctx, instance, c); err != nil {
			return err
		} else {
			r.recorder.Event(instance, corev1.EventTypeNormal,
				"OperationSucceeded",
				"process grpc call successfully")
		}
		return nil
	}()

	// Centralized error handling
	if err != nil {
		instance.Status.Result = upmv1alpha1.FailedResult
		instance.Status.Message = err.Error()
		r.recorder.Event(instance, corev1.EventTypeWarning, "OperationFailed", err.Error())
	}

	if instance.Status.CompletionTime == nil {
		now := metav1.Now()
		instance.Status.CompletionTime = &now
	}

	r.updateInstanceIfNeed(instance, oldStatus, reqLogger)

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: time.Hour,
	}, nil
}

func Setup(mgr ctrl.Manager) error {
	r := &ReconcileGrpcCall{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(appName),
		logger:   ctrl.Log.WithName(appName),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&upmv1alpha1.GrpcCall{}).
		Complete(r)
}
