package grpccall

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/upmio/compose-operator/pkg/utils"
	upmv1alpha1 "github.com/upmio/unit-operator/api/v1alpha1"
	"k8s.io/klog/v2"
)

func (r *ReconcileGrpcCall) updateInstanceIfNeed(instance *upmv1alpha1.GrpcCall,
	oldStatus *upmv1alpha1.GrpcCallStatus,
	reqLogger logr.Logger) {

	if compareStatus(&instance.Status, oldStatus, reqLogger) {

		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			klog.Errorf("failed to update grpc call [%s] status: %v", instance.Name, err)

		}
	}
}

func compareStatus(new, old *upmv1alpha1.GrpcCallStatus, reqLogger logr.Logger) bool {
	if utils.CompareStringValue("Result", string(old.Result), string(new.Result), reqLogger) {
		//reqLogger.Info(fmt.Sprintf("found status.Result changed: the old one is %s, new one is %s", old.Result, new.Result))
		klog.Infof("found status.Result changed: the old one is %s, new one is %s", old.Result, new.Result)

		return true
	}

	if utils.CompareStringValue("Message", old.Message, new.Message, reqLogger) {
		//reqLogger.Info(fmt.Sprintf("found status.Message changed: the old one is %s, new one is %s", old.Message, new.Message))
		klog.Infof("found status.Message changed: the old one is %s, new one is %s", old.Message, new.Message)
		return true
	}

	if new.StartTime.Time != old.StartTime.Time {
		klog.Infof("found status.StartTime changed: the old one is %s, new one is %s", old.StartTime.Time.String(), new.StartTime.Time.String())
		return true
	}

	if new.CompletionTime.Time != old.CompletionTime.Time {
		klog.Infof("found status.CompletionTime changed: the old one is %s, new one is %s", old.CompletionTime.Time.String(), new.CompletionTime.Time.String())
		return true
	}

	return false
}
