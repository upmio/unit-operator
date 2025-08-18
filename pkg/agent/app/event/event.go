package event

import (
	"context"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	unitv1alpha2 "github.com/upmio/unit-operator/api/v1alpha2"
)

type IEventRecorder interface {
	SendNormalEventToUnit(name, namespace, reason, msg string) error
	SendWarningEventToUnit(name, namespace, reason, msg string) error
}

type service struct {
	unitClient client.Client
	record.EventRecorder
	logger *zap.SugaredLogger
}

// Common helper methods

// getUnitInstance gets the unit instance from Kubernetes
func (s *service) getUnitInstance(name, namespace string) (*unitv1alpha2.Unit, error) {
	instance := &unitv1alpha2.Unit{}
	if err := s.unitClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, instance); err != nil {
		return nil, fmt.Errorf("failed to fetch unit[%s] in namespace[%s]: %v", name, namespace, err)
	}
	return instance, nil
}

// sendEventToUnit sends an event to a unit with the specified event type
func (s *service) sendEventToUnit(name, namespace, reason, msg, eventType string) error {
	s.logger.With(
		"name", name,
		"namespace", namespace,
		"message", msg,
		"reason", reason,
		"event_type", eventType,
	).Info("sending event to unit")

	// Get unit instance
	instance, err := s.getUnitInstance(name, namespace)
	if err != nil {
		s.logger.Error(err)
		return err
	}

	// Send event
	s.Event(instance, eventType, reason, msg)
	return nil
}

func (s *service) SendNormalEventToUnit(name, namespace, reason, msg string) error {
	return s.sendEventToUnit(name, namespace, reason, msg, v1.EventTypeNormal)
}

func (s *service) SendWarningEventToUnit(name, namespace, reason, msg string) error {
	return s.sendEventToUnit(name, namespace, reason, msg, v1.EventTypeWarning)
}

func NewIEventRecorder() (IEventRecorder, error) {
	clientSet, err := conf.GetConf().GetClientSet()
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})

	schema, err := unitv1alpha2.SchemeBuilder.Build()
	if err != nil {
		return nil, err
	}

	recorder := eventBroadcaster.NewRecorder(schema, v1.EventSource{Component: "unit-agent"})

	c, err := conf.GetConf().GetUnitClient()
	if err != nil {
		return nil, err
	}

	return &service{
		unitClient:    c,
		EventRecorder: recorder,
		logger:        zap.L().Named("[EVENT]").Sugar(),
	}, nil
}
